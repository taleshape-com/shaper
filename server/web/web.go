// SPDX-License-Identifier: MPL-2.0

// TODO: rate limit https://echo.labstack.com/docs/middleware/rate-limiter
// TODO: TLS https://echo.labstack.com/docs/cookbook/auto-tls#server
package web

import (
	"io/fs"
	"log/slog"
	"net/http"
	"shaper/server/core"
	"strings"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	slogecho "github.com/samber/slog-echo"
	"golang.org/x/crypto/acme/autocert"
)

const (
	DEFAULT_HTTP_PORT  = "80"
	DEFAULT_HTTPS_PORT = "443"
)

func Start(
	addr string,
	app *core.App,
	frontendFS fs.FS,
	modTime time.Time,
	customCSS,
	favicon,
	tlsDomain,
	tlsEmail,
	tlsCacheDir,
	httpsHost string,
	pdfDateFormat string,
) *echo.Echo {
	// Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middlewares
	e.Use(slogecho.NewWithFilters(
		app.Logger,
		slogecho.Accept(func(ctx echo.Context) bool {
			path := ctx.Request().URL.Path
			// Always log API requests
			if strings.HasPrefix(path, "/api/") {
				return true
			}
			// Only log non-API requests if debug level is enabled
			return app.Logger.Enabled(ctx.Request().Context(), slog.LevelDebug)
		}),
	))
	e.Use(middleware.BodyLimit("2M"))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		// Does more bad than good: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection
		XSSProtection:      "",
		ContentTypeNosniff: "nosniff",
		// TODO: Make this configurable to support embedding the whole app
		XFrameOptions: "SAMEORIGIN",
		HSTSMaxAge:    2592000, // 30 days
	}))
	// CORS restricted
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		// TODO: Allow to restrict origins via config
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  log.ERROR,
	}))
	// Promethues metrics
	e.Use(echoprometheus.NewMiddleware(app.Name))

	// Routes
	routes(e, app, frontendFS, modTime, customCSS, favicon, getInternalUrl(addr, tlsDomain), pdfDateFormat)

	// Configure Let's Encrypt if TLS is enabled
	if tlsDomain != "" {
		e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(tlsDomain)
		if tlsEmail != "" {
			e.AutoTLSManager.Email = tlsEmail
		}
		if tlsCacheDir != "" {
			e.AutoTLSManager.Cache = autocert.DirCache(tlsCacheDir)
		}
		e.Pre(middleware.HTTPSRedirect())
	}

	// Start server in background
	if tlsDomain != "" {
		go func() {
			if err := e.Start(httpsHost + ":" + DEFAULT_HTTP_PORT); err != http.ErrServerClosed {
				e.Logger.Fatal("Error starting HTTP server", err)
			}
		}()
		go func() {
			if err := e.StartAutoTLS(httpsHost + ":" + DEFAULT_HTTPS_PORT); err != nil && err != http.ErrServerClosed {
				e.Logger.Fatal("Error starting HTTPS server", err)
			}
		}()
	} else {
		go func() {
			if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
				e.Logger.Fatal("Error starting HTTP server", err)
			}
		}()
	}

	if tlsDomain != "" {
		app.Logger.Info("Web server listing on ports " + DEFAULT_HTTP_PORT + " and " + DEFAULT_HTTPS_PORT + " with automatic TLS via letsencrypt")
		app.Logger.Info("Open https://" + tlsDomain + " in your browser")
	} else {
		app.Logger.Info("Web server is listening at " + addr + "")
		if app.BasePath == "/" {
			logPrefix := ""
			if !strings.HasPrefix(addr, ":") {
				logPrefix = "http://"
			}
			app.Logger.Info("Open " + logPrefix + addr + " in your browser")
		} else {
			app.Logger.Info("Custom base path set. Opening the app in your browser directly won't work as expected. You are likely using a reverse proxy and need to access the app through it.", slog.Any("basepath", app.BasePath))
		}
	}

	return e
}

func getInternalUrl(addr, tlsDomain string) string {
	if tlsDomain != "" {
		// When running TLS server, go via TLS domain instead of localhost
		return "https://" + tlsDomain + ":" + DEFAULT_HTTPS_PORT
	}
	internalAddr := addr
	if strings.HasPrefix(internalAddr, "0.0.0.0") {
		internalAddr = strings.Replace(internalAddr, "0.0.0.0", "localhost", 1)
	} else if strings.HasPrefix(internalAddr, "[::]") {
		internalAddr = strings.Replace(internalAddr, "[::]", "localhost", 1)
	} else if strings.HasPrefix(internalAddr, ":") {
		internalAddr = "localhost" + internalAddr
	}
	return "http://" + internalAddr
}
