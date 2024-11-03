package server

import (
	"io/fs"
	"shaper/core"
	"shaper/server/handler"

	"github.com/labstack/echo/v4"
)

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS) {
	e.GET("/api/dashboards", handler.ListDashboards(app))
	e.GET("/api/dashboard/:name", handler.GetDashboard(app))
	e.GET("/assets/*", frontend(frontendFS))
	e.GET("/icon.svg", frontend(frontendFS))
	e.GET("/*", indexHTML(frontendFS))
}
