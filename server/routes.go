package server

import (
	"io/fs"
	"shaper/core"
	"shaper/server/handler"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS string) {
	apiWithAuth := e.Group("/api", middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "cookie:shaper-token",
		Validator: func(key string, c echo.Context) (bool, error) {
			return core.ValidLogin(app, c.Request().Context(), key)
		},
		ErrorHandler: func(err error, c echo.Context) error {
			return c.JSON(401, map[string]string{"error": "Unauthorized"})
		},
	}))

	// API routes - no caching
	e.POST("/api/login/cookie", handler.CookieLogin(app))
	apiWithAuth.GET("/login/cookie/test", handler.TestCookie)
	apiWithAuth.GET("/dashboards", handler.ListDashboards(app))
	apiWithAuth.GET("/dashboards/:name", handler.GetDashboard(app))
	apiWithAuth.GET("/dashboards/:name/query/:query/:filename", handler.DownloadQuery(app))

	// Static assets - aggressive caching
	assetsGroup := e.Group("/assets", CacheControl(CacheConfig{
		MaxAge:    365 * 24 * time.Hour, // 1 year
		Public:    true,
		Immutable: true,
	}))
	assetsGroup.GET("/*", frontend(frontendFS))

	// Icon - moderate caching
	e.GET("/icon.svg", frontend(frontendFS), CacheControl(CacheConfig{
		MaxAge: 24 * time.Hour, // 1 day
		Public: true,
	}))

	// Index HTML - light caching with revalidation
	e.GET("/*", indexHTMLWithCache(frontendFS, modTime, customCSS))
}
