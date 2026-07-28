// Package config defines and loads devcheck configuration files.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	// ErrConfigExists is returned when an initialization target already exists.
	ErrConfigExists = errors.New("configuration already exists")
	// ErrUnknownTemplate is returned for an unsupported starter template.
	ErrUnknownTemplate = errors.New("unknown configuration template")
)

const DefaultConfig = `# devcheck configuration
tools:
  - name: Go Compiler
    cmd: go
    flag: version
    pattern: 'go1\.25\.'
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

const nodeTemplate = `# Node.js web development environment
tools:
  - name: Node.js
    cmd: node
    flag: --version

  - name: pnpm
    cmd: pnpm
    flag: --version

  - name: Docker
    cmd: docker
    flag: --version

services:
  - name: PostgreSQL
    host: 127.0.0.1
    port: 5432
    timeout: 2s
`

const pythonTemplate = `# Python backend development environment
tools:
  - name: Python
    cmd: python3
    flag: --version

  - name: Poetry
    cmd: poetry
    flag: --version

  - name: Redis
    cmd: redis-server
    flag: --version

services:
  - name: Local API
    url: http://127.0.0.1:8000/health
    expected_status: 200
    timeout: 3s
`

const goTemplate = `# Go microservices development environment
tools:
  - name: Go
    cmd: go
    flag: version

  - name: Protocol Buffers compiler
    cmd: protoc
    flag: --version

  - name: golangci-lint
    cmd: golangci-lint
    flag: --version

  - name: Docker
    cmd: docker
    flag: --version

services:
  - name: Local API
    url: http://127.0.0.1:8080/health
    expected_status: 200
    timeout: 3s
`

var starterTemplates = map[string]string{
	"default": DefaultConfig,
	"node":    nodeTemplate,
	"python":  pythonTemplate,
	"go":      goTemplate,
}

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

// Resolve returns the applicable fix command for the running platform.
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
		for _, manager := range []struct{ binary, key string }{
			{"apt-get", "linux_apt"},
			{"dnf", "linux_dnf"},
			{"pacman", "linux_pacman"},
			{"zypper", "linux_zypper"},
		} {
			if _, err := exec.LookPath(manager.binary); err == nil {
				if cmd, ok := f.OS[manager.key]; ok {
					return cmd
				}
			}
		}
	}

	return f.OS["default"]
}

type ToolCheck struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"cmd"`
	Flag    string   `yaml:"flag"`
	Args    []string `yaml:"args,omitempty"`
	Pattern string   `yaml:"pattern,omitempty"`
	Timeout string   `yaml:"timeout,omitempty"`
	Fix     FixSpec  `yaml:"fix,omitempty"`
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
	Include  []string           `yaml:"include,omitempty"`
	Tools    []ToolCheck        `yaml:"tools"`
	Paths    []PathCheck        `yaml:"paths"`
	Env      []EnvCheck         `yaml:"env"`
	Services []ServiceCheck     `yaml:"services,omitempty"`
	Profiles map[string]Profile `yaml:"profiles,omitempty"`
}

// Profile adds checks to the base configuration selected by --profile.
type Profile struct {
	Tools    []ToolCheck    `yaml:"tools"`
	Paths    []PathCheck    `yaml:"paths"`
	Env      []EnvCheck     `yaml:"env"`
	Services []ServiceCheck `yaml:"services,omitempty"`
}

func ExpandPath(path string) string {
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

// ResolvePath prioritizes an explicit path, then local and global defaults.
func ResolvePath(customPath string) string {
	if customPath != "devcheck.yaml" && customPath != "" {
		return ExpandPath(customPath)
	}
	if _, err := os.Stat("devcheck.yaml"); err == nil {
		return "devcheck.yaml"
	}
	if home, err := os.UserHomeDir(); err == nil {
		globalConfig := filepath.Join(home, ".config", "devcheck", "config.yaml")
		if _, err := os.Stat(globalConfig); err == nil {
			return globalConfig
		}
	}
	return "devcheck.yaml"
}

func Load(path string) (Config, error) {
	return LoadProfile(path, "")
}

// LoadProfile loads a strict YAML config, its relative includes, and an optional profile.
func LoadProfile(path, profile string) (Config, error) {
	return load(path, profile, make(map[string]bool))
}

func load(path, profile string, visiting map[string]bool) (Config, error) {
	absPath, err := filepath.Abs(ExpandPath(path))
	if err != nil {
		return Config{}, err
	}
	if visiting[absPath] {
		return Config{}, fmt.Errorf("configuration include cycle: %s", absPath)
	}
	visiting[absPath] = true
	defer delete(visiting, absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return Config{}, err
	}
	var current Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&current); err != nil {
		return Config{}, err
	}

	merged := Config{}
	for _, include := range current.Include {
		includePath := ExpandPath(include)
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(filepath.Dir(absPath), includePath)
		}
		included, err := load(includePath, "", visiting)
		if err != nil {
			return Config{}, fmt.Errorf("include %q: %w", include, err)
		}
		merged = merge(merged, included)
	}
	current.Include = nil
	selected, ok := current.Profiles[profile]
	if profile != "" && !ok {
		return Config{}, fmt.Errorf("unknown profile %q", profile)
	}
	current.Profiles = nil
	merged = merge(merged, current)
	if profile != "" {
		merged = merge(merged, Config{Tools: selected.Tools, Paths: selected.Paths, Env: selected.Env, Services: selected.Services})
	}
	return merged, nil
}

func merge(base, extra Config) Config {
	base.Tools = append(base.Tools, extra.Tools...)
	base.Paths = append(base.Paths, extra.Paths...)
	base.Env = append(base.Env, extra.Env...)
	base.Services = append(base.Services, extra.Services...)
	return base
}

// Validate rejects malformed or ambiguous check definitions before execution.
func Validate(cfg Config) error {
	if len(cfg.Tools)+len(cfg.Paths)+len(cfg.Env)+len(cfg.Services) == 0 {
		return errors.New("configuration contains no checks")
	}
	names := make(map[string]struct{})
	uniqueName := func(category, name string) error {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%s check has an empty name", category)
		}
		if _, exists := names[name]; exists {
			return fmt.Errorf("duplicate check name %q", name)
		}
		names[name] = struct{}{}
		return nil
	}
	validatePattern := func(name, pattern string) error {
		if pattern == "" {
			return nil
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("%s has an invalid pattern: %w", name, err)
		}
		return nil
	}
	validateTimeout := func(name, timeout string) error {
		if timeout == "" {
			return nil
		}
		if _, err := time.ParseDuration(timeout); err != nil {
			return fmt.Errorf("%s has an invalid timeout: %w", name, err)
		}
		return nil
	}
	for _, check := range cfg.Tools {
		if err := uniqueName("tool", check.Name); err != nil {
			return err
		}
		if strings.TrimSpace(check.Command) == "" {
			return fmt.Errorf("tool %q has an empty cmd", check.Name)
		}
		if check.Flag != "" && len(check.Args) > 0 {
			return fmt.Errorf("tool %q cannot define both flag and args", check.Name)
		}
		if err := validatePattern("tool "+check.Name, check.Pattern); err != nil {
			return err
		}
		if err := validateTimeout("tool "+check.Name, check.Timeout); err != nil {
			return err
		}
	}
	for _, check := range cfg.Paths {
		if err := uniqueName("path", check.Name); err != nil {
			return err
		}
		if strings.TrimSpace(check.Path) == "" {
			return fmt.Errorf("path %q has an empty path", check.Name)
		}
	}
	for _, check := range cfg.Env {
		if err := uniqueName("env", check.Name); err != nil {
			return err
		}
		if strings.TrimSpace(check.Var) == "" {
			return fmt.Errorf("environment check %q has an empty var", check.Name)
		}
		if err := validatePattern("environment check "+check.Name, check.Pattern); err != nil {
			return err
		}
	}
	for _, check := range cfg.Services {
		if err := uniqueName("service", check.Name); err != nil {
			return err
		}
		if (check.Port > 0) == (check.URL != "") {
			return fmt.Errorf("service %q must define exactly one of port or url", check.Name)
		}
		if check.Port < 0 || check.Port > 65535 {
			return fmt.Errorf("service %q has an invalid port", check.Name)
		}
		if err := validateTimeout("service "+check.Name, check.Timeout); err != nil {
			return err
		}
	}
	return nil
}

// TemplateNames returns the supported starter-template names.
func TemplateNames() []string {
	names := make([]string, 0, len(starterTemplates))
	for name := range starterTemplates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// StarterConfig returns the YAML content for a named starter template.
func StarterConfig(template string) (string, error) {
	content, ok := starterTemplates[template]
	if !ok {
		return "", ErrUnknownTemplate
	}
	return content, nil
}

// WriteStarter creates a config file from the default template unless it exists.
func WriteStarter(path string) error {
	return WriteStarterTemplate(path, "default")
}

// WriteStarterTemplate creates a config file from a named template unless it exists.
func WriteStarterTemplate(path, template string) error {
	content, err := StarterConfig(template)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return ErrConfigExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
