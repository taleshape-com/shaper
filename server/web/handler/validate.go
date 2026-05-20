// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"shaper/server/core"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type ValidateRequest struct {
	Type string `json:"type"`
	SQL  string `json:"sql"`
}

type ValidateResponse struct {
	Valid    bool   `json:"valid"`
	Duration int64  `json:"duration"`
	Error    string `json:"error,omitempty"`
}

func Validate(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctxUser := c.Get("user")
		if ctxUser != nil {
			claims := ctxUser.(*jwt.Token).Claims.(jwt.MapClaims)
			if _, hasId := claims["dashboardId"]; hasId {
				return c.JSONPretty(http.StatusUnauthorized, struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
			}
		}

		var req ValidateRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		if req.Type == "task" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Task validation is currently not supported"})
		}

		if req.Type != "dashboard" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid type. Must be 'dashboard' or 'task'"})
		}

		start := time.Now()
		_, err := core.QueryDashboard(app, c.Request().Context(), core.DashboardQuery{
			Content: req.SQL,
			ID:      "validate",
		}, nil, nil)
		duration := time.Since(start).Milliseconds()

		if err != nil {
			return c.JSON(http.StatusOK, ValidateResponse{
				Valid:    false,
				Error:    err.Error(),
				Duration: duration,
			})
		}

		return c.JSON(http.StatusOK, ValidateResponse{
			Valid:    true,
			Duration: duration,
		})
	}
}
