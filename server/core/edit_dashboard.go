// SPDX-License-Identifier: MPL-2.0

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
	UpdatedBy string    `json:"updatedBy"`
}

type UpdateDashboardNamePayload struct {
	ID        string    `json:"id"`
	TimeStamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
	UpdatedBy string    `json:"updatedBy"`
}

type UpdateDashboardVisibilityPayload struct {
	ID         string    `json:"id"`
	TimeStamp  time.Time `json:"timestamp"`
	Visibility string    `json:"visibility"`
	UpdatedBy  string    `json:"updatedBy"`
}

func GetDashboardQuery(app *App, ctx context.Context, id string) (Dashboard, error) {
	var dashboard Dashboard
	err := app.DB.GetContext(ctx, &dashboard,
		`SELECT * EXCLUDE (type) FROM `+app.Schema+`.apps WHERE id = $1 AND type = 'dashboard'`, id)
	if app.NoPublicSharing {
		dashboard.Visibility = nil
	} else if dashboard.Visibility == nil {
		dashboard.Visibility = new(string)
		*dashboard.Visibility = "private"
	}
	if err != nil {
		return dashboard, fmt.Errorf("failed to get dashboard: %w", err)
	}
	return dashboard, nil
}

func SaveDashboardName(app *App, ctx context.Context, id string, name string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.DB.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.apps WHERE id = $1 AND type = 'dashboard'`, id)
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
		UpdatedBy: actor.String(),
	})
	return err
}

func SaveDashboardVisibility(app *App, ctx context.Context, id string, visibility string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.DB.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.apps WHERE id = $1 AND type = 'dashboard'`, id)
	if err != nil {
		return fmt.Errorf("failed to query dashboard: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("dashboard not found")
	}
	if visibility == "" {
		visibility = "private"
	}
	if visibility != "public" && visibility != "private" {
		return fmt.Errorf("invalid visibility value: %s", visibility)
	}
	err = app.SubmitState(ctx, "update_dashboard_visibility", UpdateDashboardVisibilityPayload{
		ID:         id,
		TimeStamp:  time.Now(),
		Visibility: visibility,
		UpdatedBy:  actor.String(),
	})
	return err
}

func SaveDashboardQuery(app *App, ctx context.Context, id string, content string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.DB.GetContext(ctx, &count, `SELECT COUNT(*) FROM `+app.Schema+`.apps WHERE id = $1 AND type = 'dashboard'`, id)
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
		UpdatedBy: actor.String(),
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
	_, err = app.DB.Exec(
		`UPDATE `+app.Schema+`.apps
		 SET content = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'dashboard'`,
		payload.Content, payload.TimeStamp, payload.UpdatedBy, payload.ID)
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
	_, err = app.DB.Exec(
		`UPDATE `+app.Schema+`.apps
		 SET name = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'dashboard'`,
		payload.Name, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}

func HandleUpdateDashboardVisibility(app *App, data []byte) bool {
	var payload UpdateDashboardVisibilityPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update dashboard visibility payload", slog.Any("error", err))
		return false
	}
	visibility := "private"
	if payload.Visibility == "public" {
		visibility = "public"
	}
	fmt.Println(visibility, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	_, err = app.DB.Exec(
		`UPDATE `+app.Schema+`.apps
		 SET visibility = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'dashboard'`,
		visibility, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}
