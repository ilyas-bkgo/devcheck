package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadRemoteConfig(t *testing.T) {
	// Create a temp local file for the merged content
	tmpFile, err := os.CreateTemp("", "local.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("tools:\n  - name: Local Tool\n    cmd: local"))
	tmpFile.Close()

	// Setup mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/remote.yaml" {
			w.Write([]byte("tools:\n  - name: Remote Tool\n    cmd: remote"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	// Create main config pointing to both
	mainConfig := filepath.Join(t.TempDir(), "main.yaml")
	err = os.WriteFile(mainConfig, []byte("include:\n  - "+tmpFile.Name()+"\n  - "+ts.URL+"/remote.yaml"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(mainConfig)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(cfg.Tools))
	}
}

func TestRemoteFetchErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	mainConfig := filepath.Join(t.TempDir(), "main.yaml")
	os.WriteFile(mainConfig, []byte("include:\n  - "+ts.URL+"/404"), 0644)

	_, err := Load(mainConfig)
	if err == nil {
		t.Error("Expected error for 404, got nil")
	}
}

func TestRemoteFetchTimeout(t *testing.T) {
	orig := fetchTimeout
	fetchTimeout = 50 * time.Millisecond
	defer func() { fetchTimeout = orig }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("tools: []"))
	}))
	defer ts.Close()

	_, err := fetchRemoteYAML(ts.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestRemoteFetchMaxSize(t *testing.T) {
	largeData := make([]byte, maxResponseSize+100)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(largeData)
	}))
	defer ts.Close()

	_, err := fetchRemoteYAML(ts.URL)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("Expected max size error, got: %v", err)
	}
}

func TestRemoteCycleDetection(t *testing.T) {
	var ts *httptest.Server
	requests := make(map[string]int)
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		if r.URL.Path == "/a.yaml" {
			fmt.Fprintf(w, "include:\n  - %s/b.yaml", ts.URL)
		} else {
			fmt.Fprintf(w, "include:\n  - %s/a.yaml", ts.URL)
		}
	}))
	defer ts.Close()

	_, err := Load(ts.URL + "/a.yaml")
	if err == nil || !strings.Contains(err.Error(), "configuration include cycle") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}

	if requests["/a.yaml"] != 1 {
		t.Errorf("Expected exactly 1 request to /a.yaml, got %d. If > 1, cycle detection ran after fetching.", requests["/a.yaml"])
	}
	if requests["/b.yaml"] != 1 {
		t.Errorf("Expected exactly 1 request to /b.yaml, got %d", requests["/b.yaml"])
	}
}

func TestRemoteRelativeIncludeRejection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("include:\n  - ./some-local-file.yaml"))
	}))
	defer ts.Close()

	_, err := Load(ts.URL + "/remote.yaml")
	if err == nil {
		t.Error("Expected error for relative include inside remote config, got nil")
	} else if !strings.Contains(err.Error(), "relative include") {
		t.Errorf("Expected relative include rejection error, got: %v", err)
	}
}

func TestRemoteDisallowedRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "ftp://malicious.com/config.yaml")
		w.WriteHeader(http.StatusFound)
	}))
	defer ts.Close()

	_, err := fetchRemoteYAML(ts.URL)
	if err == nil {
		t.Error("Expected redirect error for ftp scheme, got nil")
	} else if !strings.Contains(err.Error(), "disallowed redirect scheme") {
		t.Errorf("Expected disallowed redirect scheme error, got: %v", err)
	}
}
