package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "bare tilde",
			path: "~",
			want: home,
		},
		{
			name: "tilde with subpath",
			path: "~/foo/bar",
			want: filepath.Join(home, "foo/bar"),
		},
		{
			name: "absolute path unchanged",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path unchanged",
			path: "relative/path/file.txt",
			want: "relative/path/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.path)
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveConfigPath(t *testing.T) {
	mockHome := t.TempDir()
	t.Setenv("HOME", mockHome)

	t.Run("Custom path passed explicitly", func(t *testing.T) {
		got := resolveConfigPath("custom/config.yaml")
		want := "custom/config.yaml"
		if got != want {
			t.Errorf("resolveConfigPath() = %q, want %q", got, want)
		}
	})

	t.Run("Local devcheck.yaml exists", func(t *testing.T) {
		workDir := t.TempDir()
		localFile := filepath.Join(workDir, "devcheck.yaml")
		if err := os.WriteFile(localFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		_ = os.Chdir(workDir)

		got := resolveConfigPath("devcheck.yaml")
		if got != "devcheck.yaml" {
			t.Errorf("resolveConfigPath() = %q, want devcheck.yaml", got)
		}
	})

	t.Run("Global ~/.config/devcheck/config.yaml exists", func(t *testing.T) {
		workDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		_ = os.Chdir(workDir)

		globalDir := filepath.Join(mockHome, ".config", "devcheck")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}
		globalFile := filepath.Join(globalDir, "config.yaml")
		if err := os.WriteFile(globalFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		got := resolveConfigPath("devcheck.yaml")
		if got != globalFile {
			t.Errorf("resolveConfigPath() = %q, want %q", got, globalFile)
		}
	})

	t.Run("Default fallback when neither exists", func(t *testing.T) {
		emptyHome := t.TempDir()
		t.Setenv("HOME", emptyHome)

		workDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		_ = os.Chdir(workDir)

		got := resolveConfigPath("devcheck.yaml")
		if got != "devcheck.yaml" {
			t.Errorf("resolveConfigPath() = %q, want devcheck.yaml", got)
		}
	})
}

func TestParseDurationOrDefault(t *testing.T) {
	defaultDur := 3 * time.Second

	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"empty string falls back to default", "", defaultDur},
		{"valid duration string in seconds", "5s", 5 * time.Second},
		{"valid duration string in milliseconds", "500ms", 500 * time.Millisecond},
		{"invalid duration falls back to default", "not-a-duration", defaultDur},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDurationOrDefault(tt.input, defaultDur)
			if got != tt.expected {
				t.Errorf("parseDurationOrDefault(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFixSpecResolution(t *testing.T) {
	t.Run("Scalar String Command", func(t *testing.T) {
		fix := FixSpec{Cmd: "echo hello"}
		if got := fix.Resolve(); got != "echo hello" {
			t.Errorf("Resolve() = %q, want 'echo hello'", got)
		}
	})

	t.Run("OS Map with Default Fallback", func(t *testing.T) {
		fix := FixSpec{OS: map[string]string{"default": "echo default"}}
		if got := fix.Resolve(); got != "echo default" {
			t.Errorf("Resolve() = %q, want 'echo default'", got)
		}
	})
}

func TestCheckTool(t *testing.T) {
	tests := []struct {
		name       string
		tool       ToolCheck
		wantPassed bool
		wantDetail string
	}{
		{
			name: "existing command without pattern",
			tool: ToolCheck{
				Name:    "Echo",
				Command: "echo",
				Flag:    "hello",
			},
			wantPassed: true,
			wantDetail: "hello",
		},
		{
			name: "existing command with matching pattern",
			tool: ToolCheck{
				Name:    "Echo",
				Command: "echo",
				Flag:    "hello world",
				Pattern: "world",
			},
			wantPassed: true,
			wantDetail: "hello world",
		},
		{
			name: "existing command with custom timeout",
			tool: ToolCheck{
				Name:    "Sleep Timeout",
				Command: "sleep",
				Flag:    "2",
				Timeout: "100ms",
			},
			wantPassed: false,
			wantDetail: "Timed out",
		},
		{
			name: "existing command with failing pattern",
			tool: ToolCheck{
				Name:    "Echo",
				Command: "echo",
				Flag:    "version 1.0",
				Pattern: "^version 2\\.",
			},
			wantPassed: false,
			wantDetail: "version 1.0 (version mismatch)",
		},
		{
			name: "existing command with invalid regex pattern",
			tool: ToolCheck{
				Name:    "Echo",
				Command: "echo",
				Flag:    "hello",
				Pattern: "[unclosed-group",
			},
			wantPassed: false,
			wantDetail: "Invalid regex pattern",
		},
		{
			name: "non-existent binary",
			tool: ToolCheck{
				Name:    "MissingBinary",
				Command: "definitely-not-a-real-binary-xyz",
				Flag:    "--version",
			},
			wantPassed: false,
			wantDetail: "Not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := checkTool(tt.tool)
			if res.Passed != tt.wantPassed {
				t.Errorf("checkTool() Passed = %v, want %v", res.Passed, tt.wantPassed)
			}
			if tt.wantDetail != "" && !strings.Contains(res.Detail, tt.wantDetail) {
				t.Errorf("checkTool() Detail = %q, want to contain %q", res.Detail, tt.wantDetail)
			}
		})
	}
}

func TestCheckEnv(t *testing.T) {
	t.Setenv("TEST_DEVCHECK_EDITOR", "nvim")

	tests := []struct {
		name       string
		env        EnvCheck
		wantPassed bool
		wantDetail string
	}{
		{
			name: "existing env variable without pattern",
			env: EnvCheck{
				Name: "Editor",
				Var:  "TEST_DEVCHECK_EDITOR",
			},
			wantPassed: true,
			wantDetail: "Set (nvim)",
		},
		{
			name: "existing env variable with matching pattern",
			env: EnvCheck{
				Name:    "Editor",
				Var:     "TEST_DEVCHECK_EDITOR",
				Pattern: "^nvim$",
			},
			wantPassed: true,
			wantDetail: "Set (nvim)",
		},
		{
			name: "existing env variable with failing pattern",
			env: EnvCheck{
				Name:    "Editor",
				Var:     "TEST_DEVCHECK_EDITOR",
				Pattern: "^vim$",
			},
			wantPassed: false,
			wantDetail: "nvim (value mismatch)",
		},
		{
			name: "missing env variable",
			env: EnvCheck{
				Name: "MissingVar",
				Var:  "DEFINITELY_NOT_SET_VAR_XYZ",
			},
			wantPassed: false,
			wantDetail: "Missing ($DEFINITELY_NOT_SET_VAR_XYZ)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := checkEnv(tt.env)
			if res.Passed != tt.wantPassed {
				t.Errorf("checkEnv() Passed = %v, want %v", res.Passed, tt.wantPassed)
			}
			if tt.wantDetail != "" && !strings.Contains(res.Detail, tt.wantDetail) {
				t.Errorf("checkEnv() Detail = %q, want to contain %q", res.Detail, tt.wantDetail)
			}
		})
	}
}

func TestCheckService(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	_, portStr, _ := net.SplitHostPort(listener.Addr().String())
	var openPort int
	fmt.Sscanf(portStr, "%d", &openPort)

	tests := []struct {
		name       string
		service    ServiceCheck
		wantPassed bool
		wantDetail string
	}{
		{
			name: "TCP port reachable",
			service: ServiceCheck{
				Name: "Local TCP Server",
				Host: "127.0.0.1",
				Port: openPort,
			},
			wantPassed: true,
			wantDetail: "reachable",
		},
		{
			name: "TCP port unreachable",
			service: ServiceCheck{
				Name:    "Closed Port",
				Host:    "127.0.0.1",
				Port:    59999,
				Timeout: "100ms",
			},
			wantPassed: false,
			wantDetail: "unreachable",
		},
		{
			name: "HTTP endpoint status 200 OK",
			service: ServiceCheck{
				Name:           "HTTP Health Endpoint",
				URL:            ts.URL + "/health",
				ExpectedStatus: 200,
			},
			wantPassed: true,
			wantDetail: "HTTP 200 OK",
		},
		{
			name: "HTTP endpoint status mismatch",
			service: ServiceCheck{
				Name:           "HTTP Missing Endpoint",
				URL:            ts.URL + "/not-found",
				ExpectedStatus: 200,
			},
			wantPassed: false,
			wantDetail: "HTTP 404 (expected 200)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := checkService(tt.service)
			if res.Passed != tt.wantPassed {
				t.Errorf("checkService() Passed = %v, want %v", res.Passed, tt.wantPassed)
			}
			if tt.wantDetail != "" && !strings.Contains(res.Detail, tt.wantDetail) {
				t.Errorf("checkService() Detail = %q, want to contain %q", res.Detail, tt.wantDetail)
			}
		})
	}
}

func TestParseFixSelections(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected []int
	}{
		{
			name:     "single valid choice",
			input:    "1",
			maxLen:   3,
			expected: []int{0},
		},
		{
			name:     "multiple comma-separated choices",
			input:    "1, 3",
			maxLen:   3,
			expected: []int{0, 2},
		},
		{
			name:     "out-of-bounds indices ignored",
			input:    "0, 1, 4",
			maxLen:   3,
			expected: []int{0},
		},
		{
			name:     "invalid non-numeric parts ignored",
			input:    "1, foo, 2",
			maxLen:   3,
			expected: []int{0, 1},
		},
		{
			name:     "whitespace handled correctly",
			input:    " 2 , 1 ",
			maxLen:   3,
			expected: []int{1, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFixSelections(tt.input, tt.maxLen)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseFixSelections(%q, %d) returned %d items, want %d", tt.input, tt.maxLen, len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseFixSelections(%q, %d)[%d] = %d, want %d", tt.input, tt.maxLen, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRunChecks(t *testing.T) {
	cfg := &Config{
		Tools: []ToolCheck{
			{Name: "Echo", Command: "echo", Flag: "tool"},
		},
		Paths: []PathCheck{
			{Name: "TmpDir", Path: os.TempDir()},
		},
		Env: []EnvCheck{
			{Name: "PathEnv", Var: "PATH"},
		},
		Services: []ServiceCheck{
			{Name: "Invalid TCP", Host: "127.0.0.1", Port: 59998, Timeout: "50ms"},
		},
	}

	toolRes, pathRes, envRes, serviceRes := runChecks(cfg)

	if len(toolRes) != 1 || !toolRes[0].Passed {
		t.Errorf("Expected 1 passing tool result, got %v", toolRes)
	}
	if len(pathRes) != 1 || !pathRes[0].Passed {
		t.Errorf("Expected 1 passing path result, got %v", pathRes)
	}
	if len(envRes) != 1 || !envRes[0].Passed {
		t.Errorf("Expected 1 passing env result, got %v", envRes)
	}
	if len(serviceRes) != 1 || serviceRes[0].Passed {
		t.Errorf("Expected 1 failing service result, got %v", serviceRes)
	}
}
