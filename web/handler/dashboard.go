package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"shaper/core"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func ListDashboards(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}
		sort := c.QueryParam("sort")
		order := c.QueryParam("order")
		result, err := core.ListDashboards(app, c.Request().Context(), sort, order)
		if err != nil {
			c.Logger().Error("error listing dashboards:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func CreateDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct{ Error string }{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct{ Error string }{Error: "Invalid request"}, "  ")
		}

		// Validate dashboard name
		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Dashboard name is required"}, "  ")
		}

		id, err := core.CreateDashboard(app, c.Request().Context(), request.Name, request.Content)
		if err != nil {
			c.Logger().Error("error creating dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct{ Error string }{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusCreated, struct {
			ID string `json:"id"`
		}{
			ID: id,
		}, "  ")
	}
}

func GetDashboardQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		// Embedding JWTs that are fixed to a dashboardId are not allowed to edit the board
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}

		dashboard, err := core.GetDashboardQuery(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error getting dashboard query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, dashboard)
	}
}

func SaveDashboardName(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: "Invalid request"}, "  ")
		}

		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: "Dashboard name is required"}, "  ")
		}

		err := core.SaveDashboardName(app, c.Request().Context(), c.Param("id"), request.Name)
		if err != nil {
			c.Logger().Error("error saving dashboard name:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}

		return c.NoContent(http.StatusOK)
	}
}

func SaveDashboardQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		// Embedding JWTs that are fixed to a dashboardId are not allowed to edit the board
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: "Invalid request"}, "  ")
		}

		err := core.SaveDashboardQuery(app, c.Request().Context(), c.Param("id"), request.Content)
		if err != nil {
			c.Logger().Error("error saving dashboard query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}

		return c.NoContent(http.StatusOK)
	}
}

func GetDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if id, hasId := claims["dashboardId"]; hasId && id != c.Param("id") {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]interface{}{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]interface{})
		}
		result, err := core.GetDashboard(app, c.Request().Context(), c.Param("id"), c.QueryParams(), variables)
		if err != nil {
			c.Logger().Error("error getting dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct{ Error string }{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func DeleteDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct{ Error string }{Error: "Unauthorized"}, "  ")
		}

		err := core.DeleteDashboard(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error deleting dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct{ Error string }{Error: err.Error()}, "  ")
		}

		return c.NoContent(http.StatusOK)
	}
}

// Supports .csv and .xlsx
func DownloadQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if id, hasId := claims["dashboardId"]; hasId && id != c.Param("id") {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]interface{}{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]interface{})
		}
		// Validate filename extension
		filename := c.Param("filename")

		if strings.HasSuffix(strings.ToLower(filename), ".csv") {
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
				c.Param("id"),
				c.QueryParams(),
				c.Param("query"),
				variables,
				writer,
			)

			if err != nil {
				// If headers haven't been sent yet, return JSON error
				if c.Response().Committed {
					// If we've already started streaming, log the error since we can't modify the response
					c.Logger().Error("streaming error after response started:", slog.Any("error", err))
					return err
				}
				c.Logger().Error("error downloading CSV:", slog.Any("error", err))
				return c.JSONPretty(
					http.StatusBadRequest,
					struct{ Error string }{Error: err.Error()},
					"  ",
				)
			}

			return nil
		}

		if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
			// Set headers for Excel file download
			c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))

			// Disable response buffering
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("Transfer-Encoding", "chunked")

			// Create a writer that writes to the response
			writer := c.Response().Writer

			err := core.StreamQueryXLSX(
				app,
				c.Request().Context(),
				c.Param("id"),
				c.QueryParams(),
				c.Param("query"),
				variables,
				writer,
			)

			if err != nil {
				// If headers haven't been sent yet, return JSON error
				if c.Response().Committed {
					// If we've already started streaming, log the error since we can't modify the response
					c.Logger().Error("streaming error after response started:", slog.Any("error", err))
					return err
				}
				c.Logger().Error("error downloading .xlsx file:", slog.Any("error", err))
				return c.JSONPretty(
					http.StatusBadRequest,
					struct{ Error string }{Error: err.Error()},
					"  ",
				)
			}

			return nil
		}

		return c.JSONPretty(
			http.StatusBadRequest,
			struct{ Error string }{Error: "Invalid filename extension. Must be .csv or .xlsx"},
			"  ",
		)
	}
}

func PreviewDashboardQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var request struct {
			DashboardId string `json:"dashboardId"`
			Content     string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct{ Error string }{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]interface{}{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]interface{})
		}

		result, err := core.QueryDashboard(app, c.Request().Context(), core.DashboardQuery{
			Content: request.Content,
			ID:      request.DashboardId,
		}, c.QueryParams(), variables)

		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}
