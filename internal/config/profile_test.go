package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileAndRemoteInclude(t *testing.T) {
	bypassSSRF = true
	defer func() { bypassSSRF = false }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
profiles:
  test:
    tools:
      - name: Remote Tool
        cmd: echo
        flag: remote
`))
	}))
	defer ts.Close()

	mainConfig := filepath.Join(t.TempDir(), "main.yaml")
	os.WriteFile(mainConfig, []byte("tools:\n  - name: Local Tool\n    cmd: echo\ninclude:\n  - "+ts.URL+"\nprofiles:\n  test:\n    tools:\n      - name: Remote Tool\n        cmd: echo\n        flag: remote"), 0644)

	cfg, err := LoadProfile(mainConfig, "test")
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	if len(cfg.Tools) != 2 {
		t.Errorf("Expected 2 tools (1 local, 1 remote), got %d (Tools: %+v)", len(cfg.Tools), cfg.Tools)
	}
}
