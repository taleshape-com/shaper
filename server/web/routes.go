// SPDX-License-Identifier: MPL-2.0

package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"shaper/server/core"
	"shaper/server/web/handler"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo-contrib/echoprometheus"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Actor is either a user or an API key.
// This is useful for audit logging and saving that context to the database.
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

func SetAPIKeyActor(app *core.App, contextKey string) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !app.LoginRequired {
				actor := &core.Actor{
					Type: core.ActorNoAuth,
				}
				ctx := core.ContextWithActor(c.Request().Context(), actor)
				c.SetRequest(c.Request().WithContext(ctx))
				return next(c)
			}

			raw := c.Get(contextKey)
			token, _ := raw.(string)
			if token == "" {
				return next(c)
			}

			apiKeyID := core.GetAPIKeyID(token)
			if apiKeyID == "" {
				return next(c)
			}

			actor := &core.Actor{
				Type: core.ActorAPIKey,
				ID:   apiKeyID,
			}
			ctx := core.ContextWithActor(c.Request().Context(), actor)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

const keyAuthContextKey = "api_key_token"

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS string, favicon string, internalUrl string, pdfDateFormat string) {
	jwtMiddleware := echojwt.WithConfig(echojwt.Config{
		TokenLookup: "header:Authorization",
		KeyFunc:     GetJWTKeyfunc(app),
	})
	apiWithAuth := e.Group("/api",
		jwtMiddleware,
		SetActor(app),
	)

	keyAuthConfig := middleware.KeyAuthConfig{
		Skipper: func(echo.Context) bool {
			return !app.LoginRequired
		},
		KeyLookup:  "header:" + echo.HeaderAuthorization,
		AuthScheme: "Bearer",
		Validator: func(key string, c echo.Context) (bool, error) {
			valid, err := core.ValidateAPIKey(app.Sqlite, c.Request().Context(), key)
			if err != nil {
				return false, err
			}
			if valid {
				c.Set(keyAuthContextKey, key)
			}
			return valid, nil
		},
	}
	apiKeyActor := SetAPIKeyActor(app, keyAuthContextKey)

	e.HEAD("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.HEAD("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.GET("/metrics", echoprometheus.NewHandler(), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor)

	// API routes - no caching
	e.GET("/api/system/config", handler.GetSystemConfig(app))
	e.POST("/api/login", handler.Login(app))
	e.POST("/api/auth/token", handler.TokenAuth(app))
	e.POST("/api/auth/public", handler.PublicAuth(app))
	e.POST("/api/auth/setup", handler.Setup(app))
	e.GET("/api/invites/:code", handler.GetInvite(app))
	e.POST("/api/invites/:code/claim", handler.ClaimInvite(app))
	e.POST("/api/data/:table_name", handler.PostEvent(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor)
	e.POST("/api/deploy", handler.Deploy(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor)
	e.GET("/api/apps", handler.ListApps(app), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor))
	e.GET("/api/public/:id/status", handler.GetDashboardStatus(app))
	apiWithAuth.GET("/version", handler.GetVersion(app))
	apiWithAuth.POST("/logout", handler.Logout(app))
	apiWithAuth.POST("/folders", handler.CreateFolder(app))
	apiWithAuth.DELETE("/folders/:id", handler.DeleteFolder(app))
	apiWithAuth.POST("/folders/:id/name", handler.RenameFolder(app))
	apiWithAuth.POST("/move", handler.MoveItems(app))
	apiWithAuth.POST("/dashboards", handler.CreateDashboard(app))
	apiWithAuth.GET("/dashboards/:id", handler.GetDashboard(app))
	apiWithAuth.DELETE("/dashboards/:id", handler.DeleteDashboard(app))
	apiWithAuth.GET("/dashboards/:id/info", handler.GetDashboardInfo(app))
	apiWithAuth.POST("/dashboards/:id/query", handler.SaveDashboardQuery(app))
	apiWithAuth.POST("/dashboards/:id/name", handler.SaveDashboardName(app))
	apiWithAuth.POST("/dashboards/:id/visibility", handler.SaveDashboardVisibility(app))
	apiWithAuth.POST("/dashboards/:id/password", handler.SaveDashboardPassword(app))
	apiWithAuth.GET("/dashboards/:id/query/:query/:filename", handler.DownloadQuery(app))
	apiWithAuth.GET("/dashboards/:id/pdf/:filename", handler.DownloadPdf(app, internalUrl, pdfDateFormat))
	if !app.NoTasks {
		apiWithAuth.POST("/tasks", handler.CreateTask(app))
		apiWithAuth.GET("/tasks/:id", handler.GetTask(app))
		apiWithAuth.DELETE("/tasks/:id", handler.DeleteTask(app))
		apiWithAuth.POST("/tasks/:id/content", handler.SaveTaskContent(app))
		apiWithAuth.POST("/tasks/:id/name", handler.SaveTaskName(app))
		apiWithAuth.POST("/run/task", handler.RunTask(app))
	}
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

	e.GET("/view/:id", serveViewHTML(frontendFS, modTime), CacheControl(CacheConfig{
		MaxAge: 24 * time.Hour, // 1 day
		Public: true,
		// TODO: Once we version this file properly can set Immutable: true, and cache for a year
	}))

	e.GET("/_internal/pdfview/:id", servePdfViewHTML(frontendFS, modTime), CacheControl(CacheConfig{
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
	e.GET("/*", indexHTMLWithCache(frontendFS, modTime, customCSS, app.BasePath))
}

func jwtOrAPIKeyMiddleware(app *core.App, jwtMiddleware echo.MiddlewareFunc, setActorMid echo.MiddlewareFunc, keyAuthMiddleware echo.MiddlewareFunc, apiKeyActorMid echo.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		jwtChain := jwtMiddleware(setActorMid(next))
		apiKeyChain := keyAuthMiddleware(apiKeyActorMid(next))
		return func(c echo.Context) error {
			token := extractAuthorizationToken(c)
			if core.IsAPIKeyToken(token) || !app.LoginRequired {
				return apiKeyChain(c)
			}
			return jwtChain(c)
		}
	}
}

func extractAuthorizationToken(c echo.Context) string {
	header := strings.TrimSpace(c.Request().Header.Get(echo.HeaderAuthorization))
	if header == "" {
		return ""
	}
	const bearerPrefix = "Bearer "
	if len(header) > len(bearerPrefix) && strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return strings.TrimSpace(header[len(bearerPrefix):])
	}
	return header
}

// We overide the Keyfunc handler so we can send the JWT secret dynamically when it changes over time
func GetJWTKeyfunc(app *core.App) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != echojwt.AlgorithmHS256 {
			return nil, &echojwt.TokenError{Token: token, Err: fmt.Errorf("unexpected jwt signing method=%v", token.Header["alg"])}
		}
		return app.JWTSecret, nil
	}
}
