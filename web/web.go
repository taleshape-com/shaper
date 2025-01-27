// TODO: metrics https://echo.labstack.com/docs/middleware/prometheus
// TODO: rate limit https://echo.labstack.com/docs/middleware/rate-limiter
// TODO: TLS https://echo.labstack.com/docs/cookbook/auto-tls#server
package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"shaper/core"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	slogecho "github.com/samber/slog-echo"
)

func Start(host string, port int, app *core.App, frontendFS fs.FS, modTime time.Time, customCSS string, favicon string) *echo.Echo {
	// Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middlewares
	e.Use(slogecho.New(app.Logger.WithGroup("web")))
	e.Use(middleware.BodyLimit("2M"))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		// Does more bad than good https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection
		XSSProtection:      "",
		ContentTypeNosniff: "nosniff",
		// TODO: In the future we should make this configurable to support embedding the whole app
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

	// Routes
	routes(e, app, frontendFS, modTime, customCSS, favicon)

	// Start server
	addr := fmt.Sprintf("%s:%d", host, port)
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("error starting server", err)
		}
	}()
	app.Logger.Info("HTTP server listening on " + addr)

	return e
}
