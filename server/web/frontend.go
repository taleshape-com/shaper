// SPDX-License-Identifier: MPL-2.0

package web

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

//go:embed view.html
var viewHTML []byte

//go:embed pdfview.html
var pdfViewHTML []byte

func frontend(frontendFS fs.FS) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		fmt.Printf("Error creating frontend filesystem: %v\n", err)
		os.Exit(1)
	}
	assetHandler := http.FileServer(http.FS(fsys))
	return echo.WrapHandler(http.HandlerFunc(assetHandler.ServeHTTP))
}

func serveFavicon(frontendFS fs.FS, favicon string, modTime time.Time) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		fmt.Printf("Error creating favicon filesystem: %v\n", err)
		os.Exit(1)
	}
	return func(c echo.Context) error {
		var file fs.File
		var err error
		if favicon != "" {
			file, err = os.Open(favicon)
		} else {
			file, err = fsys.Open("favicon.ico")
		}
		if err != nil {
			return c.String(http.StatusNotFound, "Not Found")
		}
		defer file.Close()
		http.ServeContent(c.Response(), c.Request(), "favicon.ico", modTime, file.(io.ReadSeeker))
		return nil
	}
}

func serveCustomCSS(customCSS any, modTime time.Time) echo.HandlerFunc {
	var store *CustomCSSStore
	switch v := customCSS.(type) {
	case *CustomCSSStore:
		store = v
	case string:
		store = NewCustomCSSStore("", "", v, modTime, nil)
	default:
		store = NewCustomCSSStore("", "", "", modTime, nil)
	}

	return func(c echo.Context) error {
		content, etag, currentModTime, lastModified := store.Get()

		c.Response().Header().Set("ETag", `"`+etag+`"`)
		c.Response().Header().Set("Last-Modified", lastModified)

		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				return c.NoContent(http.StatusNotModified)
			}
		}

		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if m, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if currentModTime.Unix() <= m.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), "custom.css", currentModTime, bytes.NewReader(content))
		return nil
	}
}

func serveEmbedJS(frontendFS fs.FS, modTime time.Time) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		fmt.Printf("Error creating embed JS filesystem: %v\n", err)
		os.Exit(1)
	}
	lastModified := modTime.UTC().Format(http.TimeFormat)

	return func(c echo.Context) error {
		filename := path.Base(c.Request().URL.Path)
		if filename != "shaper.js" && filename != "shaper.js.map" {
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}

		file, err := fsys.Open(path.Join("embed", filename))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Error reading file stat")
		}

		etag := generateETag(modTime, stat.Size())
		c.Response().Header().Set("ETag", `"`+etag+`"`)
		c.Response().Header().Set("Last-Modified", lastModified)

		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				return c.NoContent(http.StatusNotModified)
			}
		}

		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if m, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if modTime.Unix() <= m.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), filename, modTime, file.(io.ReadSeeker))
		return nil
	}
}

func serveViewHTML(frontendFS fs.FS, modTime time.Time) echo.HandlerFunc {
	etag := generateETag(modTime, int64(len(viewHTML)))
	lastModified := modTime.UTC().Format(http.TimeFormat)
	return func(c echo.Context) error {
		c.Response().Header().Set("ETag", `"`+etag+`"`)
		c.Response().Header().Set("Last-Modified", lastModified)

		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				return c.NoContent(http.StatusNotModified)
			}
		}

		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if m, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if modTime.Unix() <= m.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), "view.html", modTime, bytes.NewReader(viewHTML))
		return nil
	}
}

func servePdfViewHTML(frontendFS fs.FS, modTime time.Time) echo.HandlerFunc {
	etag := generateETag(modTime, int64(len(pdfViewHTML)))
	lastModified := modTime.UTC().Format(http.TimeFormat)
	return func(c echo.Context) error {
		c.Response().Header().Set("ETag", `"`+etag+`"`)
		c.Response().Header().Set("Last-Modified", lastModified)

		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				return c.NoContent(http.StatusNotModified)
			}
		}

		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if m, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if modTime.Unix() <= m.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), "pdfview.html", modTime, bytes.NewReader(pdfViewHTML))
		return nil
	}
}

// IsBasePathSet returns true if a custom base path (subpath) is configured.
func IsBasePathSet(basePath string) bool {
	if basePath == "" || basePath == "/" {
		return false
	}
	u, err := url.Parse(basePath)
	if err == nil {
		return u.Path != "" && u.Path != "/"
	}
	return basePath != "/"
}

func indexHTMLWithCache(frontendFS fs.FS, modTime time.Time, customCSS any, basePath string, favicon string) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		fmt.Printf("Error creating index HTML filesystem: %v\n", err)
		os.Exit(1)
	}
	indexFile, err := fsys.Open("index.html")
	if err != nil {
		fmt.Printf("Error opening index.html: %v\n", err)
		os.Exit(1)
	}
	defer indexFile.Close()
	stat, err := indexFile.Stat()
	if err != nil {
		fmt.Printf("Error getting index.html stats: %v\n", err)
		os.Exit(1)
	}
	lastModified := modTime.UTC().Format(http.TimeFormat)
	etag := generateETag(modTime, stat.Size())
	fileContent, err := io.ReadAll(indexFile)
	if err != nil {
		fmt.Printf("Error reading index.html content: %v\n", err)
		os.Exit(1)
	}

	html := string(fileContent)
	// Set base path
	html = strings.ReplaceAll(html, "\"/assets/", "\""+basePath+"assets/")
	html = strings.ReplaceAll(html, "\"./assets/", "\""+basePath+"assets/")
	if favicon != "" || !IsBasePathSet(basePath) {
		html = strings.ReplaceAll(html, "\"/favicon.ico\"", "\""+basePath+"favicon.ico\"")
		html = strings.ReplaceAll(html, "\"./favicon.ico\"", "\""+basePath+"favicon.ico\"")
	} else {
		html = strings.ReplaceAll(html, "\"./favicon.ico\"", "\"/favicon.ico\"")
	}
	html = strings.Replace(html, "<script>window.shaper = { defaultBaseUrl: '/' }</script>", fmt.Sprintf("<script>window.shaper = { defaultBaseUrl: %q };</script>", basePath), 1)
	// Ensure custom CSS stylesheet link is present and configured for basePath
	html = strings.ReplaceAll(html, "\"/embed/custom.css\"", "\""+basePath+"embed/custom.css\"")
	html = strings.ReplaceAll(html, "\"./embed/custom.css\"", "\""+basePath+"embed/custom.css\"")
	if !strings.Contains(html, "custom.css") {
		linkTag := fmt.Sprintf("<link rel=\"stylesheet\" href=\"%sembed/custom.css\" />", basePath)
		if strings.Contains(html, "<style></style>") {
			html = strings.Replace(html, "<style></style>", linkTag, 1)
		} else if strings.Contains(html, "</head>") {
			html = strings.Replace(html, "</head>", linkTag+"\n</head>", 1)
		}
	}
	if strings.Contains(html, "<style></style>") {
		html = strings.Replace(html, "<style></style>", "", 1)
	}

	return func(c echo.Context) error {
		// Add cache headers for index.html
		c.Response().Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		c.Response().Header().Set("Expires", time.Now().UTC().Format(http.TimeFormat)) // Immediate expiration
		c.Response().Header().Set("Last-Modified", lastModified)
		c.Response().Header().Set("ETag", `"`+etag+`"`)

		// Check If-None-Match header
		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				return c.NoContent(http.StatusNotModified)
			}
		}

		// Check If-Modified-Since header
		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if m, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if modTime.Unix() <= m.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), "index.html", modTime, strings.NewReader(html))
		return nil
	}
}

// generateETag creates a simple ETag based on modification time and size
func generateETag(modTime time.Time, size int64) string {
	return fmt.Sprintf("%x", modTime.UnixNano())
}

// generateContentETag creates an ETag based on SHA-256 hash of content
func generateContentETag(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:8])
}
