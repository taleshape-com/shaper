// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"shaper/server/core"
	"shaper/server/pdf"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func CreateDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Invalid request"}, "  ")
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
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
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
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		dashboard, err := core.GetDashboardQuery(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error getting dashboard query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, dashboard)
	}
}

func SaveDashboardName(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Dashboard name is required"}, "  ")
		}

		err := core.SaveDashboardName(app, c.Request().Context(), c.Param("id"), request.Name)
		if err != nil {
			c.Logger().Error("error saving dashboard name:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveDashboardVisibility(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Visibility string `json:"visibility"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		err := core.SaveDashboardVisibility(app, c.Request().Context(), c.Param("id"), request.Visibility)
		if err != nil {
			c.Logger().Error("error saving visibility:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveDashboardPassword(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Password string `json:"password"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		if request.Password == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Password is required"}, "  ")
		}

		err := core.SaveDashboardPassword(app, c.Request().Context(), c.Param("id"), request.Password)
		if err != nil {
			c.Logger().Error("error saving dashboard password:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveDashboardQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		// Embedding JWTs that are fixed to a dashboardId are not allowed to edit the board
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		err := core.SaveDashboardQuery(app, c.Request().Context(), c.Param("id"), request.Content)
		if err != nil {
			c.Logger().Error("error saving dashboard query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func GetDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		idParam := c.Param("id")
		if id, hasId := claims["dashboardId"]; hasId && id != idParam {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]any{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]any)
		}
		result, err := core.GetDashboard(app, c.Request().Context(), idParam, c.QueryParams(), variables)
		if err != nil {
			c.Logger().Error("error getting dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func DeleteDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}

		err := core.DeleteDashboard(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error deleting dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

// Supports .csv and .xlsx
func DownloadQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if id, hasId := claims["dashboardId"]; hasId && id != c.Param("id") {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]any{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]any)
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
					struct {
						Error string `json:"error"`
					}{Error: err.Error()},
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
					struct {
						Error string `json:"error"`
					}{Error: err.Error()},
					"  ",
				)
			}

			return nil
		}

		return c.JSONPretty(
			http.StatusBadRequest,
			struct {
				Error string `json:"error"`
			}{Error: "Invalid filename extension. Must be .csv or .xlsx"},
			"  ",
		)
	}
}

func DownloadPdf(app *core.App, internalUrl string) echo.HandlerFunc {
	return func(c echo.Context) error {
		jwtToken := c.Get("user").(*jwt.Token)
		claims := jwtToken.Claims.(jwt.MapClaims)
		if id, hasId := claims["dashboardId"]; hasId && id != c.Param("id") {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]any{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]any)
		}
		// Validate filename extension
		filename := c.Param("filename")

		c.Response().Header().Set(echo.HeaderContentType, "application/pdf")
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))

		// Disable response buffering
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		c.Response().Header().Set("Transfer-Encoding", "chunked")

		// Create a writer that writes to the response
		writer := c.Response().Writer

		// Start the streaming query and write directly to response
		err := pdf.StreamDashboardPdf(
			c.Request().Context(),
			app.Logger,
			writer,
			internalUrl,
			c.Param("id"),
			c.QueryParams(),
			variables,
			jwtToken,
		)

		if err != nil {
			// If headers haven't been sent yet, return JSON error
			if c.Response().Committed {
				// If we've already started streaming, log the error since we can't modify the response
				c.Logger().Error("streaming error after response started:", slog.Any("error", err))
				return err
			}
			c.Logger().Error("error downloading PDF:", slog.Any("error", err))
			return c.JSONPretty(
				http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()},
				"  ",
			)
		}

		return nil
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
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]any{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]any)
		}

		result, err := core.QueryDashboard(app, c.Request().Context(), core.DashboardQuery{
			Content: request.Content,
			ID:      request.DashboardId,
			Name:    "Preview",
		}, c.QueryParams(), variables)

		if err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func GetDashboardStatus(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		dashboard, err := core.GetDashboardQuery(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error getting dashboard status:", slog.Any("error", err))
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "not found"}, "  ")
		}
		if dashboard.Visibility == nil ||
			(*dashboard.Visibility == "private") ||
			(app.NoPublicSharing && *dashboard.Visibility == "public") ||
			(app.NoPasswordProtectedSharing && *dashboard.Visibility == "password-protected") {
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "not found"}, "  ")
		}
		return c.JSON(http.StatusOK, struct {
			Visibility string `json:"visibility"`
		}{
			Visibility: *dashboard.Visibility,
		})
	}
}
