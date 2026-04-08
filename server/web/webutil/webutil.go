package webutil

import (
	"net/http"
	"net/url"
)

// GetRequestURL reconstructs the full URL, handling reverse proxy scenarios
func GetRequestURL(r *http.Request) *url.URL {
	// Start with the scheme (http/https)
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	// Determine the host
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	if host == "" {
		host = r.Header.Get("Host")
	}
	if host == "" {
		host = r.URL.Host
	}

	// Construct the full URL
	fullURL := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   r.URL.Path,
	}

	// Add query parameters
	fullURL.RawQuery = r.URL.RawQuery

	return &fullURL
}
