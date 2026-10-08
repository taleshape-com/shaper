// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ResolveHealthcheckURL determines the target healthcheck URL based on
// explicit URL, address, or TLS domain settings.
func ResolveHealthcheckURL(rawURL, addr, tlsDomain string) (string, error) {
	if raw := strings.TrimSpace(rawURL); raw != "" {
		if strings.Contains(raw, "://") {
			if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
				return "", fmt.Errorf("invalid URL %q: scheme must be http or https", rawURL)
			}
		} else {
			raw = "http://" + raw
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("invalid URL %q: missing host", rawURL)
		}
		if parsed.Path == "" || parsed.Path == "/" {
			parsed.Path = "/health"
		}
		return parsed.String(), nil
	}

	if domain := strings.TrimSpace(tlsDomain); domain != "" {
		return "https://" + domain + "/health", nil
	}

	internalAddr := strings.TrimSpace(addr)
	if internalAddr == "" {
		internalAddr = "localhost:5454"
	}

	if strings.HasPrefix(internalAddr, "0.0.0.0") {
		internalAddr = strings.Replace(internalAddr, "0.0.0.0", "localhost", 1)
	} else if strings.HasPrefix(internalAddr, "[::]") {
		internalAddr = strings.Replace(internalAddr, "[::]", "localhost", 1)
	} else if strings.HasPrefix(internalAddr, ":") {
		internalAddr = "localhost" + internalAddr
	}

	if strings.Contains(internalAddr, "://") {
		if !strings.HasPrefix(internalAddr, "http://") && !strings.HasPrefix(internalAddr, "https://") {
			return "", fmt.Errorf("invalid address %q: scheme must be http or https", addr)
		}
	} else {
		internalAddr = "http://" + internalAddr
	}

	parsed, err := url.Parse(internalAddr)
	if err != nil {
		return "", fmt.Errorf("invalid address %q: %w", addr, err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid address %q: missing host", addr)
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/health"
	}
	return parsed.String(), nil
}

// RunHealthcheckCommand executes a health check against a Shaper server.
func RunHealthcheckCommand(ctx context.Context, rawURL, addr, tlsDomain string, timeout time.Duration, quiet bool) error {
	targetURL, err := ResolveHealthcheckURL(rawURL, addr, tlsDomain)
	if err != nil {
		return err
	}

	reqCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create healthcheck request: %w", err)
	}

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("healthcheck failed with HTTP status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if !quiet {
		fmt.Println("OK")
	}

	return nil
}
