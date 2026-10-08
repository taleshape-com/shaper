// SPDX-License-Identifier: MPL-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"shaper/server/core"
)

func TestIsBasePathSet(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		expected bool
	}{
		{
			name:     "empty basepath",
			basePath: "",
			expected: false,
		},
		{
			name:     "root basepath",
			basePath: "/",
			expected: false,
		},
		{
			name:     "subpath without trailing slash",
			basePath: "/shaper",
			expected: true,
		},
		{
			name:     "subpath with trailing slash",
			basePath: "/shaper/",
			expected: true,
		},
		{
			name:     "nested subpath",
			basePath: "/apps/shaper/",
			expected: true,
		},
		{
			name:     "full URL root without trailing slash",
			basePath: "https://example.com",
			expected: false,
		},
		{
			name:     "full URL root with trailing slash",
			basePath: "https://example.com/",
			expected: false,
		},
		{
			name:     "full URL with subpath",
			basePath: "https://example.com/shaper/",
			expected: true,
		},
		{
			name:     "full URL with subpath without trailing slash",
			basePath: "https://example.com/shaper",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsBasePathSet(tt.basePath))
		})
	}
}

func TestIndexHTMLWithCache_Favicon(t *testing.T) {
	mockIndexHTML := `<!DOCTYPE html>
<html>
<head>
  <link rel="icon" type="image/x-icon" href="/favicon.ico" />
  <link rel="stylesheet" href="/assets/main.css" />
  <script>window.shaper = { defaultBaseUrl: '/' }</script>
  <style></style>
</head>
<body></body>
</html>`

	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data:    []byte(mockIndexHTML),
			ModTime: time.Now(),
		},
	}

	tests := []struct {
		name                 string
		basePath             string
		favicon              string
		expectedFaviconHref  string
		expectedAssetsHref   string
	}{
		{
			name:                "default root without custom favicon",
			basePath:            "/",
			favicon:             "",
			expectedFaviconHref: `href="/favicon.ico"`,
			expectedAssetsHref:  `href="/assets/main.css"`,
		},
		{
			name:                "default root with custom favicon",
			basePath:            "/",
			favicon:             "/path/to/custom.ico",
			expectedFaviconHref: `href="/favicon.ico"`,
			expectedAssetsHref:  `href="/assets/main.css"`,
		},
		{
			name:                "basepath set and no favicon set (should use top-level favicon)",
			basePath:            "/shaper/",
			favicon:             "",
			expectedFaviconHref: `href="/favicon.ico"`,
			expectedAssetsHref:  `href="/shaper/assets/main.css"`,
		},
		{
			name:                "basepath set and custom favicon set",
			basePath:            "/shaper/",
			favicon:             "/path/to/custom.ico",
			expectedFaviconHref: `href="/shaper/favicon.ico"`,
			expectedAssetsHref:  `href="/shaper/assets/main.css"`,
		},
		{
			name:                "full URL basepath set and no favicon set (should use top-level favicon)",
			basePath:            "https://example.com/shaper/",
			favicon:             "",
			expectedFaviconHref: `href="/favicon.ico"`,
			expectedAssetsHref:  `href="https://example.com/shaper/assets/main.css"`,
		},
		{
			name:                "full URL basepath set and custom favicon set",
			basePath:            "https://example.com/shaper/",
			favicon:             "/path/to/custom.ico",
			expectedFaviconHref: `href="https://example.com/shaper/favicon.ico"`,
			expectedAssetsHref:  `href="https://example.com/shaper/assets/main.css"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			handler := indexHTMLWithCache(mockFS, time.Now(), "", tt.basePath, tt.favicon)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)

			body := rec.Body.String()
			assert.Contains(t, body, tt.expectedFaviconHref)
			assert.Contains(t, body, tt.expectedAssetsHref)
		})
	}
}

func TestFaviconRoute(t *testing.T) {
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data:    []byte("<html></html>"),
			ModTime: time.Now(),
		},
		"dist/favicon.ico": &fstest.MapFile{
			Data:    []byte("builtin-favicon-content"),
			ModTime: time.Now(),
		},
	}

	t.Run("default root without custom favicon serves builtin favicon", func(t *testing.T) {
		e := echo.New()
		app := &core.App{BasePath: "/"}
		routes(e, app, mockFS, time.Now(), "", "", "", "")

		req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "builtin-favicon-content", rec.Body.String())
	})

	t.Run("basepath set without custom favicon returns 404", func(t *testing.T) {
		e := echo.New()
		app := &core.App{BasePath: "/shaper/"}
		routes(e, app, mockFS, time.Now(), "", "", "", "")

		req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "Not Found", rec.Body.String())
	})
}
