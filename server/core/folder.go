// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateFolderRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type MoveItemsRequest struct {
	Apps    []string `json:"apps"`
	Folders []string `json:"folders"`
	Path    string   `json:"path"`
}

// resolveFolderPath resolves a folder path to a folder ID
// Returns nil for root path ("/") or empty path
// Supports trailing slashes (e.g., "/Marketing/" is equivalent to "/Marketing")
func resolveFolderPath(app *App, ctx context.Context, path string) (*string, error) {
	// Root path or empty path
	if path == "" || path == "/" {
		return nil, nil
	}

	// Remove leading slash if present
	if path[0] == '/' {
		path = path[1:]
	}

	// Remove trailing slash if present
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// If path is now empty, it was just "/"
	if path == "" {
		return nil, nil
	}

	// Split path into components
	components := []string{}
	for _, component := range strings.Split(path, "/") {
		if component != "" {
			components = append(components, component)
		}
	}

	if len(components) == 0 {
		return nil, nil
	}

	// Walk the path from root
	var currentFolderID *string = nil
	for _, folderName := range components {
		var folderID string
		err := app.Sqlite.GetContext(ctx, &folderID, `
			SELECT id FROM folders WHERE parent_folder_id IS ? AND name = ?
		`, currentFolderID, folderName)
		if err != nil {
			return nil, fmt.Errorf("folder not found in path '%s': %w", path, err)
		}
		currentFolderID = &folderID
	}

	return currentFolderID, nil
}

func CreateFolder(app *App, ctx context.Context, req CreateFolderRequest) (FolderListItem, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return FolderListItem{}, fmt.Errorf("no actor in context")
	}

	// Resolve parent path to folder ID
	parentFolderID, err := resolveFolderPath(app, ctx, req.Path)
	if err != nil {
		return FolderListItem{}, fmt.Errorf("error resolving parent path: %w", err)
	}

	// Check if a folder with the same parent folder and name already exists
	var count int
	err = app.Sqlite.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM folders WHERE parent_folder_id IS ? AND name = ?
	`, parentFolderID, req.Name)
	if err != nil {
		return FolderListItem{}, fmt.Errorf("error checking for duplicate folder: %w", err)
	}
	if count > 0 {
		parentDesc := "root"
		if parentFolderID != nil {
			parentDesc = fmt.Sprintf("path '%s'", req.Path)
		}
		return FolderListItem{}, fmt.Errorf("a folder with the name '%s' already exists in %s", req.Name, parentDesc)
	}

	// Generate a unique ID for the folder
	id := uuid.New().String()
	now := time.Now()

	// Insert the folder into the database
	_, err = app.Sqlite.ExecContext(ctx, `
		INSERT INTO folders (id, parent_folder_id, name, created_at, updated_at, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, parentFolderID, req.Name, now, now, actor.ID, actor.ID)

	if err != nil {
		return FolderListItem{}, fmt.Errorf("error creating folder: %w", err)
	}

	// Return the created folder
	return FolderListItem{
		ID:             id,
		ParentFolderID: parentFolderID,
		Name:           req.Name,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      &actor.ID,
		UpdatedBy:      &actor.ID,
	}, nil
}

func DeleteFolder(app *App, ctx context.Context, id string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	// Delete the folder - CASCADE constraints will automatically delete:
	// - All apps in this folder and subfolders (via folder_id FK)
	// - All subfolders recursively (via parent_folder_id FK)
	result, err := app.Sqlite.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("error deleting folder: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("folder not found")
	}

	// Find orphaned task_runs (those without matching apps) and unschedule them
	var orphanedTaskIDs []string
	err = app.Sqlite.SelectContext(ctx, &orphanedTaskIDs, `
		SELECT task_id FROM task_runs
		WHERE task_id NOT IN (SELECT id FROM apps WHERE type = 'task')
	`)
	if err != nil {
		app.Logger.Error("failed to find orphaned task_runs", slog.Any("error", err))
	} else {
		// Unschedule orphaned tasks
		for _, taskID := range orphanedTaskIDs {
			unscheduleTask(app, taskID)
		}

		// Clean up orphaned task_runs
		if len(orphanedTaskIDs) > 0 {
			_, err = app.Sqlite.ExecContext(ctx, `
				DELETE FROM task_runs
				WHERE task_id NOT IN (SELECT id FROM apps WHERE type = 'task')
			`)
			if err != nil {
				app.Logger.Error("failed to clean up orphaned task_runs", slog.Any("error", err))
			}
		}
	}

	return nil
}

func MoveItems(app *App, ctx context.Context, req MoveItemsRequest) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	// Validate that we have items to move
	if len(req.Apps) == 0 && len(req.Folders) == 0 {
		return fmt.Errorf("no items to move")
	}

	// Resolve destination path to folder ID
	toFolderID, err := resolveFolderPath(app, ctx, req.Path)
	if err != nil {
		return fmt.Errorf("error resolving destination path: %w", err)
	}

	// Start a transaction to ensure atomicity
	tx, err := app.Sqlite.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Move apps
	for _, appID := range req.Apps {
		if appID == "" {
			continue
		}

		result, err := tx.ExecContext(ctx, `
			UPDATE apps
			SET folder_id = ?, updated_at = ?, updated_by = ?
			WHERE id = ?
		`, toFolderID, time.Now(), actor.ID, appID)

		if err != nil {
			return fmt.Errorf("error moving app %s: %w", appID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("error getting rows affected for app %s: %w", appID, err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("app %s not found", appID)
		}
	}

	// Move folders
	for _, folderID := range req.Folders {
		if folderID == "" {
			continue
		}

		// Check for circular references - prevent moving a folder into its own subtree
		if toFolderID != nil {
			var isCircular bool
			err := tx.GetContext(ctx, &isCircular, `
				WITH RECURSIVE folder_ancestors(id) AS (
					SELECT parent_folder_id FROM folders WHERE id = ?
					UNION ALL
					SELECT f.parent_folder_id FROM folders f
					JOIN folder_ancestors fa ON f.id = fa.id
					WHERE f.parent_folder_id IS NOT NULL
				)
				SELECT COUNT(*) > 0 FROM folder_ancestors WHERE id = ?
			`, *toFolderID, folderID)
			if err != nil {
				return fmt.Errorf("error checking for circular reference for folder %s: %w", folderID, err)
			}
			if isCircular {
				return fmt.Errorf("cannot move folder into its own subtree")
			}
		}

		// Update the folder itself
		result, err := tx.ExecContext(ctx, `
			UPDATE folders
			SET parent_folder_id = ?, updated_at = ?, updated_by = ?
			WHERE id = ?
		`, toFolderID, time.Now(), actor.ID, folderID)

		if err != nil {
			return fmt.Errorf("error moving folder %s: %w", folderID, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("error getting rows affected for folder %s: %w", folderID, err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("folder %s not found", folderID)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}
