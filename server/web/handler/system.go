// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"net/http"
	"shaper/server/core"

	"github.com/labstack/echo/v4"
)

func GetSystemConfig(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]bool{
			"loginRequired": app.LoginRequired,
			"tasksEnabled":  !app.NoTasks,
		})
	}
}
