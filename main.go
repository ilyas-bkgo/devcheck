package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfig = `# devcheck configuration
tools:
  - name: Git
    cmd: git
    flag: --version
  - name: Neovim
    cmd: nvim
    flag: --version
  - name: Docker
    cmd: docker
    flag: --version

paths:
  - name: SSH Key Dir
    path: ~/.ssh
`

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
	Name    string `yaml:"name"`
	Command string `yaml:"cmd"`
	Flag    string `yaml:"flag"`
}

type PathCheck struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type Config struct {
	Tools []ToolCheck `yaml:"tools"`
	Paths []PathCheck `yaml:"paths"`
}

type CheckResult struct {
	Name     string `json:"name"`
	Category string `json:"category"` // "tool" or "path"
	Passed   bool   `json:"passed"`
	Detail   string `json:"detail"`
}

type Report struct {
	Success bool          `json:"success"`
	Results []CheckResult `json:"results"`
}

// Handles bare "~" as well as "~/"
func expandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func resolveConfigPath(customPath string) string {
	if customPath != "devcheck.yaml" {
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

func main() {

	if len(os.Args) > 1 && os.Args[1] == "init" {
		initConfigFile()
		return
	}

	var configPath string
	var jsonOutput bool

	flag.StringVar(&configPath, "config", "devcheck.yaml", "Path to config file")
	flag.StringVar(&configPath, "c", "devcheck.yaml", "Path to config file (shorthand)")
	flag.BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
	flag.BoolVar(&jsonOutput, "j", false, "Output results in JSON format (shorthand)")
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

	var report Report
	report.Success = true

	// --- 1. Tools Check ---
	for _, tool := range cfg.Tools {
		flagToUse := tool.Flag
		if flagToUse == "" {
			flagToUse = "--version"
		}

		res := CheckResult{Name: tool.Name, Category: "tool"}

		_, err := exec.LookPath(tool.Command)
		if err != nil {
			res.Passed = false
			res.Detail = "Not installed"
			report.Success = false
		} else {
			out, err := exec.Command(tool.Command, flagToUse).CombinedOutput()
			version := "Version unknown"
			if err == nil {
				lines := strings.Split(string(out), "\n")
				if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
					version = strings.TrimSpace(lines[0])
				}
			}
			res.Passed = true
			res.Detail = version
		}
		report.Results = append(report.Results, res)
	}

	// --- 2. Paths Check ---
	for _, p := range cfg.Paths {
		resolved := expandPath(p.Path)
		res := CheckResult{Name: p.Name, Category: "path"}

		info, err := os.Stat(resolved)
		if os.IsNotExist(err) {
			res.Passed = false
			res.Detail = fmt.Sprintf("Missing (%s)", p.Path)
			report.Success = false
		} else if err != nil {
			res.Passed = false
			res.Detail = "Error reading path"
			report.Success = false
		} else {
			res.Passed = true
			if info.IsDir() {
				res.Detail = "Directory found"
			} else {
				res.Detail = "File found"
			}
		}
		report.Results = append(report.Results, res)
	}

	// --- 3. Output Handling ---
	if jsonOutput {
		outBytes, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(outBytes))
	} else {
		fmt.Printf("🔍 Running devcheck (%s)...\n\n", resolvedConfig)

		var toolResults, pathResults []CheckResult
		for _, r := range report.Results {
			if r.Category == "tool" {
				toolResults = append(toolResults, r)
			} else {
				pathResults = append(pathResults, r)
			}
		}

		if len(toolResults) > 0 {
			fmt.Println("🛠️  CLI Tools:")
			for _, r := range toolResults {
				if r.Passed {
					fmt.Printf(" \033[32m✔\033[0m %-16s %s\n", r.Name, r.Detail)
				} else {
					fmt.Printf(" \033[31m✘\033[0m %-16s %s\n", r.Name, r.Detail)
				}
			}
		}

		if len(pathResults) > 0 {
			if len(toolResults) > 0 {
				fmt.Println()
			}
			fmt.Println("📁 Paths & Configurations:")
			for _, r := range pathResults {
				if r.Passed {
					fmt.Printf(" \033[32m✔\033[0m %-16s %s\n", r.Name, r.Detail)
				} else {
					fmt.Printf(" \033[31m✘\033[0m %-16s %s\n", r.Name, r.Detail)
				}
			}
		}

		if !report.Success {
			fmt.Println("\n❌ System health check failed.")
		} else {
			fmt.Println("\n✨ All checks passed!")
		}
	}

	if !report.Success {
		os.Exit(1)
	}
}
