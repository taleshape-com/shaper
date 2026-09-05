// SPDX-License-Identifier: MPL-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
)

func TestParseCORSDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string defaults to wildcard",
			input:    "",
			expected: []string{"*"},
		},
		{
			name:     "whitespace only defaults to wildcard",
			input:    "   ",
			expected: []string{"*"},
		},
		{
			name:     "single wildcard",
			input:    "*",
			expected: []string{"*"},
		},
		{
			name:     "wildcard in list returns wildcard",
			input:    "example.com, *, test.org",
			expected: []string{"*"},
		},
		{
			name:     "single domain without scheme expands to https and http",
			input:    "example.com",
			expected: []string{"https://example.com", "http://example.com"},
		},
		{
			name:     "domain with trailing slash",
			input:    "example.com/",
			expected: []string{"https://example.com", "http://example.com"},
		},
		{
			name:     "origin with scheme preserved as is",
			input:    "https://example.com",
			expected: []string{"https://example.com"},
		},
		{
			name:     "origin with http scheme preserved as is",
			input:    "http://localhost:3000",
			expected: []string{"http://localhost:3000"},
		},
		{
			name:     "origin with scheme and trailing slash",
			input:    "https://example.com/",
			expected: []string{"https://example.com"},
		},
		{
			name:     "multiple comma separated domains and origins",
			input:    "https://app.example.com, localhost:5173, http://localhost:3000",
			expected: []string{"https://app.example.com", "https://localhost:5173", "http://localhost:5173", "http://localhost:3000"},
		},
		{
			name:     "wildcard subdomain without scheme",
			input:    "*.example.com",
			expected: []string{"https://*.example.com", "http://*.example.com"},
		},
		{
			name:     "wildcard subdomain with scheme",
			input:    "https://*.example.com",
			expected: []string{"https://*.example.com"},
		},
		{
			name:     "leading dot expands to apex and wildcard",
			input:    ".example.com",
			expected: []string{"https://example.com", "http://example.com", "https://*.example.com", "http://*.example.com"},
		},
		{
			name:     "protocol relative url",
			input:    "//example.com",
			expected: []string{"https://example.com", "http://example.com"},
		},
		{
			name:     "deduplication",
			input:    "example.com, https://example.com, example.com",
			expected: []string{"https://example.com", "http://example.com"},
		},
		{
			name:     "empty entries ignored",
			input:    ", example.com, , test.com, ",
			expected: []string{"https://example.com", "http://example.com", "https://test.com", "http://test.com"},
		},
		{
			name:     "domain with port without scheme",
			input:    "example.com:8443",
			expected: []string{"https://example.com:8443", "http://example.com:8443"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCORSDomains(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCORSMiddlewareIntegration(t *testing.T) {
	setupEcho := func(corsDomains string) *echo.Echo {
		e := echo.New()
		allowOrigins := ParseCORSDomains(corsDomains)
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: allowOrigins,
			AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		}))
		e.GET("/api/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
		return e
	}

	t.Run("default unrestricted CORS allows any origin", func(t *testing.T) {
		e := setupEcho("")

		// Preflight OPTIONS request
		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://anywhere.org")
		req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "*", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Actual GET request
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://anywhere.org")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "*", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	})

	t.Run("restricted CORS allows configured domains and blocks others", func(t *testing.T) {
		e := setupEcho("example.com, https://app.trusted.io, *.sub.org")

		// Allowed origin: https://example.com (expanded from example.com)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://example.com")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "https://example.com", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Allowed origin: http://example.com (expanded from example.com)
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "http://example.com")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "http://example.com", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Allowed origin: https://app.trusted.io
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://app.trusted.io")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "https://app.trusted.io", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Allowed origin: https://foo.sub.org (matched by *.sub.org)
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://foo.sub.org")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "https://foo.sub.org", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Preflight for allowed origin
		req = httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://example.com")
		req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "https://example.com", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
		assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlAllowMethods), http.MethodPost)

		// Disallowed origin: http://app.trusted.io (only https was specified)
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "http://app.trusted.io")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Disallowed origin: https://attacker.com
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://attacker.com")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Preflight for disallowed origin: does NOT return Access-Control-Allow-Origin
		req = httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://attacker.com")
		req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))

		// Disallowed origin: https://attacker-sub.org (not a subdomain of sub.org)
		req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(echo.HeaderOrigin, "https://attacker-sub.org")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	})
}
