package handler

import (
	"net/http"
	"shaper/core"

	"github.com/labstack/echo/v4"
)

func GetDashboard(app *core.App) func(echo.Context) error {
	return func(c echo.Context) error {
		result, err := core.GetDashboard(app, c.Request().Context(), c.Param("name"))
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error error }{Error: err}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}
