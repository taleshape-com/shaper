package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxAge     time.Duration
	Public     bool
	Immutable  bool
	MustRevali bool
}

// CacheControl middleware adds cache headers based on configuration
func CacheControl(config CacheConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set Cache-Control
			var cacheControl strings.Builder
			if config.Public {
				cacheControl.WriteString("public, ")
			} else {
				cacheControl.WriteString("private, ")
			}
			cacheControl.WriteString(fmt.Sprintf("max-age=%d", int(config.MaxAge.Seconds())))
			if config.Immutable {
				cacheControl.WriteString(", immutable")
			}
			if config.MustRevali {
				cacheControl.WriteString(", must-revalidate")
			}

			// Set cache headers
			c.Response().Header().Set("Cache-Control", cacheControl.String())
			c.Response().Header().Set("Expires", time.Now().Add(config.MaxAge).UTC().Format(http.TimeFormat))

			return next(c)
		}
	}
}
