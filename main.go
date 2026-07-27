package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

const defaultConfig = `# devcheck configuration
tools:
  - name: Go Compiler
    cmd: go
    flag: version
    pattern: 'go1\.(2[2-9]|[3-9][0-9])'
    timeout: 2s
    fix:
      linux_apt: "sudo apt install -y golang"
      linux_dnf: "sudo dnf install -y golang"
      linux_pacman: "sudo pacman -S --noconfirm go"
      darwin: "brew install go"

  - name: Git
    cmd: git
    flag: --version

  - name: Neovim
    cmd: nvim
    flag: --version
    fix:
      linux_apt: "sudo apt install -y neovim"
      linux_dnf: "sudo dnf install -y neovim"
      linux_pacman: "sudo pacman -S --noconfirm neovim"
      darwin: "brew install neovim"

  - name: Docker
    cmd: docker
    flag: --version
    timeout: 5s

paths:
  - name: SSH Key Dir
    path: ~/.ssh
    fix: "mkdir -p ~/.ssh && chmod 700 ~/.ssh"

env:
  - name: Default Editor
    var: EDITOR
    fix: "echo 'export EDITOR=nvim' >> ~/.bashrc"

services:
  - name: PostgreSQL Database
    host: 127.0.0.1
    port: 5432
    timeout: 2s

  - name: Local Web API
    url: http://localhost:8080/health
    expected_status: 200
    timeout: 3s
`

type FixSpec struct {
	Cmd string            `yaml:"-"`
	OS  map[string]string `yaml:"-"`
}

func (f *FixSpec) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		f.Cmd = value.Value
		return nil
	}
	if value.Kind == yaml.MappingNode {
		f.OS = make(map[string]string)
		return value.Decode(&f.OS)
	}
	return nil
}

func (f FixSpec) Resolve() string {
	if f.Cmd != "" {
		return f.Cmd
	}
	if len(f.OS) == 0 {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		if cmd, ok := f.OS["darwin"]; ok {
			return cmd
		}
	case "linux":
		if _, err := exec.LookPath("apt-get"); err == nil {
			if cmd, ok := f.OS["linux_apt"]; ok {
				return cmd
			}
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			if cmd, ok := f.OS["linux_dnf"]; ok {
				return cmd
			}
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			if cmd, ok := f.OS["linux_pacman"]; ok {
				return cmd
			}
		}
		if _, err := exec.LookPath("zypper"); err == nil {
			if cmd, ok := f.OS["linux_zypper"]; ok {
				return cmd
			}
		}
	}

	if cmd, ok := f.OS["default"]; ok {
		return cmd
	}
	return ""
}

func initConfigFile() {
	target := "devcheck.yaml"
	if _, err := os.Stat(target); err == nil {
		fmt.Printf("%s already exists!\n", target)
		return
	}

	err := os.WriteFile(target, []byte(defaultConfig), 0644)
	if err != nil {
		fmt.Printf("Failed to create %s: %v\n", target, err)
		os.Exit(1)
	}

	fmt.Printf("Created starter config at ./%s! Run 'devcheck' to test it. \n", target)
}

type ToolCheck struct {
	Name    string  `yaml:"name"`
	Command string  `yaml:"cmd"`
	Flag    string  `yaml:"flag"`
	Pattern string  `yaml:"pattern,omitempty"`
	Timeout string  `yaml:"timeout,omitempty"`
	Fix     FixSpec `yaml:"fix,omitempty"`
}

type PathCheck struct {
	Name string  `yaml:"name"`
	Path string  `yaml:"path"`
	Fix  FixSpec `yaml:"fix,omitempty"`
}

type EnvCheck struct {
	Name    string  `yaml:"name"`
	Var     string  `yaml:"var"`
	Pattern string  `yaml:"pattern,omitempty"`
	Fix     FixSpec `yaml:"fix,omitempty"`
}

type ServiceCheck struct {
	Name           string  `yaml:"name"`
	Host           string  `yaml:"host,omitempty"`
	Port           int     `yaml:"port,omitempty"`
	URL            string  `yaml:"url,omitempty"`
	ExpectedStatus int     `yaml:"expected_status,omitempty"`
	Timeout        string  `yaml:"timeout,omitempty"`
	Fix            FixSpec `yaml:"fix,omitempty"`
}

type Config struct {
	Tools    []ToolCheck    `yaml:"tools"`
	Paths    []PathCheck    `yaml:"paths"`
	Env      []EnvCheck     `yaml:"env"`
	Services []ServiceCheck `yaml:"services,omitempty"`
}

type CheckResult struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Passed   bool   `json:"passed"`
	Detail   string `json:"detail"`
	Fix      string `json:"fix,omitempty"`
}

type Report struct {
	Success bool          `json:"success"`
	Results []CheckResult `json:"results"`
}

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func expandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	} else if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func resolveConfigPath(customPath string) string {
	if customPath != "devcheck.yaml" && customPath != "" {
		return expandPath(customPath)
	}

	if _, err := os.Stat("devcheck.yaml"); err == nil {
		return "devcheck.yaml"
	}

	home, err := os.UserHomeDir()
	if err == nil {
		globalConfig := filepath.Join(home, ".config", "devcheck", "config.yaml")
		if _, err := os.Stat(globalConfig); err == nil {
			return globalConfig
		}
	}

	return "devcheck.yaml"
}

func parseDurationOrDefault(dStr string, defaultDur time.Duration) time.Duration {
	if dStr == "" {
		return defaultDur
	}
	d, err := time.ParseDuration(dStr)
	if err != nil {
		return defaultDur
	}
	return d
}

func checkTool(tool ToolCheck) CheckResult {
	flagToUse := tool.Flag
	if flagToUse == "" {
		flagToUse = "--version"
	}

	resolvedFix := tool.Fix.Resolve()
	res := CheckResult{Name: tool.Name, Category: "tool", Fix: resolvedFix}

	_, err := exec.LookPath(tool.Command)
	if err != nil {
		res.Passed = false
		res.Detail = "Not installed"
		return res
	}

	timeout := parseDurationOrDefault(tool.Timeout, 3*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, tool.Command, flagToUse).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		res.Passed = false
		res.Detail = fmt.Sprintf("Timed out (%v)", timeout)
		return res
	}

	outputStr := string(out)
	version := "Version unknown"
	if err == nil {
		lines := strings.Split(outputStr, "\n")
		if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			version = strings.TrimSpace(lines[0])
		}
	}

	if tool.Pattern != "" {
		re, err := regexp.Compile(tool.Pattern)
		if err != nil {
			res.Passed = false
			res.Detail = "Invalid regex pattern"
			return res
		}

		if !re.MatchString(outputStr) {
			res.Passed = false
			if version != "Version unknown" {
				res.Detail = fmt.Sprintf("%s (version mismatch)", version)
			} else {
				res.Detail = "Version mismatch"
			}
			return res
		}
	}

	res.Passed = true
	res.Detail = version
	return res
}

func checkPath(p PathCheck) CheckResult {
	resolvedPath := expandPath(p.Path)
	resolvedFix := p.Fix.Resolve()
	res := CheckResult{Name: p.Name, Category: "path", Fix: resolvedFix}

	info, err := os.Stat(resolvedPath)
	if os.IsNotExist(err) {
		res.Passed = false
		res.Detail = fmt.Sprintf("Missing (%s)", p.Path)
	} else if err != nil {
		res.Passed = false
		res.Detail = "Error reading path"
	} else {
		res.Passed = true
		if info.IsDir() {
			res.Detail = "Directory found"
		} else {
			res.Detail = "File found"
		}
	}
	return res
}

func checkEnv(e EnvCheck) CheckResult {
	resolvedFix := e.Fix.Resolve()
	res := CheckResult{Name: e.Name, Category: "env", Fix: resolvedFix}

	val, exists := os.LookupEnv(e.Var)
	if !exists || val == "" {
		res.Passed = false
		res.Detail = fmt.Sprintf("Missing ($%s)", e.Var)
		return res
	}

	if e.Pattern != "" {
		re, err := regexp.Compile(e.Pattern)
		if err != nil {
			res.Passed = false
			res.Detail = "Invalid regex pattern"
			return res
		}

		if !re.MatchString(val) {
			res.Passed = false
			res.Detail = fmt.Sprintf("%s (value mismatch)", val)
			return res
		}
	}

	res.Passed = true
	res.Detail = fmt.Sprintf("Set (%s)", val)
	return res
}

func checkService(s ServiceCheck) CheckResult {
	resolvedFix := s.Fix.Resolve()
	res := CheckResult{Name: s.Name, Category: "service", Fix: resolvedFix}
	timeout := parseDurationOrDefault(s.Timeout, 3*time.Second)

	if s.Port > 0 {
		host := s.Host
		if host == "" {
			host = "127.0.0.1"
		}
		addr := net.JoinHostPort(host, strconv.Itoa(s.Port))
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			res.Passed = false
			res.Detail = fmt.Sprintf("Port %d unreachable", s.Port)
			return res
		}
		conn.Close()
		res.Passed = true
		res.Detail = fmt.Sprintf("Port %d reachable (%s)", s.Port, host)
		return res
	}

	if s.URL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
		if err != nil {
			res.Passed = false
			res.Detail = "Invalid HTTP request"
			return res
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			res.Passed = false
			res.Detail = "Connection failed / timed out"
			return res
		}
		defer resp.Body.Close()

		expected := s.ExpectedStatus
		if expected == 0 {
			expected = http.StatusOK
		}

		if resp.StatusCode != expected {
			res.Passed = false
			res.Detail = fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, expected)
			return res
		}

		res.Passed = true
		res.Detail = fmt.Sprintf("HTTP %d OK", resp.StatusCode)
		return res
	}

	res.Passed = false
	res.Detail = "No target port or URL specified"
	return res
}

func runChecks(cfg *Config) ([]CheckResult, []CheckResult, []CheckResult, []CheckResult) {
	var wg sync.WaitGroup
	toolResults := make([]CheckResult, len(cfg.Tools))
	pathResults := make([]CheckResult, len(cfg.Paths))
	envResults := make([]CheckResult, len(cfg.Env))
	serviceResults := make([]CheckResult, len(cfg.Services))

	for i, tool := range cfg.Tools {
		wg.Add(1)
		go func(idx int, t ToolCheck) {
			defer wg.Done()
			toolResults[idx] = checkTool(t)
		}(i, tool)
	}

	for i, p := range cfg.Paths {
		wg.Add(1)
		go func(idx int, p PathCheck) {
			defer wg.Done()
			pathResults[idx] = checkPath(p)
		}(i, p)
	}

	for i, e := range cfg.Env {
		wg.Add(1)
		go func(idx int, e EnvCheck) {
			defer wg.Done()
			envResults[idx] = checkEnv(e)
		}(i, e)
	}

	for i, s := range cfg.Services {
		wg.Add(1)
		go func(idx int, s ServiceCheck) {
			defer wg.Done()
			serviceResults[idx] = checkService(s)
		}(i, s)
	}

	wg.Wait()
	return toolResults, pathResults, envResults, serviceResults
}

func executeFix(fixCmd string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", fixCmd)
	} else {
		cmd = exec.Command("sh", "-c", fixCmd)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseFixSelections(input string, maxLen int) []int {
	var selected []int
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			if idx >= 1 && idx <= maxLen {
				selected = append(selected, idx-1)
			}
		}
	}
	return selected
}

func applyFixes(failed []CheckResult) {
	var fixable []CheckResult
	for _, res := range failed {
		if strings.TrimSpace(res.Fix) != "" {
			fixable = append(fixable, res)
		}
	}

	if len(fixable) == 0 {
		fmt.Println("\n🔧 Auto-Fix Mode: No executable fix commands available for failed checks.")
		return
	}

	fmt.Println("\n🔧 Running Auto-Fix Mode...")
	fmt.Println("Available fixes for failed checks:")
	for i, res := range fixable {
		fmt.Printf("  [%d] %-10s %-18s -> %s\n", i+1, "["+res.Category+"]", res.Name, res.Fix)
	}

	fmt.Println("\nSelect options:")
	fmt.Println("  a) Fix ALL")
	fmt.Println("  s) Select specific items")
	fmt.Println("  q) Quit / Skip")
	fmt.Print("\nChoice [a/s/q]: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.ToLower(strings.TrimSpace(choice))

	var toFix []CheckResult

	switch choice {
	case "a", "all":
		toFix = fixable
	case "s", "select":
		fmt.Print("Enter item numbers to fix (comma-separated, e.g., 1, 2): ")
		input, _ := reader.ReadString('\n')
		indices := parseFixSelections(input, len(fixable))
		for _, idx := range indices {
			toFix = append(toFix, fixable[idx])
		}
	case "q", "quit", "":
		fmt.Println("Auto-fix skipped.")
		return
	default:
		fmt.Println("Invalid choice. Skipping auto-fix.")
		return
	}

	if len(toFix) == 0 {
		fmt.Println("No valid items selected. Skipping auto-fix.")
		return
	}

	fmt.Println()
	for _, res := range toFix {
		fmt.Printf("Executing fix for [%s] %s: %s\n", res.Category, res.Name, res.Fix)
		if err := executeFix(res.Fix); err != nil {
			fmt.Printf("❌ Fix failed: %v\n", err)
		} else {
			fmt.Println("✔ Fix executed successfully.")
		}
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		initConfigFile()
		return
	}

	var configPath string
	var jsonOutput bool
	var quietOutput bool
	var fixMode bool

	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flag.BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
	flag.BoolVar(&jsonOutput, "j", false, "Output results in JSON format (shorthand)")
	flag.BoolVar(&quietOutput, "quiet", false, "Only print failed checks")
	flag.BoolVar(&quietOutput, "q", false, "Only print failed checks (shorthand)")
	flag.BoolVar(&fixMode, "fix", false, "Prompt and execute defined fixes for failed checks")
	flag.BoolVar(&fixMode, "f", false, "Prompt and execute defined fixes for failed checks (shorthand)")
	flag.Parse()

	resolvedConfig := resolveConfigPath(configPath)
	data, err := os.ReadFile(resolvedConfig)
	if err != nil {
		if jsonOutput {
			errReport, _ := json.Marshal(Report{Success: false})
			fmt.Println(string(errReport))
		} else {
			fmt.Printf("❌ Could not read %s: %v\n", resolvedConfig, err)
		}
		os.Exit(1)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		if jsonOutput {
			errReport, _ := json.Marshal(Report{Success: false})
			fmt.Println(string(errReport))
		} else {
			fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		}
		os.Exit(1)
	}

	toolResults, pathResults, envResults, serviceResults := runChecks(&cfg)

	var report Report
	report.Success = true
	report.Results = append(report.Results, toolResults...)
	report.Results = append(report.Results, pathResults...)
	report.Results = append(report.Results, envResults...)
	report.Results = append(report.Results, serviceResults...)

	var failedChecks []CheckResult

	for _, r := range report.Results {
		if !r.Passed {
			report.Success = false
			failedChecks = append(failedChecks, r)
		}
	}

	if jsonOutput {
		outBytes, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(outBytes))
	} else {
		passSym := "\033[32m✔\033[0m"
		failSym := "\033[31m✘\033[0m"
		if !colorEnabled() {
			passSym = "✔"
			failSym = "✘"
		}

		if quietOutput {
			if !report.Success {
				var failedTools, failedPaths, failedEnvs, failedServices []CheckResult
				for _, r := range failedChecks {
					switch r.Category {
					case "tool":
						failedTools = append(failedTools, r)
					case "path":
						failedPaths = append(failedPaths, r)
					case "env":
						failedEnvs = append(failedEnvs, r)
					case "service":
						failedServices = append(failedServices, r)
					}
				}

				if len(failedTools) > 0 {
					fmt.Println("🛠️  CLI Tools:")
					for _, r := range failedTools {
						fmt.Printf(" %s %-16s %s\n", failSym, r.Name, r.Detail)
					}
				}

				if len(failedPaths) > 0 {
					if len(failedTools) > 0 {
						fmt.Println()
					}
					fmt.Println("📁 Paths & Configurations:")
					for _, r := range failedPaths {
						fmt.Printf(" %s %-16s %s\n", failSym, r.Name, r.Detail)
					}
				}

				if len(failedEnvs) > 0 {
					if len(failedTools) > 0 || len(failedPaths) > 0 {
						fmt.Println()
					}
					fmt.Println("🌐 Environment Variables:")
					for _, r := range failedEnvs {
						fmt.Printf(" %s %-16s %s\n", failSym, r.Name, r.Detail)
					}
				}

				if len(failedServices) > 0 {
					if len(failedTools) > 0 || len(failedPaths) > 0 || len(failedEnvs) > 0 {
						fmt.Println()
					}
					fmt.Println("🌐 Network Services:")
					for _, r := range failedServices {
						fmt.Printf(" %s %-16s %s\n", failSym, r.Name, r.Detail)
					}
				}

				fmt.Println("\n❌ System health check failed.")
			}
		} else {
			fmt.Printf("🔍 Running devcheck (%s)...\n\n", resolvedConfig)

			if len(toolResults) > 0 {
				fmt.Println("🛠️  CLI Tools:")
				for _, r := range toolResults {
					sym := passSym
					if !r.Passed {
						sym = failSym
					}
					fmt.Printf(" %s %-16s %s\n", sym, r.Name, r.Detail)
				}
			}

			if len(pathResults) > 0 {
				if len(toolResults) > 0 {
					fmt.Println()
				}
				fmt.Println("📁 Paths & Configurations:")
				for _, r := range pathResults {
					sym := passSym
					if !r.Passed {
						sym = failSym
					}
					fmt.Printf(" %s %-16s %s\n", sym, r.Name, r.Detail)
				}
			}

			if len(envResults) > 0 {
				if len(toolResults) > 0 || len(pathResults) > 0 {
					fmt.Println()
				}
				fmt.Println("🌐 Environment Variables:")
				for _, r := range envResults {
					sym := passSym
					if !r.Passed {
						sym = failSym
					}
					fmt.Printf(" %s %-16s %s\n", sym, r.Name, r.Detail)
				}
			}

			if len(serviceResults) > 0 {
				if len(toolResults) > 0 || len(pathResults) > 0 || len(envResults) > 0 {
					fmt.Println()
				}
				fmt.Println("🌐 Network Services:")
				for _, r := range serviceResults {
					sym := passSym
					if !r.Passed {
						sym = failSym
					}
					fmt.Printf(" %s %-16s %s\n", sym, r.Name, r.Detail)
				}
			}

			if !report.Success {
				fmt.Println("\n❌ System health check failed.")
			} else {
				fmt.Println("\n✨ All checks passed!")
			}
		}
	}

	if fixMode && len(failedChecks) > 0 {
		applyFixes(failedChecks)
	}

	if !report.Success {
		os.Exit(1)
	}
}
