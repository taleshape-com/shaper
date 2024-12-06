package web

import (
	"bytes"
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

func frontend(frontendFS fs.FS) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
	}
	assetHandler := http.FileServer(http.FS(fsys))
	return echo.WrapHandler(http.HandlerFunc(assetHandler.ServeHTTP))
}
func serveFavicon(frontendFS fs.FS, favicon string, modTime time.Time) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
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

func serveEmbedJS(frontendFS fs.FS, modTime time.Time) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
	}
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

		if filename == "shaper.js" {
			content, err := io.ReadAll(file)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Error reading file")
			}

			defaultBaseUrl := strings.TrimSuffix(getRequestURL(c.Request()).String(), "/embed/shaper.js")
			appendScript := fmt.Sprintf("\nwindow.shaper.defaultBaseUrl = %q;\n", defaultBaseUrl)
			content = append(content, []byte(appendScript)...)

			http.ServeContent(c.Response(), c.Request(), filename, modTime, bytes.NewReader(content))
		} else {
			http.ServeContent(c.Response(), c.Request(), filename, modTime, file.(io.ReadSeeker))
		}

		return nil
	}
}

// getRequestURL reconstructs the full URL, handling reverse proxy scenarios
func getRequestURL(r *http.Request) *url.URL {
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

func indexHTMLWithCache(frontendFS fs.FS, modTime time.Time, customCSS string, favicon string) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
	}
	indexFile, err := fsys.Open("index.html")
	if err != nil {
		panic(err)
	}
	defer indexFile.Close()
	stat, err := indexFile.Stat()
	if err != nil {
		panic(err)
	}
	lastModified := modTime.UTC().Format(http.TimeFormat)
	etag := generateETag(modTime, stat.Size())
	fileContent, err := io.ReadAll(indexFile)
	if err != nil {
		panic(err)
	}

	// Inject custom CSS
	html := strings.Replace(string(fileContent), "<style></style>", "<style>"+customCSS+"</style>", 1)

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
