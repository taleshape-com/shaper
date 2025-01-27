package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"shaper/core"
	"shaper/web/handler"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func SetActor(app *core.App) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)

			var actor *core.Actor
			if userID, ok := claims["userId"].(string); ok {
				actor = &core.Actor{
					Type: core.ActorUser,
					ID:   userID,
				}
			} else if apiKeyID, ok := claims["apiKeyId"].(string); ok {
				actor = &core.Actor{
					Type: core.ActorAPIKey,
					ID:   apiKeyID,
				}
			} else if !app.LoginRequired {
				actor = &core.Actor{
					Type: core.ActorNoAuth,
				}
			}

			if actor != nil {
				c.SetRequest(c.Request().WithContext(core.ContextWithActor(c.Request().Context(), actor)))
			}

			return next(c)
		}
	}
}

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS string, favicon string) {

	apiWithAuth := e.Group("/api",
		echojwt.WithConfig(echojwt.Config{
			TokenLookup: "header:Authorization",
			KeyFunc:     GetJWTKeyfunc(app),
		}),
		SetActor(app),
	)

	e.GET("/status", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// API routes - no caching
	e.GET("/api/login/enabled", handler.LoginEnabled(app))
	e.POST("/api/login", handler.Login(app))
	e.POST("/api/auth/token", handler.TokenAuth(app))
	e.POST("/api/auth/setup", handler.Setup(app))
	e.GET("/api/invites/:code", handler.GetInvite(app))
	e.POST("/api/invites/:code/claim", handler.ClaimInvite(app))
	apiWithAuth.POST("/logout", handler.Logout(app))
	apiWithAuth.GET("/dashboards", handler.ListDashboards(app))
	apiWithAuth.POST("/dashboards", handler.CreateDashboard(app))
	apiWithAuth.GET("/dashboards/:id", handler.GetDashboard(app))
	apiWithAuth.DELETE("/dashboards/:id", handler.DeleteDashboard(app))
	apiWithAuth.GET("/dashboards/:id/query", handler.GetDashboardQuery(app))
	apiWithAuth.POST("/dashboards/:id/query", handler.SaveDashboardQuery(app))
	apiWithAuth.POST("/dashboards/:id/name", handler.SaveDashboardName(app))
	apiWithAuth.GET("/dashboards/:id/query/:query/:filename", handler.DownloadQuery(app))
	apiWithAuth.POST("/query/dashboard", handler.PreviewDashboardQuery(app))
	apiWithAuth.GET("/users", handler.ListUsers(app))
	apiWithAuth.DELETE("/users/:id", handler.DeleteUser(app))
	apiWithAuth.DELETE("/invites/:code", handler.DeleteInvite(app))
	apiWithAuth.POST("/invites", handler.CreateInvite(app))
	apiWithAuth.GET("/keys", handler.ListAPIKeys(app))
	apiWithAuth.POST("/keys", handler.CreateAPIKey(app))
	apiWithAuth.DELETE("/keys/:id", handler.DeleteAPIKey(app))
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

// We overide the Keyfunc handler so we can send the JWT secret dynamically when it changes over time
func GetJWTKeyfunc(app *core.App) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != echojwt.AlgorithmHS256 {
			return nil, &echojwt.TokenError{Token: token, Err: fmt.Errorf("unexpected jwt signing method=%v", token.Header["alg"])}
		}
		return app.JWTSecret, nil
	}
}
