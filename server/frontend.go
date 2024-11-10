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

// generateETag creates a simple ETag based on modification time and size
func generateETag(modTime time.Time, size int64) string {
	return fmt.Sprintf("%x-%x", modTime.UnixNano(), size)
}

func frontendWithHeaders(frontendFS fs.FS) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
	}

	return func(c echo.Context) error {
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/assets/") {
			path = strings.TrimPrefix(path, "/assets/")
		}

		file, err := fsys.Open(path)
		if err != nil {
			return echo.NotFoundHandler(c)
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return echo.NotFoundHandler(c)
		}

		// Set Last-Modified header
		c.Response().Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))

		// Check If-Modified-Since header
		if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
			if modTime, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
				if stat.ModTime().Unix() <= modTime.Unix() {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}

		http.ServeContent(c.Response(), c.Request(), path, stat.ModTime(), file.(io.ReadSeeker))
		return nil
	}
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
