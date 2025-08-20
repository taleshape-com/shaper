// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
)

type CreateDashboardPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"createdBy"`
}

func CreateDashboard(app *App, ctx context.Context, name string, content string) (string, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return "", fmt.Errorf("no actor in context")
	}
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("dashboard name cannot be empty")
	}
	id := cuid2.Generate()
	err := app.SubmitState(ctx, "create_dashboard", CreateDashboardPayload{
		ID:        id,
		Timestamp: time.Now(),
		Path:      "/",
		Name:      name,
		Content:   content,
		CreatedBy: actor.String(),
	})
	return id, err
}

func HandleCreateDashboard(app *App, data []byte) bool {
	var payload CreateDashboardPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create dashboard payload", slog.Any("error", err))
		return false
	}
	// Insert into DB
	_, err = app.DB.Exec(
		`INSERT OR IGNORE INTO `+app.Schema+`.apps (
			id, path, name, content, created_at, updated_at, created_by, updated_by, type
		) VALUES ($1, $2, $3, $4, $5, $5, $6, $6, 'dashboard')`,
		payload.ID, payload.Path, payload.Name, payload.Content, payload.Timestamp, payload.CreatedBy,
	)
	if err != nil {
		app.Logger.Error("failed to insert dashboard into DB", slog.Any("error", err))
		return false
	}
	return true
}
