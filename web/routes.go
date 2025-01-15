package web

import (
	"io/fs"
	"net/http"
	"shaper/core"
	"shaper/web/handler"
	"time"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS string, favicon string) {

	apiWithAuth := e.Group("/api", echojwt.WithConfig(echojwt.Config{
		TokenLookup: "header:Authorization",
		SigningKey:  app.JWTSecret,
	}))

	e.GET("/status", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// API routes - no caching
	e.POST("/api/login/token", handler.TokenLogin(app))
	e.POST("/api/auth/token", handler.TokenAuth(app))
	apiWithAuth.GET("/dashboards", handler.ListDashboards(app))
	apiWithAuth.POST("/dashboards", handler.CreateDashboard(app))
	apiWithAuth.GET("/dashboards/:id", handler.GetDashboard(app))
	apiWithAuth.DELETE("/dashboards/:id", handler.DeleteDashboard(app))
	apiWithAuth.GET("/dashboards/:id/query", handler.GetDashboardQuery(app))
	apiWithAuth.POST("/dashboards/:id/query", handler.SaveDashboardQuery(app))
	apiWithAuth.POST("/dashboards/:id/name", handler.SaveDashboardName(app))
	apiWithAuth.GET("/dashboards/:id/query/:query/:filename", handler.DownloadQuery(app))
	apiWithAuth.POST("/query/dashboard", handler.PreviewDashboardQuery(app))
	apiWithAuth.POST("/admin/reset-jwt-secret", handler.ResetJWTSecret(app))

	// Static assets - aggressive caching
	assetsGroup := e.Group("/assets", CacheControl(CacheConfig{
		MaxAge:    365 * 24 * time.Hour, // 1 year
		Public:    true,
		Immutable: true,
	}))
	assetsGroup.GET("/*", frontend(frontendFS))

	e.GET("/embed/*", serveEmbedJS(frontendFS, modTime, customCSS), CacheControl(CacheConfig{
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
