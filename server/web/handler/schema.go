// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"shaper/server/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func GetSchema(app *core.App) echo.HandlerFunc {
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
		res, err := app.GetSchema(c.Request().Context())
		if err != nil {
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}
		return c.JSONPretty(http.StatusOK, res, "  ")
	}
}
