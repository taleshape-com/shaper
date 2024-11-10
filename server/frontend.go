package server

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
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
	return echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add Last-Modified header
		if f, err := fsys.Open(strings.TrimPrefix(r.URL.Path, "/assets/")); err == nil {
			if stat, err := f.Stat(); err == nil {
				w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
			}
			f.Close()
		}
		assetHandler.ServeHTTP(w, r)
	}))
}

func indexHTMLWithCache(frontendFS fs.FS) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
	}
	return func(c echo.Context) error {
		indexFile, err := fsys.Open("index.html")
		if err != nil {
			return echo.NotFoundHandler(c)
		}
		defer indexFile.Close()

		stat, err := indexFile.Stat()
		if err != nil {
			return echo.NotFoundHandler(c)
		}

		// Add cache headers for index.html
		c.Response().Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		c.Response().Header().Set("Expires", time.Now().UTC().Format(http.TimeFormat)) // Immediate expiration
		c.Response().Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
		c.Response().Header().Set("ETag", `"`+generateETag(stat.ModTime(), stat.Size())+`"`)

		// Check If-None-Match header
		if match := c.Request().Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, generateETag(stat.ModTime(), stat.Size())) {
				return c.NoContent(http.StatusNotModified)
			}
		}

		// Check If-Modified-Since header
		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if modTime, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if stat.ModTime().Unix() <= modTime.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
		return nil
	}
}

// generateETag creates a simple ETag based on modification time and size
func generateETag(modTime time.Time, size int64) string {
	return fmt.Sprintf("%x-%x", modTime.UnixNano(), size)
}
