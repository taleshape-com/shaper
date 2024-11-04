package server

import (
	"fmt"
	"io/fs"
	"shaper/core"
	"shaper/server/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS) {
	apiWithAuth := e.Group("/api", middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		// TODO: make this config
		KeyLookup: "cookie:shaper-token",
		Validator: func(key string, c echo.Context) (bool, error) {
			fmt.Println(key)
			return key == "test", nil
		},
		ErrorHandler: func(err error, c echo.Context) error {
			return c.JSON(401, map[string]string{"error": "Unauthorized"})
		},
	}))

	e.POST("/api/login/cookie", handler.CookieLogin(app))
	apiWithAuth.GET("/login/cookie/test", handler.TestCookie)
	apiWithAuth.GET("/dashboards", handler.ListDashboards(app))
	apiWithAuth.GET("/dashboard/:name", handler.GetDashboard(app))
	e.GET("/assets/*", frontend(frontendFS))
	e.GET("/icon.svg", frontend(frontendFS))
	e.GET("/*", indexHTML(frontendFS))
}
