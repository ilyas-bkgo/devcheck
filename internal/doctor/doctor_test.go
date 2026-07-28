package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithValidConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devcheck.yaml")
	if err := os.WriteFile(path, []byte("tools:\n  - name: Go\n    cmd: go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if !Run(&output, path, "") {
		t.Fatal("Run returned false for a valid configuration")
	}
	if !strings.Contains(output.String(), "Configuration is valid (1 checks)") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestRunWithMissingConfiguration(t *testing.T) {
	var output bytes.Buffer
	if Run(&output, filepath.Join(t.TempDir(), "missing.yaml"), "") {
		t.Fatal("Run returned true for a missing configuration")
	}
	if !strings.Contains(output.String(), "Configuration is not usable") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}
