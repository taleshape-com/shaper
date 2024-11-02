package server

import (
	"shaper/core"
	"shaper/server/handler"

	"github.com/labstack/echo/v4"
)

func routes(e *echo.Echo, app *core.App) {
	e.GET("/api/sample", handler.Sample(app))
}
