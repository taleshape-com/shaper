// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
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
	To      string   `json:"to"`
}

func CreateFolder(app *App, ctx context.Context, req CreateFolderRequest) (FolderListItem, error) {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return FolderListItem{}, fmt.Errorf("no actor in context")
	}

	// Check if a folder with the same path and name already exists
	var count int
	err := app.Sqlite.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM folders WHERE path = ? AND name = ?
	`, req.Path, req.Name)
	if err != nil {
		return FolderListItem{}, fmt.Errorf("error checking for duplicate folder: %w", err)
	}
	if count > 0 {
		return FolderListItem{}, fmt.Errorf("a folder with the name '%s' already exists in path '%s'", req.Name, req.Path)
	}

	// Generate a unique ID for the folder
	id := uuid.New().String()
	now := time.Now()

	// Insert the folder into the database
	_, err = app.Sqlite.ExecContext(ctx, `
		INSERT INTO folders (id, path, name, created_at, updated_at, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, req.Path, req.Name, now, now, actor.ID, actor.ID)

	if err != nil {
		return FolderListItem{}, fmt.Errorf("error creating folder: %w", err)
	}

	// Return the created folder
	return FolderListItem{
		ID:        id,
		Path:      req.Path,
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: &actor.ID,
		UpdatedBy: &actor.ID,
	}, nil
}

func ListFolders(app *App, ctx context.Context, sort string, order string) (FolderListResponse, error) {
	var orderBy string
	switch sort {
	case "created":
		orderBy = "created_at"
	case "name":
		orderBy = "name"
	default:
		orderBy = "updated_at"
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	dbFolders := []FolderDbRecord{}
	err := app.Sqlite.SelectContext(ctx, &dbFolders,
		fmt.Sprintf(`SELECT
			id,
			path,
			name,
			created_at,
			updated_at,
			created_by,
			updated_by
			FROM folders
			ORDER BY %s %s`, orderBy, order))
	if err != nil {
		err = fmt.Errorf("error listing folders: %w", err)
	}

	folders := make([]FolderListItem, len(dbFolders))
	for i, f := range dbFolders {
		folders[i] = FolderListItem{
			ID:        f.ID,
			Path:      f.Path,
			Name:      f.Name,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
			CreatedBy: f.CreatedBy,
			UpdatedBy: f.UpdatedBy,
		}
	}

	return FolderListResponse{Folders: folders}, err
}

func DeleteFolder(app *App, ctx context.Context, id string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return fmt.Errorf("no actor in context")
	}

	// Delete the folder from the database
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

	// Validate destination path
	if req.To == "" {
		return fmt.Errorf("destination path is required")
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
			SET path = ?, updated_at = ?, updated_by = ?
			WHERE id = ?
		`, req.To, time.Now(), actor.ID, appID)
		
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
		
		// Get the current folder path to update child paths
		var currentPath string
		err := tx.GetContext(ctx, &currentPath, `SELECT path FROM folders WHERE id = ?`, folderID)
		if err != nil {
			return fmt.Errorf("error getting current path for folder %s: %w", folderID, err)
		}
		
		// Update the folder itself
		result, err := tx.ExecContext(ctx, `
			UPDATE folders 
			SET path = ?, updated_at = ?, updated_by = ?
			WHERE id = ?
		`, req.To, time.Now(), actor.ID, folderID)
		
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
		
		// Update all child folders and apps that are in subfolders
		// This updates paths like "/old/path/subfolder" to "/new/path/subfolder"
		_, err = tx.ExecContext(ctx, `
			UPDATE folders 
			SET path = REPLACE(path, ?, ?), updated_at = ?, updated_by = ?
			WHERE path LIKE ? AND id != ?
		`, currentPath, req.To, time.Now(), actor.ID, currentPath+"/%", folderID)
		
		if err != nil {
			return fmt.Errorf("error updating child folder paths for folder %s: %w", folderID, err)
		}
		
		// Update all child apps that are in subfolders
		_, err = tx.ExecContext(ctx, `
			UPDATE apps 
			SET path = REPLACE(path, ?, ?), updated_at = ?, updated_by = ?
			WHERE path LIKE ?
		`, currentPath, req.To, time.Now(), actor.ID, currentPath+"/%")
		
		if err != nil {
			return fmt.Errorf("error updating child app paths for folder %s: %w", folderID, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}