package server

import (
	"io"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

func frontend(frontendFS fs.FS) echo.HandlerFunc {
	fsys, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic(err)
	}
	assetHandler := http.FileServer(http.FS(fsys))
	return echo.WrapHandler(assetHandler)
}

func indexHTML(frontendFS fs.FS) echo.HandlerFunc {
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
		http.ServeContent(c.Response(), c.Request(), "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
		return nil
	}
}
