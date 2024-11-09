package handler

import (
	"net/http"
	"shaper/core"

	"github.com/labstack/echo/v4"
)

func ListDashboards(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		result, err := core.ListDashboards(app, c.Request().Context())
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func GetDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		result, err := core.GetDashboard(app, c.Request().Context(), c.Param("name"), c.QueryParams())
		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}
