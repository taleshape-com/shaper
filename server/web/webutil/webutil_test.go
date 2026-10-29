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
