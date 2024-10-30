// TODO: consider https://go-chi.io/
// Chi works with plain http interfaces.
// Need to do TLS myself (see https://github.com/caddyserver/certmagic or https://docs.vultr.com/secure-a-golang-web-server-with-a-selfsigned-or-lets-encrypt-ssl-certificate)
// Need to do JSON encoding myself
// TODO: GZIP https://echo.labstack.com/docs/middleware/gzip
// TODO: metrics https://echo.labstack.com/docs/middleware/prometheus
// TODO: rate limit https://echo.labstack.com/docs/middleware/rate-limiter
// TODO: TLS https://echo.labstack.com/docs/cookbook/auto-tls#server
// TODO: CORS https://echo.labstack.com/docs/cookbook/cors
// TODO: embed and serve frontend https://echo.labstack.com/docs/cookbook/embed-resources
package server

import (
	"duckshape/core"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	slogecho "github.com/samber/slog-echo"
)

func Start(app *core.App) *echo.Echo {
	// Echo instance
	e := echo.New()

	// Middlewares
	e.Use(slogecho.New(app.Logger))
	e.Use(middleware.BodyLimit("2M"))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  log.ERROR,
	}))
	e.Use(middleware.Recover())

	// Routes
	routes(e, app)

	// Start server
	go func() {
		if err := e.Start(":1323"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("error starting server", err)
		}
	}()

	return e
}
