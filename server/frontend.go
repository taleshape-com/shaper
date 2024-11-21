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
	return echo.WrapHandler(http.HandlerFunc(assetHandler.ServeHTTP))
}

func indexHTMLWithCache(frontendFS fs.FS, modTime time.Time, customCSS string) echo.HandlerFunc {
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
	content := strings.Replace(string(fileContent), "<style></style>", "<style>"+customCSS+"</style>", 1)
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

		http.ServeContent(c.Response(), c.Request(), "index.html", modTime, strings.NewReader(content))
		return nil
	}
}

// generateETag creates a simple ETag based on modification time and size
func generateETag(modTime time.Time, size int64) string {
	return fmt.Sprintf("%x", modTime.UnixNano())
}
