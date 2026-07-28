package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := ExpandPath("~"); got != home {
		t.Fatalf("ExpandPath(~) = %q, want %q", got, home)
	}
	if got := ExpandPath("~/foo/bar"); got != filepath.Join(home, "foo/bar") {
		t.Fatalf("unexpected expanded path: %q", got)
	}
	if got := ExpandPath("relative/path"); got != "relative/path" {
		t.Fatalf("relative path changed: %q", got)
	}
}

func TestStarterTemplates(t *testing.T) {
	for _, name := range TemplateNames() {
		content, err := StarterConfig(name)
		if err != nil {
			t.Fatalf("StarterConfig(%q): %v", name, err)
		}
		var cfg Config
		if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
			t.Fatalf("StarterConfig(%q) returned invalid YAML: %v", name, err)
		}
	}
	if _, err := StarterConfig("unknown"); !errors.Is(err, ErrUnknownTemplate) {
		t.Fatalf("StarterConfig(unknown) error = %v", err)
	}
}

func TestResolvePath(t *testing.T) {
	mockHome := t.TempDir()
	t.Setenv("HOME", mockHome)
	if got := ResolvePath("custom/config.yaml"); got != "custom/config.yaml" {
		t.Fatalf("custom path = %q", got)
	}

	workDir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	if got := ResolvePath(""); got != "devcheck.yaml" {
		t.Fatalf("fallback = %q", got)
	}
	if err := os.WriteFile("devcheck.yaml", nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got := ResolvePath(""); got != "devcheck.yaml" {
		t.Fatalf("local config = %q", got)
	}
}

func TestFixSpecResolution(t *testing.T) {
	if got := (FixSpec{Cmd: "echo hello"}).Resolve(); got != "echo hello" {
		t.Fatalf("command = %q", got)
	}
	if got := (FixSpec{OS: map[string]string{"default": "echo default"}}).Resolve(); got != "echo default" {
		t.Fatalf("default = %q", got)
	}
}

func TestLoadProfileAndIncludes(t *testing.T) {
	directory := t.TempDir()
	basePath := filepath.Join(directory, "base.yaml")
	projectPath := filepath.Join(directory, "devcheck.yaml")
	if err := os.WriteFile(basePath, []byte("tools:\n  - name: Git\n    cmd: git\n"), 0644); err != nil {
		t.Fatal(err)
	}
	project := "include:\n  - base.yaml\nprofiles:\n  ci:\n    env:\n      - name: CI\n        var: CI\n"
	if err := os.WriteFile(projectPath, []byte(project), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadProfile(projectPath, "ci")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tools) != 1 || len(cfg.Env) != 1 {
		t.Fatalf("unexpected merged config: %+v", cfg)
	}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnsafeDefinitions(t *testing.T) {
	err := Validate(Config{Tools: []ToolCheck{{Name: "Tool", Command: "echo", Flag: "ok", Args: []string{"ok"}}}})
	if err == nil || !strings.Contains(err.Error(), "both flag and args") {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(Config{Env: []EnvCheck{{Name: "Token", Var: "TOKEN", Pattern: "["}}})
	if err == nil || !strings.Contains(err.Error(), "invalid pattern") {
		t.Fatalf("unexpected error: %v", err)
	}
}
