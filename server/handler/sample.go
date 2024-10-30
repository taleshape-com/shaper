package handler

import (
	"duckshape/core"
	"net/http"

	"github.com/labstack/echo/v4"
)

func Sample(app *core.App) func(echo.Context) error {
	return func(c echo.Context) error {
		results, err := core.Sample(app, c.Request().Context())
		if err != nil {
			return err
		}
		return c.JSONPretty(http.StatusOK, results, "  ")
	}
}
