// Package checker runs the checks defined in a devcheck configuration.
package checker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ilyas-bkgo/devcheck/internal/config"
)

type Result struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Passed   bool   `json:"passed"`
	Detail   string `json:"detail"`
	Fix      string `json:"fix,omitempty"`
}

type Report struct {
	Success bool     `json:"success"`
	Results []Result `json:"results"`
}

func NewReport(results []Result) Report {
	report := Report{Success: true, Results: results}
	for _, result := range results {
		if !result.Passed {
			report.Success = false
			break
		}
	}
	return report
}

func Failed(results []Result) []Result {
	failed := make([]Result, 0)
	for _, result := range results {
		if !result.Passed {
			failed = append(failed, result)
		}
	}
	return failed
}

func ParseDurationOrDefault(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func checkTool(tool config.ToolCheck) Result {
	args := tool.Args
	if len(args) == 0 {
		flagToUse := tool.Flag
		if flagToUse == "" {
			flagToUse = "--version"
		}
		args = []string{flagToUse}
	}
	result := Result{Name: tool.Name, Category: "tool", Fix: tool.Fix.Resolve()}
	if _, err := exec.LookPath(tool.Command); err != nil {
		result.Detail = "Not installed"
		return result
	}

	timeout := ParseDurationOrDefault(tool.Timeout, 3*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, tool.Command, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		result.Detail = fmt.Sprintf("Timed out (%v)", timeout)
		return result
	}

	output := string(out)
	if err != nil {
		result.Detail = "Command failed"
		if lines := strings.Split(output, "\n"); len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			result.Detail = fmt.Sprintf("Command failed: %s", strings.TrimSpace(lines[0]))
		}
		return result
	}
	version := "Version unknown"
	if lines := strings.Split(output, "\n"); len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		version = strings.TrimSpace(lines[0])
	}
	if tool.Pattern != "" {
		re, err := regexp.Compile(tool.Pattern)
		if err != nil {
			result.Detail = "Invalid regex pattern"
			return result
		}
		if !re.MatchString(output) {
			if version != "Version unknown" {
				result.Detail = fmt.Sprintf("%s (version mismatch)", version)
			} else {
				result.Detail = "Version mismatch"
			}
			return result
		}
	}
	result.Passed = true
	result.Detail = version
	return result
}

func checkPath(path config.PathCheck) Result {
	result := Result{Name: path.Name, Category: "path", Fix: path.Fix.Resolve()}
	info, err := os.Stat(config.ExpandPath(path.Path))
	if os.IsNotExist(err) {
		result.Detail = fmt.Sprintf("Missing (%s)", path.Path)
	} else if err != nil {
		result.Detail = "Error reading path"
	} else if info.IsDir() {
		result.Passed, result.Detail = true, "Directory found"
	} else {
		result.Passed, result.Detail = true, "File found"
	}
	return result
}

func checkEnv(env config.EnvCheck) Result {
	result := Result{Name: env.Name, Category: "env", Fix: env.Fix.Resolve()}
	value, exists := os.LookupEnv(env.Var)
	if !exists || value == "" {
		result.Detail = fmt.Sprintf("Missing ($%s)", env.Var)
		return result
	}
	if env.Pattern != "" {
		re, err := regexp.Compile(env.Pattern)
		if err != nil {
			result.Detail = "Invalid regex pattern"
			return result
		}
		if !re.MatchString(value) {
			result.Detail = "Value does not match pattern"
			return result
		}
	}
	result.Passed = true
	result.Detail = "Set"
	return result
}

func checkService(service config.ServiceCheck) Result {
	result := Result{Name: service.Name, Category: "service", Fix: service.Fix.Resolve()}
	timeout := ParseDurationOrDefault(service.Timeout, 3*time.Second)
	if service.Port > 0 {
		host := service.Host
		if host == "" {
			host = "127.0.0.1"
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(service.Port)), timeout)
		if err != nil {
			result.Detail = fmt.Sprintf("Port %d unreachable", service.Port)
			return result
		}
		_ = conn.Close()
		result.Passed = true
		result.Detail = fmt.Sprintf("Port %d reachable (%s)", service.Port, host)
		return result
	}
	if service.URL == "" {
		result.Detail = "No target port or URL specified"
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, service.URL, nil)
	if err != nil {
		result.Detail = "Invalid HTTP request"
		return result
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Detail = "Connection failed / timed out"
		return result
	}
	defer resp.Body.Close()
	expected := service.ExpectedStatus
	if expected == 0 {
		expected = http.StatusOK
	}
	if resp.StatusCode != expected {
		result.Detail = fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, expected)
		return result
	}
	result.Passed = true
	result.Detail = fmt.Sprintf("HTTP %d OK", resp.StatusCode)
	return result
}

// Run executes every configured check concurrently and keeps config order.
func Run(cfg config.Config) []Result {
	results := make([]Result, 0, len(cfg.Tools)+len(cfg.Paths)+len(cfg.Env)+len(cfg.Services))
	tools := make([]Result, len(cfg.Tools))
	paths := make([]Result, len(cfg.Paths))
	envs := make([]Result, len(cfg.Env))
	services := make([]Result, len(cfg.Services))
	var wg sync.WaitGroup
	for i, check := range cfg.Tools {
		wg.Add(1)
		go func(i int, check config.ToolCheck) { defer wg.Done(); tools[i] = checkTool(check) }(i, check)
	}
	for i, check := range cfg.Paths {
		wg.Add(1)
		go func(i int, check config.PathCheck) { defer wg.Done(); paths[i] = checkPath(check) }(i, check)
	}
	for i, check := range cfg.Env {
		wg.Add(1)
		go func(i int, check config.EnvCheck) { defer wg.Done(); envs[i] = checkEnv(check) }(i, check)
	}
	for i, check := range cfg.Services {
		wg.Add(1)
		go func(i int, check config.ServiceCheck) { defer wg.Done(); services[i] = checkService(check) }(i, check)
	}
	wg.Wait()
	results = append(results, tools...)
	results = append(results, paths...)
	results = append(results, envs...)
	return append(results, services...)
}
