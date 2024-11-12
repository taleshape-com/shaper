package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"shaper/core"
	"strings"

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

// For now only supports .csv
func DownloadQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Validate filename extension
		filename := c.Param("filename")
		if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
			return c.JSONPretty(
				http.StatusBadRequest,
				struct{ Error string }{Error: "filename must have .csv extension"},
				"  ",
			)
		}

		// Set headers for CSV file download
		c.Response().Header().Set(echo.HeaderContentType, "text/csv")
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))

		// Disable response buffering
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		c.Response().Header().Set("Transfer-Encoding", "chunked")

		// Create a writer that writes to the response
		writer := c.Response().Writer

		// Start the streaming query and write directly to response
		err := core.StreamQueryCSV(
			app,
			c.Request().Context(),
			c.Param("name"),
			c.QueryParams(),
			c.Param("query"),
			writer,
		)

		if err != nil {
			// If headers haven't been sent yet, return JSON error
			if c.Response().Committed {
				// If we've already started streaming, log the error since we can't modify the response
				app.Logger.Error("streaming error after response started:", slog.String("error", err.Error()))
				return err
			}
			return c.JSONPretty(
				http.StatusBadRequest,
				struct{ Error string }{Error: err.Error()},
				"  ",
			)
		}

		return nil
	}
}
