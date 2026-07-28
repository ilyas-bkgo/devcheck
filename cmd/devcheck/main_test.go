package main

import (
	"os"
	"testing"

	"github.com/ilyas-bkgo/devcheck/internal/config"
)

func TestRunInitCreatesTemplate(t *testing.T) {
	workingDirectory := t.TempDir()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDirectory) })
	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}

	if exitCode := runInit([]string{"--template", "node"}); exitCode != 0 {
		t.Fatalf("runInit exit code = %d", exitCode)
	}
	cfg, err := config.Load("devcheck.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tools) != 3 || len(cfg.Services) != 1 {
		t.Fatalf("unexpected node template: %+v", cfg)
	}
}

func TestRunInitRejectsUnknownTemplate(t *testing.T) {
	if exitCode := runInit([]string{"--template", "unknown"}); exitCode != 1 {
		t.Fatalf("runInit exit code = %d, want 1", exitCode)
	}
}
