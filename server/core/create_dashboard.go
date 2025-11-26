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

type TmpDashboard struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func CreateDashboard(app *App, ctx context.Context, name string, content string, path string, temporary bool) (string, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return "", fmt.Errorf("no actor in context")
	}
	id := cuid2.Generate()
	if temporary {
		key := TMP_DASHBOARD_PREFIX + id
		d := TmpDashboard{
			Name:    name,
			Path:    path,
			Content: content,
		}
		j, err := json.Marshal(d)
		if err != nil {
			return "", err
		}
		_, err = app.TmpDashboardsKv.Put(ctx, key, j)
		return key, err
	}
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("dashboard name cannot be empty")
	}
	// Check for duplicate name in same folder
	folderID, err := ResolveFolderPath(app, ctx, path)
	if err != nil {
		folderID = nil
	}
	var count int
	if folderID == nil {
		err = app.Sqlite.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM apps WHERE name = $1 AND folder_id IS NULL AND type = 'dashboard'`, name)
	} else {
		err = app.Sqlite.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM apps WHERE name = $1 AND folder_id = $2 AND type = 'dashboard'`, name, *folderID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to check for duplicate name: %w", err)
	}
	if count > 0 {
		return "", fmt.Errorf("a dashboard with this name already exists in this folder")
	}
	err = app.SubmitState(ctx, "create_dashboard", CreateDashboardPayload{
		ID:        id,
		Timestamp: time.Now(),
		Path:      path,
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

	folderID, err := ResolveFolderPath(app, context.Background(), payload.Path)
	if err != nil {
		app.Logger.Warn("failed to resolve folder path, creating at root", slog.String("path", payload.Path), slog.Any("error", err))
		folderID = nil
	}

	_, err = app.Sqlite.Exec(
		`INSERT OR IGNORE INTO apps (
			id, folder_id, name, content, created_at, updated_at, created_by, updated_by, type
		) VALUES ($1, $2, $3, $4, $5, $5, $6, $6, 'dashboard')`,
		payload.ID, folderID, payload.Name, payload.Content, payload.Timestamp, payload.CreatedBy,
	)
	if err != nil {
		app.Logger.Error("failed to insert dashboard into DB", slog.Any("error", err))
		return false
	}
	return true
}
