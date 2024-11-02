package handler

import (
	"net/http"
	"shaper/core"

	"github.com/labstack/echo/v4"
)

func Sample(app *core.App) func(echo.Context) error {
	return func(c echo.Context) error {
		result, err := core.Sample(app, c.Request().Context())
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error error }{Error: err}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}
