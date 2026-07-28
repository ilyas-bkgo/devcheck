package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	fetchTimeout = 10 * time.Second
)

const maxResponseSize = 1 * 1024 * 1024 // 1MB

// fetchRemoteYAML fetches a YAML configuration from a remote URL.
// TODO: No caching for now — always fetch fresh. Consider adding an --offline or local-cache flag in the future.
func fetchRemoteYAML(url string) ([]byte, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("invalid scheme: only http/https supported for %q", url)
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch include %q: %w", url, err)
	}

	// Prevent leaking credentials or sensitive headers by using a clean client.
	// Enforce that redirects only follow http/https schemes.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			scheme := req.URL.Scheme
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("disallowed redirect scheme %q", scheme)
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch include %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch include %q: status code %d", url, resp.StatusCode)
	}

	// Limit response size to prevent OOM
	reader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("fetch include %q: %w", url, err)
	}

	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("fetch include %q: response exceeds maximum size of 1MB", url)
	}

	return body, nil
}
