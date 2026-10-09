// SPDX-License-Identifier: MPL-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
			handler := indexHTMLWithCache(mockFS, time.Now(), tt.basePath, tt.favicon)

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

	t.Run("custom.css is placed after bundled css to win the cascade", func(t *testing.T) {
		mockHTMLWithEarlyCustomCSS := `<!doctype html>
<html>
<head>
  <link id="shaper-custom-css" rel="stylesheet" href="/embed/custom.css" />
  <link rel="stylesheet" href="/assets/index-123.css" />
</head>
<body></body>
</html>`

		mockFS := fstest.MapFS{
			"dist/index.html": &fstest.MapFile{
				Data:    []byte(mockHTMLWithEarlyCustomCSS),
				ModTime: time.Now(),
			},
		}

		e := echo.New()
		handler := indexHTMLWithCache(mockFS, time.Now(), "/", "")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		require.NoError(t, err)
		body := rec.Body.String()

		indexCSSPos := strings.Index(body, "/assets/index-123.css")
		customCSSPos := strings.Index(body, "/embed/custom.css")
		assert.Greater(t, indexCSSPos, -1)
		assert.Greater(t, customCSSPos, -1)
		// custom.css must appear AFTER index-123.css so that custom styles override default styles
		assert.Greater(t, customCSSPos, indexCSSPos, "custom.css should appear after index.css")
		// It should only appear once
		assert.Equal(t, 1, strings.Count(body, "custom.css"))
	})
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

func TestServeCustomCSS(t *testing.T) {
	modTime := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	css := "body { background: red; }"

	t.Run("serves custom css with etag and last-modified", func(t *testing.T) {
		e := echo.New()
		handler := serveCustomCSS(css, modTime)

		req := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/css")
		assert.NotEmpty(t, rec.Header().Get("ETag"))
		assert.NotEmpty(t, rec.Header().Get("Last-Modified"))
		assert.Equal(t, css, rec.Body.String())
	})

	t.Run("returns 304 on matching If-None-Match", func(t *testing.T) {
		e := echo.New()
		handler := serveCustomCSS(css, modTime)

		req1 := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)
		err := handler(c1)
		require.NoError(t, err)
		etag := rec1.Header().Get("ETag")

		req2 := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
		req2.Header.Set("If-None-Match", etag)
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)
		err = handler(c2)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotModified, rec2.Code)
	})

	t.Run("returns 304 on matching If-Modified-Since", func(t *testing.T) {
		e := echo.New()
		handler := serveCustomCSS(css, modTime)

		req := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
		req.Header.Set("If-Modified-Since", modTime.Format(http.TimeFormat))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotModified, rec.Code)
	})

	t.Run("serves empty body when custom css is empty", func(t *testing.T) {
		e := echo.New()
		handler := serveCustomCSS("", modTime)

		req := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/css")
		assert.Empty(t, rec.Body.String())
	})
}

func TestServeEmbedJS(t *testing.T) {
	modTime := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	staticJS := "console.log('shaper embed');"
	mockFS := fstest.MapFS{
		"dist/embed/shaper.js": &fstest.MapFile{
			Data:    []byte(staticJS),
			ModTime: modTime,
		},
		"dist/embed/shaper.js.map": &fstest.MapFile{
			Data:    []byte(`{"version":3}`),
			ModTime: modTime,
		},
	}

	t.Run("serves static shaper.js with etag without dynamic injection", func(t *testing.T) {
		e := echo.New()
		handler := serveEmbedJS(mockFS, modTime)

		req := httptest.NewRequest(http.MethodGet, "/embed/shaper.js", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, staticJS, rec.Body.String())
		assert.NotEmpty(t, rec.Header().Get("ETag"))
		assert.NotEmpty(t, rec.Header().Get("Last-Modified"))
		assert.NotContains(t, rec.Body.String(), "shaper.defaultBaseUrl")
	})

	t.Run("returns 304 on matching If-None-Match", func(t *testing.T) {
		e := echo.New()
		handler := serveEmbedJS(mockFS, modTime)

		req1 := httptest.NewRequest(http.MethodGet, "/embed/shaper.js", nil)
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)
		err := handler(c1)
		require.NoError(t, err)
		etag := rec1.Header().Get("ETag")

		req2 := httptest.NewRequest(http.MethodGet, "/embed/shaper.js", nil)
		req2.Header.Set("If-None-Match", etag)
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)
		err = handler(c2)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotModified, rec2.Code)
	})

	t.Run("returns 404 for unknown file", func(t *testing.T) {
		e := echo.New()
		handler := serveEmbedJS(mockFS, modTime)

		req := httptest.NewRequest(http.MethodGet, "/embed/other.js", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		assert.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusNotFound, httpErr.Code)
	})
}

func TestEmbedAndCustomCSSRoutes_CacheControl(t *testing.T) {
	modTime := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mockFS := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data:    []byte("<html><head></head><body></body></html>"),
			ModTime: modTime,
		},
		"dist/embed/shaper.js": &fstest.MapFile{
			Data:    []byte("console.log('shaper');"),
			ModTime: modTime,
		},
	}

	e := echo.New()
	app := &core.App{BasePath: "/"}
	routes(e, app, mockFS, modTime, "body { color: blue; }", "", "", "")

	t.Run("/embed/custom.css has revalidation cache headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "public, max-age=0, must-revalidate", rec.Header().Get("Cache-Control"))
		assert.NotEmpty(t, rec.Header().Get("ETag"))
	})

	t.Run("/embed/shaper.js has revalidation cache headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/embed/shaper.js", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "public, max-age=0, must-revalidate", rec.Header().Get("Cache-Control"))
		assert.NotEmpty(t, rec.Header().Get("ETag"))
	})

	t.Run("/view/:id has revalidation cache headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/view/test-dash", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "public, max-age=0, must-revalidate", rec.Header().Get("Cache-Control"))
		assert.NotEmpty(t, rec.Header().Get("ETag"))
	})
}
