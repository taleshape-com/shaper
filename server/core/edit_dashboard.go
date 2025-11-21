// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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

type UpdateDashboardPasswordPayload struct {
	ID           string    `json:"id"`
	TimeStamp    time.Time `json:"timestamp"`
	PasswordHash string    `json:"passwordHash"`
	UpdatedBy    string    `json:"updatedBy"`
}

func GetDashboardInfo(app *App, ctx context.Context, id string) (Dashboard, error) {
	if strings.HasPrefix(id, TMP_DASHBOARD_PREFIX) {
		entry, err := app.TmpDashboardsKv.Get(ctx, id)
		if err != nil {
			return Dashboard{}, fmt.Errorf("failed to get dashboard: %w", err)
		}
		var d TmpDashboard
		json.Unmarshal(entry.Value(), &d)
		visibility := "private"
		dashboard := Dashboard{
			ID:         id,
			CreatedAt:  entry.Created(),
			UpdatedAt:  entry.Created(),
			Visibility: &visibility,
			Name:       d.Name,
			Path:       d.Path,
			Content:    d.Content,
		}
		return dashboard, nil
	}
	var dashboard Dashboard
	err := app.Sqlite.GetContext(ctx, &dashboard,
		`SELECT id, folder_id, name, content, created_at, updated_at, created_by, updated_by, visibility
		FROM apps
		WHERE id = $1 AND type = 'dashboard'`, id)
	if err != nil {
		return dashboard, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Resolve folder_id to path
	path, err := ResolveFolderIDToPath(app, ctx, dashboard.FolderID)
	if err != nil {
		// If path resolution fails, default to root
		app.Logger.Warn("failed to resolve folder ID to path, defaulting to root", slog.Any("folder_id", dashboard.FolderID), slog.Any("error", err))
		dashboard.Path = "/"
	} else {
		dashboard.Path = path
	}

	if dashboard.Visibility == nil {
		dashboard.Visibility = new(string)
		*dashboard.Visibility = "private"
	} else if app.NoPublicSharing && *dashboard.Visibility == "public" {
		dashboard.Visibility = nil
	} else if app.NoPasswordProtectedSharing && *dashboard.Visibility == "password-protected" {
		dashboard.Visibility = nil
	}

	return dashboard, nil
}

func SaveDashboardName(app *App, ctx context.Context, id string, name string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'dashboard'`, id)
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
	if err != nil {
		return fmt.Errorf("failed to submit dashboard name update: %w", err)
	}
	return nil
}

func SaveDashboardVisibility(app *App, ctx context.Context, id string, visibility string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'dashboard'`, id)
	if err != nil {
		return fmt.Errorf("failed to query dashboard: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("dashboard not found")
	}
	if visibility == "" {
		visibility = "private"
	}
	if visibility != "public" && visibility != "private" && visibility != "password-protected" {
		return fmt.Errorf("invalid visibility value: %s", visibility)
	}
	if app.NoPublicSharing && visibility == "public" {
		return fmt.Errorf("Public dashboard sharing is disabled")
	}
	if app.NoPasswordProtectedSharing && visibility == "password-protected" {
		return fmt.Errorf("Password-protected dashboard sharing is disabled")
	}
	err = app.SubmitState(ctx, "update_dashboard_visibility", UpdateDashboardVisibilityPayload{
		ID:         id,
		TimeStamp:  time.Now(),
		Visibility: visibility,
		UpdatedBy:  actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit dashboard visibility update: %w", err)
	}
	return nil
}

func SaveDashboardQuery(app *App, ctx context.Context, id string, content string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'dashboard'`, id)
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
	if err != nil {
		return fmt.Errorf("failed to submit dashboard content update: %w", err)
	}
	return nil
}

func SaveDashboardPassword(app *App, ctx context.Context, id string, password string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `SELECT COUNT(*) FROM apps WHERE id = $1 AND type = 'dashboard'`, id)
	if err != nil {
		return fmt.Errorf("failed to query dashboard: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("dashboard not found")
	}
	// Hash the password using bcrypt
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	err = app.SubmitState(ctx, "update_dashboard_password", UpdateDashboardPasswordPayload{
		ID:           id,
		TimeStamp:    time.Now(),
		PasswordHash: string(passwordHash),
		UpdatedBy:    actor.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to submit dashboard password update: %w", err)
	}
	return nil
}

func VerifyDashboardPassword(app *App, ctx context.Context, id string, password string) (bool, error) {
	var passwordHash string
	err := app.Sqlite.GetContext(ctx, &passwordHash,
		`SELECT password_hash FROM apps WHERE id = $1 AND type = 'dashboard' AND password_hash IS NOT NULL`, id)
	if err != nil {
		return false, fmt.Errorf("failed to get dashboard password hash: %w", err)
	}

	// Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil // Invalid password, but not an error
		}
		return false, fmt.Errorf("failed to verify password: %w", err)
	}

	return true, nil
}

func HandleUpdateDashboardContent(app *App, data []byte) bool {
	var payload UpdateDashboardContentPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update dashboard content payload", slog.Any("error", err))
		return false
	}
	_, err = app.Sqlite.Exec(
		`UPDATE apps
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
	_, err = app.Sqlite.Exec(
		`UPDATE apps
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
	} else if payload.Visibility == "password-protected" {
		visibility = "password-protected"
	}
	_, err = app.Sqlite.Exec(
		`UPDATE apps
		 SET visibility = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'dashboard'`,
		visibility, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}

func HandleUpdateDashboardPassword(app *App, data []byte) bool {
	var payload UpdateDashboardPasswordPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal update dashboard password payload", slog.Any("error", err))
		return false
	}
	_, err = app.Sqlite.Exec(
		`UPDATE apps
		 SET password_hash = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4 AND type = 'dashboard'`,
		payload.PasswordHash, payload.TimeStamp, payload.UpdatedBy, payload.ID)
	if err != nil {
		app.Logger.Error("failed to execute UPDATE statement", slog.Any("error", err))
		return false
	}
	return true
}
