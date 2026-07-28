package config

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	fetchTimeout = 10 * time.Second
	// bypassSSRF is for testing against localhost
	bypassSSRF = false
)

const maxResponseSize = 1 * 1024 * 1024 // 1MB

// isPrivateIP checks if an IP belongs to private or loopback ranges.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	privateRanges := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7",
	}
	for _, r := range privateRanges {
		_, cidr, _ := net.ParseCIDR(r)
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// DialContext with SSRF protection.
func ssrfProtectedDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	if !bypassSSRF {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}

		ips, err := net.LookupIP(host)
		if err == nil {
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return nil, fmt.Errorf("SSRF protection: access to private IP %s blocked", ip)
				}
			}
		}
	}

	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

// fetchRemoteYAML fetches a YAML configuration from a remote URL.
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

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: ssrfProtectedDialer,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			scheme := req.URL.Scheme
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("disallowed redirect scheme %q", scheme)
			}

			if !bypassSSRF {
				host := req.URL.Hostname()
				ips, err := net.LookupIP(host)
				if err == nil {
					for _, ip := range ips {
						if isPrivateIP(ip) {
							return fmt.Errorf("SSRF protection: redirect to private IP %s blocked", ip)
						}
					}
				}
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
