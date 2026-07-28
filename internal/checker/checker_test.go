package checker

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ilyas-bkgo/devcheck/internal/config"
)

func TestParseDurationOrDefault(t *testing.T) {
	if got := ParseDurationOrDefault("invalid", time.Second); got != time.Second {
		t.Fatalf("got %v", got)
	}
	if got := ParseDurationOrDefault("500ms", time.Second); got != 500*time.Millisecond {
		t.Fatalf("got %v", got)
	}
}

func TestCheckTool(t *testing.T) {
	for _, test := range []struct {
		tool   config.ToolCheck
		passed bool
		detail string
	}{
		{config.ToolCheck{Name: "Echo", Command: "echo", Flag: "hello"}, true, "hello"},
		{config.ToolCheck{Name: "Mismatch", Command: "echo", Flag: "one", Pattern: "two"}, false, "version mismatch"},
		{config.ToolCheck{Name: "Invalid", Command: "echo", Flag: "one", Pattern: "["}, false, "Invalid regex pattern"},
		{config.ToolCheck{Name: "Missing", Command: "not-a-devcheck-test-binary"}, false, "Not installed"},
		{config.ToolCheck{Name: "Fails", Command: "sh", Args: []string{"-c", "exit 1"}}, false, "Command failed"},
	} {
		result := checkTool(test.tool)
		if result.Passed != test.passed || !strings.Contains(result.Detail, test.detail) {
			t.Errorf("%+v", result)
		}
	}
}

func TestCheckEnv(t *testing.T) {
	t.Setenv("DEVCHECK_TEST_EDITOR", "nvim")
	if result := checkEnv(config.EnvCheck{Name: "Editor", Var: "DEVCHECK_TEST_EDITOR", Pattern: "^nvim$"}); !result.Passed {
		t.Errorf("%+v", result)
	}
	if result := checkEnv(config.EnvCheck{Name: "Editor", Var: "DEVCHECK_TEST_EDITOR", Pattern: "["}); result.Passed || result.Detail != "Invalid regex pattern" {
		t.Errorf("%+v", result)
	}
	if result := checkEnv(config.EnvCheck{Name: "Editor", Var: "DEVCHECK_TEST_EDITOR"}); !result.Passed || result.Detail != "Set" {
		t.Errorf("%+v", result)
	}
}

func TestCheckService(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	server.Listener = listener
	server.Start()
	defer server.Close()
	if result := checkService(config.ServiceCheck{Name: "HTTP", URL: server.URL}); !result.Passed {
		t.Errorf("%+v", result)
	}

	_, port, _ := net.SplitHostPort(listener.Addr().String())
	var number int
	_, _ = fmt.Sscanf(port, "%d", &number)
	if result := checkService(config.ServiceCheck{Name: "TCP", Port: number}); !result.Passed {
		t.Errorf("%+v", result)
	}
}

func TestRunAndReport(t *testing.T) {
	results := Run(config.Config{
		Tools: []config.ToolCheck{{Name: "Echo", Command: "echo", Flag: "ok"}},
		Paths: []config.PathCheck{{Name: "Temp", Path: os.TempDir()}},
		Env:   []config.EnvCheck{{Name: "Path", Var: "PATH"}},
	})
	if len(results) != 3 || !NewReport(results).Success {
		t.Fatalf("unexpected results: %+v", results)
	}
}
