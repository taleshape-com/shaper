package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type DeleteDashboardPayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	// TODO: Not used, but might want to log this in the future
	DeletedBy string `json:"deletedBy"`
}

func DeleteDashboard(app *App, ctx context.Context, id string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.dashboards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to query dashboard: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("dashboard not found")
	}
	err = app.SubmitState(ctx, "delete_dashboard", DeleteDashboardPayload{
		ID:        id,
		TimeStamp: time.Now(),
		DeletedBy: actor.String(),
	})
	return err
}

func HandleDeleteDashboard(app *App, data []byte) bool {
	var payload DeleteDashboardPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete dashboard payload", slog.Any("error", err))
		return false
	}
	_, err = app.db.Exec(
		`DELETE FROM `+app.Schema+`.dashboards WHERE id = $1`, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute DELETE statement", slog.Any("error", err))
		return false
	}
	return true
}
