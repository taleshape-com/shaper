// SPDX-License-Identifier: MPL-2.0

package webutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/shaper", "/shaper/"},
		{"/shaper/", "/shaper/"},
		{"shaper", "/shaper/"},
		{"shaper/", "/shaper/"},
		{"/apps/shaper", "/apps/shaper/"},
		{"/apps/shaper/", "/apps/shaper/"},
		{"http://example.com", "http://example.com/"},
		{"http://example.com/", "http://example.com/"},
		{"https://example.com/shaper", "https://example.com/shaper/"},
		{"https://example.com/shaper/", "https://example.com/shaper/"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeBasePath(tt.input))
		})
	}
}

func TestResolveBasePath(t *testing.T) {
	e := echo.New()

	t.Run("uses configured basepath when no header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "/", ResolveBasePath(c, "/"))
		assert.Equal(t, "/shaper/", ResolveBasePath(c, "/shaper/"))
	})

	t.Run("header overrides configured basepath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Prefix", "/dynamic/")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "/dynamic/", ResolveBasePath(c, "/"))
		assert.Equal(t, "/dynamic/", ResolveBasePath(c, "/static/"))
	})

	t.Run("header value is normalized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Prefix", "no-leading-slash")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "/no-leading-slash/", ResolveBasePath(c, "/"))
	})

	t.Run("header without trailing slash is normalized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Prefix", "/prefix")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "/prefix/", ResolveBasePath(c, "/"))
	})
}

func TestExtractClientIP(t *testing.T) {
	t.Run("direct connection without headers", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "198.51.100.5:12345"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "198.51.100.5", ExtractClientIP(c))
	})

	t.Run("direct untrusted connection ignores spoofed X-Forwarded-For", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "198.51.100.5:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Attacker directly connecting cannot spoof their IP
		assert.Equal(t, "198.51.100.5", ExtractClientIP(c))
	})

	t.Run("trusted reverse proxy extracts real client IP from X-Forwarded-For", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:54321" // loopback proxy
		req.Header.Set("X-Forwarded-For", "198.51.100.5")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "198.51.100.5", ExtractClientIP(c))
	})

	t.Run("trusted reverse proxy ignores client-spoofed prefix in X-Forwarded-For", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:54321" // loopback proxy
		req.Header.Set("X-Forwarded-For", "203.0.113.99, 198.51.100.5")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Nearest untrusted IP to proxy is 198.51.100.5
		assert.Equal(t, "198.51.100.5", ExtractClientIP(c))
	})

	t.Run("works with echo instance having IPExtractor configured", func(t *testing.T) {
		e := echo.New()
		e.IPExtractor = echo.ExtractIPFromXFFHeader()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "198.51.100.5:12345"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Equal(t, "198.51.100.5", ExtractClientIP(c))
	})
}
