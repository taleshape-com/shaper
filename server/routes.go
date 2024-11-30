package server

import (
	"io/fs"
	"shaper/core"
	"shaper/server/handler"
	"time"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS string, favicon string) {

	apiWithAuth := e.Group("/api", echojwt.WithConfig(echojwt.Config{
		TokenLookup: "header:Authorization",
		SigningKey:  app.JWTSecret,
	}))

	// API routes - no caching
	e.POST("/api/login/token", handler.TokenLogin(app))
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

	e.GET("/embed/*", frontend(frontendFS), CacheControl(CacheConfig{
		MaxAge: 24 * time.Hour, // 1 day
		Public: true,
		// TODO: Once we version this file properly can set Immutable: true, and cache for a year
	}))

	// Icon - moderate caching
	e.GET("/favicon.ico", serveFavicon(frontendFS, favicon, modTime), CacheControl(CacheConfig{
		MaxAge: 24 * time.Hour, // 1 day
		Public: true,
	}))

	// Index HTML - light caching with revalidation
	e.GET("/*", indexHTMLWithCache(frontendFS, modTime, customCSS, "/favicon.ico"))
}
