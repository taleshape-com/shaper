// TODO: https://echo.labstack.com/docs/middleware/secure
// TODO: GZIP https://echo.labstack.com/docs/middleware/gzip
// TODO: metrics https://echo.labstack.com/docs/middleware/prometheus
// TODO: rate limit https://echo.labstack.com/docs/middleware/rate-limiter
// TODO: TLS https://echo.labstack.com/docs/cookbook/auto-tls#server
package server

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

func Start(host string, port int, app *core.App, frontendFS fs.FS, modTime time.Time) *echo.Echo {
	// Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middlewares
	e.Use(slogecho.New(app.Logger))
	e.Use(middleware.BodyLimit("2M"))
	// CORS restricted
	// Allows requests from any `https://labstack.com` or `https://labstack.net` origin
	// wth GET, PUT, POST or DELETE method.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  log.ERROR,
	}))

	// Routes
	routes(e, app, frontendFS, modTime)

	// Start server
	go func() {
		if err := e.Start(fmt.Sprintf("%s:%d", host, port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("error starting server", err)
		}
	}()

	return e
}
