// SPDX-License-Identifier: MPL-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomCSSStore_Reload(t *testing.T) {
	tempDir := t.TempDir()
	cssFile := filepath.Join(tempDir, "style.css")

	err := os.WriteFile(cssFile, []byte(".initial { color: red; }"), 0644)
	require.NoError(t, err)

	stat1, err := os.Stat(cssFile)
	require.NoError(t, err)
	modTime := stat1.ModTime()
	store := NewCustomCSSStore(cssFile, ".inline { margin: 0; }", ".inline { margin: 0; }\n.initial { color: red; }", modTime, nil)

	assert.Equal(t, ".inline { margin: 0; }\n.initial { color: red; }", store.Content())
	_, initialETag, _, _ := store.Get()
	assert.NotEmpty(t, initialETag)

	t.Run("reload updates content, modTime and ETag", func(t *testing.T) {
		time.Sleep(20 * time.Millisecond)
		err := os.WriteFile(cssFile, []byte(".updated { color: blue; }"), 0644)
		require.NoError(t, err)

		err = store.Reload()
		require.NoError(t, err)

		assert.Equal(t, ".inline { margin: 0; }\n.updated { color: blue; }", store.Content())
		content, newETag, newModTime, _ := store.Get()
		assert.Equal(t, ".inline { margin: 0; }\n.updated { color: blue; }", string(content))
		assert.NotEqual(t, initialETag, newETag)
		assert.False(t, newModTime.Before(modTime))
	})

	t.Run("reload fails gracefully on missing file", func(t *testing.T) {
		prevContent := store.Content()
		missingStore := NewCustomCSSStore(filepath.Join(tempDir, "missing.css"), "", prevContent, modTime, nil)

		err := missingStore.Reload()
		assert.Error(t, err)
		assert.Equal(t, prevContent, missingStore.Content(), "content should not be wiped on reload failure")
	})

	t.Run("reload without file configured is a no-op", func(t *testing.T) {
		noFileStore := NewCustomCSSStore("", "", ".static { padding: 0; }", modTime, nil)
		err := noFileStore.Reload()
		assert.NoError(t, err)
		assert.Equal(t, ".static { padding: 0; }", noFileStore.Content())
	})
}

func TestCustomCSSStore_HTTPRevalidationFlow(t *testing.T) {
	tempDir := t.TempDir()
	cssFile := filepath.Join(tempDir, "style.css")

	err := os.WriteFile(cssFile, []byte("body { background: white; }"), 0644)
	require.NoError(t, err)

	modTime := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	store := NewCustomCSSStore(cssFile, "", "body { background: white; }", modTime, nil)

	e := echo.New()
	handler := serveCustomCSS(store, modTime)

	// 1. Initial request -> 200 OK with ETag 1
	req1 := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	err = handler(c1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec1.Code)
	etag1 := rec1.Header().Get("ETag")
	assert.NotEmpty(t, etag1)
	assert.Equal(t, "body { background: white; }", rec1.Body.String())

	// 2. Client revalidates with ETag 1 -> 304 Not Modified
	req2 := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
	req2.Header.Set("If-None-Match", etag1)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	err = handler(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotModified, rec2.Code)

	// 3. File changes on disk and reload is triggered (simulating SIGHUP)
	err = os.WriteFile(cssFile, []byte("body { background: darkgreen; }"), 0644)
	require.NoError(t, err)
	err = store.Reload()
	require.NoError(t, err)

	// 4. Client sends old ETag 1 -> now returns 200 OK with new content and ETag 2
	req3 := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
	req3.Header.Set("If-None-Match", etag1)
	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	err = handler(c3)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec3.Code)
	etag2 := rec3.Header().Get("ETag")
	assert.NotEmpty(t, etag2)
	assert.NotEqual(t, etag1, etag2)
	assert.Equal(t, "body { background: darkgreen; }", rec3.Body.String())

	// 5. Client revalidates with ETag 2 -> 304 Not Modified
	req4 := httptest.NewRequest(http.MethodGet, "/embed/custom.css", nil)
	req4.Header.Set("If-None-Match", etag2)
	rec4 := httptest.NewRecorder()
	c4 := e.NewContext(req4, rec4)
	err = handler(c4)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotModified, rec4.Code)
}
