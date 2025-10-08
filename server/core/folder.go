// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
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

// ResolveFolderPath resolves a folder path to a folder ID
// Returns nil for root path ("/") or empty path
// Supports trailing slashes (e.g., "/Marketing/" is equivalent to "/Marketing")
func ResolveFolderPath(app *App, ctx context.Context, path string) (*string, error) {
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

// ResolveFolderIDToPath resolves a folder ID to its full path
// Returns "/" for root folder (nil folderID) or empty path
func ResolveFolderIDToPath(app *App, ctx context.Context, folderID *string) (string, error) {
	// Root folder
	if folderID == nil {
		return "/", nil
	}

	// Build path by walking up the folder hierarchy
	var pathComponents []string
	currentFolderID := folderID

	for currentFolderID != nil {
		var folderInfo struct {
			Name           string  `db:"name"`
			ParentFolderID *string `db:"parent_folder_id"`
		}
		
		err := app.Sqlite.GetContext(ctx, &folderInfo, `
			SELECT name, parent_folder_id FROM folders WHERE id = ?
		`, *currentFolderID)
		
		if err != nil {
			return "", fmt.Errorf("failed to get folder info: %w", err)
		}
		
		pathComponents = append([]string{folderInfo.Name}, pathComponents...)
		currentFolderID = folderInfo.ParentFolderID
	}

	if len(pathComponents) == 0 {
		return "/", nil
	}

	return "/" + strings.Join(pathComponents, "/") + "/", nil
}

func CreateFolder(app *App, ctx context.Context, req CreateFolderRequest) (FolderListItem, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return FolderListItem{}, fmt.Errorf("no actor in context")
	}

	// Resolve parent path to folder ID
	parentFolderID, err := ResolveFolderPath(app, ctx, req.Path)
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

	// Create the payload and submit to NATS
	payload := CreateFolderPayload{
		ID:             id,
		ParentFolderID: parentFolderID,
		Name:           req.Name,
		Timestamp:      now,
		CreatedBy:      actor.String(),
	}

	err = app.SubmitState(ctx, "create_folder", payload)
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

	// Check if folder exists before submitting the event
	var exists bool
	err := app.Sqlite.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM folders WHERE id = ?)`, id)
	if err != nil {
		return fmt.Errorf("error checking folder existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("folder not found")
	}

	// Create the payload and submit to NATS
	payload := DeleteFolderPayload{
		ID:        id,
		Timestamp: time.Now(),
		DeletedBy: actor.String(),
	}

	err = app.SubmitState(ctx, "delete_folder", payload)
	if err != nil {
		return fmt.Errorf("error deleting folder: %w", err)
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
	toFolderID, err := ResolveFolderPath(app, ctx, req.Path)
	if err != nil {
		return fmt.Errorf("error resolving destination path: %w", err)
	}

	// Validate that all apps and folders exist before submitting the event
	for _, appID := range req.Apps {
		if appID == "" {
			continue
		}
		var exists bool
		err := app.Sqlite.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM apps WHERE id = ?)`, appID)
		if err != nil {
			return fmt.Errorf("error checking app existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("app %s not found", appID)
		}
	}

	for _, folderID := range req.Folders {
		if folderID == "" {
			continue
		}
		var exists bool
		err := app.Sqlite.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM folders WHERE id = ?)`, folderID)
		if err != nil {
			return fmt.Errorf("error checking folder existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("folder %s not found", folderID)
		}

		// Check for circular references - prevent moving a folder into its own subtree
		if toFolderID != nil {
			var isCircular bool
			err := app.Sqlite.GetContext(ctx, &isCircular, `
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

		// Check for duplicate folder names in destination
		var folderName string
		err = app.Sqlite.GetContext(ctx, &folderName, `SELECT name FROM folders WHERE id = ?`, folderID)
		if err != nil {
			return fmt.Errorf("error getting folder name: %w", err)
		}

		var duplicateCount int
		err = app.Sqlite.GetContext(ctx, &duplicateCount, `
			SELECT COUNT(*) FROM folders
			WHERE parent_folder_id IS ? AND name = ? AND id != ?
		`, toFolderID, folderName, folderID)
		if err != nil {
			return fmt.Errorf("error checking for duplicate folder name: %w", err)
		}
		if duplicateCount > 0 {
			destinationDesc := "root"
			if toFolderID != nil {
				destinationDesc = "this location"
			}
			return fmt.Errorf("a folder with the name '%s' already exists in %s", folderName, destinationDesc)
		}
	}

	// Create the payload and submit to NATS
	payload := MoveItemsPayload{
		Apps:       req.Apps,
		Folders:    req.Folders,
		Path:       req.Path,
		ToFolderID: toFolderID,
		Timestamp:  time.Now(),
		MovedBy:    actor.String(),
	}

	err = app.SubmitState(ctx, "move_items", payload)
	if err != nil {
		return fmt.Errorf("error moving items: %w", err)
	}

	return nil
}

type RenameFolderRequest struct {
	Name string `json:"name"`
}

func RenameFolder(app *App, ctx context.Context, id string, req RenameFolderRequest) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	// Check if folder exists
	var exists bool
	err := app.Sqlite.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM folders WHERE id = ?)`, id)
	if err != nil {
		return fmt.Errorf("error checking folder existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("folder not found")
	}

	// Get the folder's parent to check for duplicate names
	var parentFolderID *string
	err = app.Sqlite.GetContext(ctx, &parentFolderID, `SELECT parent_folder_id FROM folders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("error getting folder parent: %w", err)
	}

	// Check if a folder with the same parent folder and name already exists
	var count int
	err = app.Sqlite.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM folders WHERE parent_folder_id IS ? AND name = ? AND id != ?
	`, parentFolderID, req.Name, id)
	if err != nil {
		return fmt.Errorf("error checking for duplicate folder name: %w", err)
	}
	if count > 0 {
		parentDesc := "root"
		if parentFolderID != nil {
			parentDesc = "this location"
		}
		return fmt.Errorf("a folder with the name '%s' already exists in %s", req.Name, parentDesc)
	}

	// Create the payload and submit to NATS
	payload := RenameFolderPayload{
		ID:        id,
		Name:      req.Name,
		Timestamp: time.Now(),
		RenamedBy: actor.String(),
	}

	err = app.SubmitState(ctx, "rename_folder", payload)
	if err != nil {
		return fmt.Errorf("error renaming folder: %w", err)
	}

	return nil
}

// Event payloads for NATS
type CreateFolderPayload struct {
	ID             string    `json:"id"`
	ParentFolderID *string   `json:"parentFolderId"`
	Name           string    `json:"name"`
	Timestamp      time.Time `json:"timestamp"`
	CreatedBy      string    `json:"createdBy"`
}

type DeleteFolderPayload struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	DeletedBy string    `json:"deletedBy"`
}

type MoveItemsPayload struct {
	Apps       []string  `json:"apps"`
	Folders    []string  `json:"folders"`
	Path       string    `json:"path"`
	ToFolderID *string   `json:"toFolderId"`
	Timestamp  time.Time `json:"timestamp"`
	MovedBy    string    `json:"movedBy"`
}

type RenameFolderPayload struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	RenamedBy string    `json:"renamedBy"`
}

// Event handlers
func HandleCreateFolder(app *App, data []byte) bool {
	var payload CreateFolderPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal create folder payload", slog.Any("error", err))
		return false
	}

	_, err = app.Sqlite.Exec(
		`INSERT OR IGNORE INTO folders (
			id, parent_folder_id, name, created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $4, $5, $5)`,
		payload.ID, payload.ParentFolderID, payload.Name, payload.Timestamp, payload.CreatedBy,
	)
	if err != nil {
		app.Logger.Error("failed to insert folder into DB", slog.Any("error", err))
		return false
	}
	return true
}

func HandleDeleteFolder(app *App, data []byte) bool {
	var payload DeleteFolderPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal delete folder payload", slog.Any("error", err))
		return false
	}

	// Delete the folder - CASCADE constraints will automatically delete:
	// - All apps in this folder and subfolders (via folder_id FK)
	// - All subfolders recursively (via parent_folder_id FK)
	result, err := app.Sqlite.Exec(`DELETE FROM folders WHERE id = $1`, payload.ID)
	if err != nil {
		app.Logger.Error("failed to delete folder from DB", slog.Any("error", err))
		return false
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		app.Logger.Error("failed to get rows affected", slog.Any("error", err))
		return false
	}

	if rowsAffected == 0 {
		app.Logger.Warn("folder not found for deletion", slog.String("folderId", payload.ID))
		return true // Not an error, just already deleted
	}

	// Find orphaned task_runs (those without matching apps) and unschedule them
	var orphanedTaskIDs []string
	err = app.Sqlite.Select(&orphanedTaskIDs, `
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
			_, err = app.Sqlite.Exec(`
				DELETE FROM task_runs
				WHERE task_id NOT IN (SELECT id FROM apps WHERE type = 'task')
			`)
			if err != nil {
				app.Logger.Error("failed to clean up orphaned task_runs", slog.Any("error", err))
			}
		}
	}

	return true
}

func HandleMoveItems(app *App, data []byte) bool {
	var payload MoveItemsPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal move items payload", slog.Any("error", err))
		return false
	}

	// Start a transaction to ensure atomicity
	tx, err := app.Sqlite.Beginx()
	if err != nil {
		app.Logger.Error("failed to begin transaction", slog.Any("error", err))
		return false
	}
	defer tx.Rollback()

	// Move apps
	for _, appID := range payload.Apps {
		if appID == "" {
			continue
		}

		result, err := tx.Exec(`
			UPDATE apps
			SET folder_id = $1, updated_at = $2, updated_by = $3
			WHERE id = $4
		`, payload.ToFolderID, payload.Timestamp, payload.MovedBy, appID)

		if err != nil {
			app.Logger.Error("failed to move app", slog.String("appId", appID), slog.Any("error", err))
			return false
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			app.Logger.Error("failed to get rows affected for app", slog.String("appId", appID), slog.Any("error", err))
			return false
		}

		if rowsAffected == 0 {
			app.Logger.Warn("app not found for move", slog.String("appId", appID))
		}
	}

	// Move folders
	for _, folderID := range payload.Folders {
		if folderID == "" {
			continue
		}

		// Check for circular references - prevent moving a folder into its own subtree
		if payload.ToFolderID != nil {
			var isCircular bool
			err := tx.Get(&isCircular, `
				WITH RECURSIVE folder_ancestors(id) AS (
					SELECT parent_folder_id FROM folders WHERE id = $1
					UNION ALL
					SELECT f.parent_folder_id FROM folders f
					JOIN folder_ancestors fa ON f.id = fa.id
					WHERE f.parent_folder_id IS NOT NULL
				)
				SELECT COUNT(*) > 0 FROM folder_ancestors WHERE id = $2
			`, *payload.ToFolderID, folderID)
			if err != nil {
				app.Logger.Error("failed to check for circular reference", slog.String("folderId", folderID), slog.Any("error", err))
				return false
			}
			if isCircular {
				app.Logger.Error("cannot move folder into its own subtree", slog.String("folderId", folderID))
				return false
			}
		}

		// Update the folder itself
		result, err := tx.Exec(`
			UPDATE folders
			SET parent_folder_id = $1, updated_at = $2, updated_by = $3
			WHERE id = $4
		`, payload.ToFolderID, payload.Timestamp, payload.MovedBy, folderID)

		if err != nil {
			app.Logger.Error("failed to move folder", slog.String("folderId", folderID), slog.Any("error", err))
			return false
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			app.Logger.Error("failed to get rows affected for folder", slog.String("folderId", folderID), slog.Any("error", err))
			return false
		}

		if rowsAffected == 0 {
			app.Logger.Warn("folder not found for move", slog.String("folderId", folderID))
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		app.Logger.Error("failed to commit transaction", slog.Any("error", err))
		return false
	}

	return true
}

func HandleRenameFolder(app *App, data []byte) bool {
	var payload RenameFolderPayload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		app.Logger.Error("failed to unmarshal rename folder payload", slog.Any("error", err))
		return false
	}

	_, err = app.Sqlite.Exec(
		`UPDATE folders
		 SET name = $1, updated_at = $2, updated_by = $3
		 WHERE id = $4`,
		payload.Name, payload.Timestamp, payload.RenamedBy, payload.ID,
	)
	if err != nil {
		app.Logger.Error("failed to rename folder in DB", slog.Any("error", err))
		return false
	}
	return true
}
