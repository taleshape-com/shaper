package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"shaper/core"

	"github.com/labstack/echo/v4"
)

type SingleEventResponse struct {
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
}

type MultiEventResponse struct {
	Status   string   `json:"status"`
	IDs      []string `json:"ids,omitempty"`
	Accepted []string `json:"accepted,omitempty"`
	Error    string   `json:"error,omitempty"`
}

func PostEvent(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var rawJSON json.RawMessage
		if err := c.Bind(&rawJSON); err != nil {
			return c.JSON(http.StatusBadRequest, SingleEventResponse{
				Status: "error",
				Error:  "Invalid JSON payload",
			})
		}

		// Try to unmarshal as array first
		var arrayPayload []map[string]any
		if err := json.Unmarshal(rawJSON, &arrayPayload); err == nil {
			// Handle array of events
			ids, err := core.PublishEvents(c.Request().Context(), app, c.Param("table_name"), arrayPayload)
			if err != nil {
				c.Logger().Error("Failed to ingest JSON array via HTTP", slog.Any("error", err))
				return c.JSON(http.StatusPartialContent, MultiEventResponse{
					Status:   "error",
					Accepted: ids, // Contains successfully processed IDs before error
					Error:    err.Error(),
				})
			}
			return c.JSON(http.StatusAccepted, MultiEventResponse{
				Status: "ok",
				IDs:    ids,
			})
		}

		// Try as single object
		var singlePayload map[string]any
		if err := json.Unmarshal(rawJSON, &singlePayload); err != nil {
			return c.JSON(http.StatusBadRequest, SingleEventResponse{
				Status: "error",
				Error:  "Invalid JSON payload: must be object or array of objects",
			})
		}

		// Handle single event
		id, err := core.PublishEvent(c.Request().Context(), app, c.Param("table_name"), singlePayload)
		if err != nil {
			c.Logger().Error("Failed to ingest JSON via HTTP", slog.Any("error", err))
			return c.JSON(http.StatusBadRequest, SingleEventResponse{
				Status: "error",
				Error:  err.Error(),
			})
		}

		return c.JSON(http.StatusAccepted, SingleEventResponse{
			Status: "ok",
			ID:     id,
		})
	}
}
