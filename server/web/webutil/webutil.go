package webutil

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

// ExtractClientIP extracts the client IP address from echo.Context, handling reverse proxy headers with fallback to RemoteAddr.
func ExtractClientIP(ctx echo.Context) string {
	id := ctx.RealIP()
	if id == "" {
		host, _, err := net.SplitHostPort(ctx.Request().RemoteAddr)
		if err == nil {
			id = host
		} else {
			id = ctx.Request().RemoteAddr
		}
	}
	return id
}

// NormalizeBasePath ensures the path has a leading slash and trailing slash.
func NormalizeBasePath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "http://") && !strings.HasPrefix(p, "https://") && p[0] != '/' {
		p = "/" + p
	}
	if p[len(p)-1] != '/' {
		p += "/"
	}
	return p
}

// ResolveBasePath returns the effective base path for a request.
// If X-Forwarded-Prefix header is present, it takes precedence.
// Otherwise the configured basePath is used.
func ResolveBasePath(c echo.Context, configuredBasePath string) string {
	if prefix := c.Request().Header.Get("X-Forwarded-Prefix"); prefix != "" {
		return NormalizeBasePath(prefix)
	}
	return configuredBasePath
}

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
