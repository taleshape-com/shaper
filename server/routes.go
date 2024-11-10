package server

import (
	"io/fs"
	"shaper/core"
	"shaper/server/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// CacheControl middleware adds cache headers based on path
func CacheControl(duration string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", duration)
			return next(c)
		}
	}
}

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS) {
	apiWithAuth := e.Group("/api", middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "cookie:shaper-token",
		Validator: func(key string, c echo.Context) (bool, error) {
			return core.ValidLogin(app, c.Request().Context(), key)
		},
		ErrorHandler: func(err error, c echo.Context) error {
			return c.JSON(401, map[string]string{"error": "Unauthorized"})
		},
	}))

	e.POST("/api/login/cookie", handler.CookieLogin(app))
	apiWithAuth.GET("/login/cookie/test", handler.TestCookie)
	apiWithAuth.GET("/dashboards", handler.ListDashboards(app))
	apiWithAuth.GET("/dashboard/:name", handler.GetDashboard(app))

	assetsGroup := e.Group("/assets")
	assetsGroup.Use(CacheControl("public, max-age=31536000, immutable")) // 1 year
	assetsGroup.GET("/*", frontend(frontendFS))

	iconGroup := e.Group("")
	iconGroup.Use(CacheControl("public, max-age=86400")) // 1 day
	iconGroup.GET("/icon.svg", frontend(frontendFS))

	e.GET("/*", indexHTMLWithCache(frontendFS))
}
