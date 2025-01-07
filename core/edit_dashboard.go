package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type UpdateDashboardContentPayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}

type UpdateDashboardNamePayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
}

func GetDashboardQuery(app *App, ctx context.Context, id string) (Dashboard, error) {
	var dashboard Dashboard
	err := app.db.GetContext(ctx, &dashboard,
		`SELECT * FROM `+app.Schema+`.dashboards WHERE id = $1`, id)
	if err != nil {
		return dashboard, fmt.Errorf("failed to get dashboard: %w", err)
	}
	return dashboard, nil
}

func SaveDashboardName(app *App, ctx context.Context, id string, name string) error {
	var count int
	err := app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.dashboards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to query dashboard: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("dashboard not found")
	}
	err = app.SubmitState(ctx, "update_dashboard_name", UpdateDashboardNamePayload{
		ID:        id,
		TimeStamp: time.Now(),
		Name:      name,
	})
	return err
}

func SaveDashboardQuery(app *App, ctx context.Context, id string, content string) error {
	var count int
	err := app.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.dashboards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to query dashboard: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("dashboard not found")
	}
	err = app.SubmitState(ctx, "update_dashboard_content", UpdateDashboardContentPayload{
		ID:        id,
		TimeStamp: time.Now(),
		Content:   content,
	})
	return err
}

func HandleUpdateDashboardContent(app *App, data []byte) bool {
	var payload UpdateDashboardContentPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update dashboard content payload", slog.Any("error", err))
		return false
	}
	_, err = app.db.Exec(
		`UPDATE `+app.Schema+`.dashboards
		 SET content = $1, updated_at = $2
		 WHERE id = $3`,
		payload.Content, payload.TimeStamp, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}

func HandleUpdateDashboardName(app *App, data []byte) bool {
	var payload UpdateDashboardNamePayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update dashboard name payload", slog.Any("error", err))
		return false
	}
	_, err = app.db.Exec(
		`UPDATE `+app.Schema+`.dashboards
		 SET name = $1, updated_at = $2
		 WHERE id = $3`,
		payload.Name, payload.TimeStamp, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}
