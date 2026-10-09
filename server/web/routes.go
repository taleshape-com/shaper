// SPDX-License-Identifier: MPL-2.0

package web

import (
	"fmt"
	"io/fs"
	"net"
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
	"golang.org/x/time/rate"
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
			} else if _, ok := claims["public"].(string); ok {
				actor = &core.Actor{
					Type: core.ActorPublic,
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

func RequirePermission(app *core.App, permissions ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			actor := core.ActorFromContext(c.Request().Context())
			if actor == nil {
				return echo.ErrUnauthorized
			}
			for _, permission := range permissions {
				if actor.HasPermission(c.Request().Context(), app.Sqlite, permission) {
					return next(c)
				}
			}
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Missing required permission"}, "  ")
		}
	}
}

// createRateLimiter creates an Echo rate limiter middleware with common client IP extraction and JSON error formatting.
func createRateLimiter(r rate.Limit, burst int, expiresIn time.Duration, errorMessage string) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      r,
				Burst:     burst,
				ExpiresIn: expiresIn,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			if id == "" {
				host, _, err := net.SplitHostPort(ctx.Request().RemoteAddr)
				if err == nil {
					id = host
				} else {
					id = ctx.Request().RemoteAddr
				}
			}
			return id, nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Failed to determine client IP"}, "  ")
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			c.Response().Header().Set("Retry-After", "60")
			return c.JSONPretty(http.StatusTooManyRequests, struct {
				Error string `json:"error"`
			}{Error: errorMessage}, "  ")
		},
	})
}

// LoginRateLimiter returns an Echo middleware that rate limits login attempts per IP address.
// By default, it allows up to 5 requests per minute with a burst of 5 requests.
func LoginRateLimiter() echo.MiddlewareFunc {
	return LoginRateLimiterWithConfig(5.0/60.0, 5, 3*time.Minute)
}

// LoginRateLimiterWithConfig returns a login rate limiting middleware with custom rate limit settings.
func LoginRateLimiterWithConfig(r rate.Limit, burst int, expiresIn time.Duration) echo.MiddlewareFunc {
	return createRateLimiter(r, burst, expiresIn, "Too many login attempts, please try again later")
}

// PublicAuthRateLimiter returns an Echo middleware that rate limits public and password-protected dashboard auth attempts.
// By default, it allows up to 20 requests per minute with a burst of 10 requests.
func PublicAuthRateLimiter() echo.MiddlewareFunc {
	return PublicAuthRateLimiterWithConfig(20.0/60.0, 10, 3*time.Minute)
}

// PublicAuthRateLimiterWithConfig returns a public auth rate limiting middleware with custom rate limit settings.
func PublicAuthRateLimiterWithConfig(r rate.Limit, burst int, expiresIn time.Duration) echo.MiddlewareFunc {
	return createRateLimiter(r, burst, expiresIn, "Too many requests, please try again later")
}

// InviteRateLimiter returns an Echo middleware that rate limits invite code queries and claims.
// By default, it allows up to 10 requests per minute with a burst of 5 requests.
func InviteRateLimiter() echo.MiddlewareFunc {
	return InviteRateLimiterWithConfig(10.0/60.0, 5, 3*time.Minute)
}

// InviteRateLimiterWithConfig returns an invite rate limiting middleware with custom rate limit settings.
func InviteRateLimiterWithConfig(r rate.Limit, burst int, expiresIn time.Duration) echo.MiddlewareFunc {
	return createRateLimiter(r, burst, expiresIn, "Too many invite requests, please try again later")
}

// DownloadRateLimiter returns an Echo middleware that rate limits file/report downloads and Chromium renders.
// By default, it allows up to 10 requests per minute with a burst of 5 requests.
func DownloadRateLimiter() echo.MiddlewareFunc {
	return DownloadRateLimiterWithConfig(10.0/60.0, 5, 3*time.Minute)
}

// DownloadRateLimiterWithConfig returns a download rate limiting middleware with custom rate limit settings.
func DownloadRateLimiterWithConfig(r rate.Limit, burst int, expiresIn time.Duration) echo.MiddlewareFunc {
	return createRateLimiter(r, burst, expiresIn, "Too many download requests, please try again later")
}

func routes(e *echo.Echo, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS any, favicon string, internalUrl string, pdfDateFormat string) {
	jwtMiddleware := echojwt.WithConfig(echojwt.Config{
		TokenLookup: "header:Authorization",
		// Custom token parsing to handle with and without bearer prefix to support outdated shaper cli clients that don't send bearer prefix
		ParseTokenFunc: func(c echo.Context, auth string) (any, error) {
			tokenString := strings.TrimSpace(auth)
			if len(tokenString) >= 7 && strings.EqualFold(tokenString[:7], "bearer ") {
				tokenString = strings.TrimSpace(tokenString[7:])
			}
			token, err := jwt.Parse(tokenString, GetJWTKeyfunc(app))
			if err != nil {
				return nil, err
			}
			return token, nil
		},
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
			ok, err := core.ValidateAPIKey(app.Sqlite, c.Request().Context(), key)
			if err != nil {
				return false, err
			}
			if ok {
				c.Set(keyAuthContextKey, key)
			}
			return ok, nil
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
	e.GET("/metrics", echoprometheus.NewHandler(), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor, RequirePermission(app, core.PermissionReadMetrics))

	inviteLimiter := InviteRateLimiter()
	downloadLimiter := DownloadRateLimiter()

	// API routes - no caching
	e.GET("/api/system/config", handler.GetSystemConfig(app))
	e.POST("/api/login", handler.Login(app), LoginRateLimiter())
	e.POST("/api/auth/token", handler.TokenAuth(app))
	e.POST("/api/auth/public", handler.PublicAuth(app), PublicAuthRateLimiter())
	e.POST("/api/auth/setup", handler.Setup(app))
	e.GET("/api/invites/:code", handler.GetInvite(app), inviteLimiter)
	e.POST("/api/invites/:code/claim", handler.ClaimInvite(app), inviteLimiter)
	e.POST("/api/data/:table_name", handler.PostEvent(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor, RequirePermission(app, core.PermissionIngestData))
	e.POST("/api/deploy", handler.Deploy(app), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor), RequirePermission(app, core.PermissionDeploy))
	e.POST("/api/validate", handler.Validate(app), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor), RequirePermission(app, core.PermissionDeploy, core.PermissionDeployDryRun))
	e.POST("/api/sql", handler.ExecuteSQL(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor, RequirePermission(app, core.PermissionQueryData))
	e.GET("/api/schema", handler.GetSchema(app), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor), RequirePermission(app, core.PermissionReadSchema))
	e.POST("/api/download/:filename", handler.DownloadSQL(app, internalUrl, pdfDateFormat), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor, RequirePermission(app, core.PermissionQueryData), downloadLimiter)
	e.GET("/api/apps", handler.ListApps(app), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor), RequirePermission(app, core.PermissionDeploy, core.PermissionDeployDryRun))
	e.GET("/api/public/:id/status", handler.GetDashboardStatus(app))
	apiWithAuth.GET("/version", handler.GetVersion(app))
	apiWithAuth.POST("/logout", handler.Logout(app))
	apiWithAuth.POST("/folders", handler.CreateFolder(app))
	apiWithAuth.DELETE("/folders/:id", handler.DeleteFolder(app))
	apiWithAuth.POST("/folders/:id/name", handler.RenameFolder(app))
	apiWithAuth.POST("/move", handler.MoveItems(app))
	e.POST("/api/dashboards", handler.CreateDashboard(app), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor), RequirePermission(app, core.PermissionQueryData))

	apiWithAuth.GET("/dashboards/:id", handler.GetDashboard(app))
	apiWithAuth.DELETE("/dashboards/:id", handler.DeleteDashboard(app))
	apiWithAuth.GET("/dashboards/:id/info", handler.GetDashboardInfo(app))
	apiWithAuth.POST("/dashboards/:id/query", handler.SaveDashboardQuery(app))
	apiWithAuth.POST("/dashboards/:id/name", handler.SaveDashboardName(app))
	apiWithAuth.POST("/dashboards/:id/visibility", handler.SaveDashboardVisibility(app))
	apiWithAuth.POST("/dashboards/:id/password", handler.SaveDashboardPassword(app))
	// If user auth with JWT generated by a JWT, the api key permission check checks for the `jwt` permission. Makes sense that you can download a dashboard if you can generate a JWT for it.
	e.GET("/api/dashboards/:id/download/:filename", handler.RequestDashboardDownload(app, internalUrl, pdfDateFormat), jwtOrAPIKeyMiddleware(app, jwtMiddleware, SetActor(app), middleware.KeyAuthWithConfig(keyAuthConfig), apiKeyActor), RequirePermission(app, core.PermissionReadDashboard, core.PermissionGenerateJWT), downloadLimiter)
	e.GET("/api/download/:key/:filename", handler.DownloadFileByKey(app, internalUrl, pdfDateFormat), downloadLimiter)
	if !app.NoTasks {
		apiWithAuth.POST("/tasks", handler.CreateTask(app))
		apiWithAuth.GET("/tasks/:id", handler.GetTask(app))
		apiWithAuth.DELETE("/tasks/:id", handler.DeleteTask(app))
		apiWithAuth.POST("/tasks/:id/content", handler.SaveTaskContent(app))
		apiWithAuth.POST("/tasks/:id/name", handler.SaveTaskName(app))
		apiWithAuth.POST("/run/task", handler.RunTask(app))
	}
	apiWithAuth.GET("/users", handler.ListUsers(app))
	apiWithAuth.POST("/users/:id/password", handler.UpdatePassword(app))
	apiWithAuth.POST("/users/:id/name", handler.UpdateName(app))
	apiWithAuth.DELETE("/users/:id", handler.DeleteUser(app))
	apiWithAuth.DELETE("/invites/:code", handler.DeleteInvite(app))
	apiWithAuth.POST("/invites", handler.CreateInvite(app))
	apiWithAuth.GET("/keys", handler.ListAPIKeys(app))
	apiWithAuth.POST("/keys", handler.CreateAPIKey(app))
	apiWithAuth.POST("/keys/:id/permissions", handler.UpdateAPIKeyPermissions(app))
	apiWithAuth.DELETE("/keys/:id", handler.DeleteAPIKey(app))
	apiWithAuth.POST("/admin/reset-jwt-secret", handler.ResetJWTSecret(app))

	// Static assets - aggressive caching
	assetsGroup := e.Group("/assets", CacheControl(CacheConfig{
		MaxAge:    365 * 24 * time.Hour, // 1 year
		Public:    true,
		Immutable: true,
	}))
	assetsGroup.GET("/*", frontend(frontendFS))

	// Unversioned entrypoints - cached with revalidation (ETags)
	revalidateCache := CacheControl(CacheConfig{
		MaxAge:         0,
		Public:         true,
		MustRevalidate: true,
	})

	e.GET("/embed/custom.css", serveCustomCSS(customCSS, modTime), revalidateCache)
	e.GET("/embed/*", serveEmbedJS(frontendFS, modTime), revalidateCache)
	e.GET("/view/:id", serveViewHTML(frontendFS, modTime), revalidateCache)
	e.GET("/_internal/pdfview/:id", servePdfViewHTML(frontendFS, modTime), revalidateCache)

	// Icon - moderate caching
	if IsBasePathSet(app.BasePath) && favicon == "" {
		e.GET("/favicon.ico", func(c echo.Context) error {
			return c.String(http.StatusNotFound, "Not Found")
		})
	} else {
		e.GET("/favicon.ico", serveFavicon(frontendFS, favicon, modTime), CacheControl(CacheConfig{
			MaxAge: 24 * time.Hour, // 1 day
			Public: true,
		}))
	}

	// Index HTML - light caching with revalidation
	e.GET("/*", indexHTMLWithCache(frontendFS, modTime, app.BasePath, favicon))
}

func jwtOrAPIKeyMiddleware(app *core.App, jwtMiddleware echo.MiddlewareFunc, setActorMid echo.MiddlewareFunc, keyAuthMiddleware echo.MiddlewareFunc, apiKeyActorMid echo.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		jwtChain := jwtMiddleware(setActorMid(next))
		apiKeyChain := keyAuthMiddleware(apiKeyActorMid(next))
		return func(c echo.Context) error {
			token := extractAuthorizationToken(c)
			if core.IsAPIKeyToken(token) || (!app.LoginRequired && token == "") {
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

// We overide the Keyfunc handler to handle if our JWT secret changes
func GetJWTKeyfunc(app *core.App) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != echojwt.AlgorithmHS256 {
			return nil, &echojwt.TokenError{Token: token, Err: fmt.Errorf("unexpected jwt signing method=%v", token.Header["alg"])}
		}
		return app.JWTSecret, nil
	}
}
