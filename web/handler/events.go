package handler

import (
	"log/slog"
	"net/http"
	"shaper/core"

	"github.com/labstack/echo/v4"
)

func PostEvent(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var payload map[string]interface{}
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid JSON payload",
			})
		}

		resp, err := core.PublishEvent(c.Request().Context(), app, c.Param("table_name"), payload)
		if err != nil {
			c.Logger().Error("Failed to ingest JSON via HTTP", slog.Any("error", err))
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusAccepted, resp)
	}
}
