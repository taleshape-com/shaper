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
) (*echo.Echo, *http.Server) {
	// Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middlewares
	e.Use(slogecho.New(app.Logger.WithGroup("web")))
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
	routes(e, app, frontendFS, modTime, customCSS, favicon)

	// Configure Let's Encrypt if TLS is enabled
	var httpRedirectServer *http.Server
	if tlsDomain != "" {
		if tlsCacheDir != "" {
			e.AutoTLSManager.Cache = autocert.DirCache(tlsCacheDir)
		}
		if tlsEmail != "" {
			e.AutoTLSManager.Email = tlsEmail
		}
		httpRedirectServer = &http.Server{
			Addr:    ":80",
			Handler: e.AutoTLSManager.HTTPHandler(nil),
		}
	}

	// Start server in background
	go func() {
		if tlsDomain != "" {
			if err := httpRedirectServer.ListenAndServe(); err != http.ErrServerClosed {
				e.Logger.Fatal("Error starting HTTP server", err)
			}
			if err := e.StartAutoTLS(httpsHost + ":433"); err != nil && err != http.ErrServerClosed {
				e.Logger.Fatal("Error starting HTTPS server", err)
			}
		} else {
			if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
				e.Logger.Fatal("Error starting HTTP server", err)
			}
		}
	}()
	if tlsDomain != "" {
		app.Logger.Info("Web server listing on ports 80 and 443 with automatic TLS via letsencrypt")
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

	return e, httpRedirectServer
}
