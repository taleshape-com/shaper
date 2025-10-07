// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"time"
)

type AppListItem struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	FolderID   *string   `json:"folderId,omitempty"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CreatedBy  *string   `json:"createdBy,omitempty"`
	UpdatedBy  *string   `json:"updatedBy,omitempty"`
	Visibility *string   `json:"visibility,omitempty"`
	TaskInfo   *TaskInfo `json:"taskInfo,omitempty"`
	Type       string    `json:"type"`
}

type AppDbRecord struct {
	ID              string     `db:"id"`
	Path            string     `db:"path"`
	FolderID        *string    `db:"folder_id"`
	Name            string     `db:"name"`
	Content         string     `db:"content"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	CreatedBy       *string    `db:"created_by"`
	UpdatedBy       *string    `db:"updated_by"`
	Visibility      *string    `db:"visibility"`
	Type            string     `db:"type"`
	LastRunAt       *time.Time `db:"last_run_at"`
	LastRunSuccess  *bool      `db:"last_run_success"`
	LastRunDuration *int64     `db:"last_run_duration"`
	NextRunAt       *time.Time `db:"next_run_at"`
}

type TaskInfo struct {
	LastRunAt       *time.Time `json:"lastRunAt,omitempty"`
	LastRunSuccess  *bool      `json:"lastRunSuccess,omitempty"`
	LastRunDuration *int64     `json:"lastRunDuration,omitempty"`
	NextRunAt       *time.Time `json:"nextRunAt,omitempty"`
}

type AppListResponse struct {
	Apps []AppListItem `json:"apps"`
}

type FolderListItem struct {
	ID             string    `json:"id"`
	Path           string    `json:"path"`
	ParentFolderID *string   `json:"parentFolderId,omitempty"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	CreatedBy      *string   `json:"createdBy,omitempty"`
	UpdatedBy      *string   `json:"updatedBy,omitempty"`
}

type FolderDbRecord struct {
	ID             string    `db:"id"`
	Path           string    `db:"path"`
	ParentFolderID *string   `db:"parent_folder_id"`
	Name           string    `db:"name"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
	CreatedBy      *string   `db:"created_by"`
	UpdatedBy      *string   `db:"updated_by"`
}

type FolderListResponse struct {
	Folders []FolderListItem `json:"folders"`
}

func ListApps(app *App, ctx context.Context, sort string, order string, path string) (AppListResponse, error) {
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

	dbApps := []AppDbRecord{}

	// Find folder_id from path using recursive CTE
	var folderIDFilter string
	var folderIDArgs []interface{}

	if path == "/" || path == "" {
		// Root level - items with folder_id = NULL
		folderIDFilter = "WHERE a.folder_id IS NULL"
	} else {
		// Non-root level - need to find folder ID from path
		folderIDFilter = `WHERE a.folder_id = (
			WITH RECURSIVE folder_path(id, parent_folder_id, name, path) AS (
				SELECT id, parent_folder_id, name, '/' || name || '/' as path
				FROM folders
				WHERE parent_folder_id IS NULL

				UNION ALL

				SELECT f.id, f.parent_folder_id, f.name, fp.path || f.name || '/' as path
				FROM folders f
				JOIN folder_path fp ON f.parent_folder_id = fp.id
			)
			SELECT id FROM folder_path WHERE path = ?
		)`
		folderIDArgs = append(folderIDArgs, path)
	}

	// Build type filter
	typeFilter := ""
	if app.NoTasks {
		typeFilter = " AND a.type = 'dashboard'"
	}

	// Combine filters
	whereClause := folderIDFilter + typeFilter

	// Build the query with recursive CTE for folder paths
	query := fmt.Sprintf(`
		WITH RECURSIVE folder_path(id, parent_folder_id, name, path) AS (
			SELECT id, parent_folder_id, name, '/' || name || '/' as path
			FROM folders
			WHERE parent_folder_id IS NULL

			UNION ALL

			SELECT f.id, f.parent_folder_id, f.name, fp.path || f.name || '/' as path
			FROM folders f
			JOIN folder_path fp ON f.parent_folder_id = fp.id
		)
		SELECT
			a.id,
			COALESCE(fp.path, '/') as path,
			a.folder_id,
			a.name,
			a.content,
			a.created_at,
			a.updated_at,
			a.created_by,
			a.updated_by,
			a.visibility,
			a.type,
			t.last_run_at,
			t.last_run_success,
			t.last_run_duration,
			t.next_run_at
		FROM apps a
		LEFT JOIN folder_path fp ON a.folder_id = fp.id
		LEFT JOIN task_runs t ON t.task_id = a.id AND a.type = 'task'
		%s

		UNION ALL

		SELECT
			f.id,
			COALESCE(fp.path, '/') as path,
			f.parent_folder_id as folder_id,
			f.name as name,
			'' as content,
			f.created_at,
			f.updated_at,
			f.created_by,
			f.updated_by,
			NULL as visibility,
			'_folder' as type,
			NULL as last_run_at,
			NULL as last_run_success,
			NULL as last_run_duration,
			NULL as next_run_at
		FROM folders f
		LEFT JOIN folder_path fp ON f.parent_folder_id = fp.id
		WHERE f.parent_folder_id %s
		ORDER BY type, %s %s`, whereClause,
		func() string {
			if path == "/" || path == "" {
				return "IS NULL"
			} else {
				return fmt.Sprintf(`= (
					WITH RECURSIVE folder_path(id, parent_folder_id, name, path) AS (
						SELECT id, parent_folder_id, name, '/' || name || '/' as path
						FROM folders
						WHERE parent_folder_id IS NULL

						UNION ALL

						SELECT f.id, f.parent_folder_id, f.name, fp.path || f.name || '/' as path
						FROM folders f
						JOIN folder_path fp ON f.parent_folder_id = fp.id
					)
					SELECT id FROM folder_path WHERE path = ?
				)`)
			}
		}(), orderBy, order)

	// Prepare arguments
	args := folderIDArgs
	if path != "/" && path != "" {
		args = append(args, path) // For the folder query
	}

	err := app.Sqlite.SelectContext(ctx, &dbApps, query, args...)
	if err != nil {
		err = fmt.Errorf("error listing apps: %w", err)
	}

	apps := make([]AppListItem, len(dbApps))
	for i, a := range dbApps {
		apps[i] = AppListItem{
			ID:         a.ID,
			Path:       a.Path,
			FolderID:   a.FolderID,
			Name:       a.Name,
			Content:    a.Content,
			CreatedAt:  a.CreatedAt,
			UpdatedAt:  a.UpdatedAt,
			CreatedBy:  a.CreatedBy,
			UpdatedBy:  a.UpdatedBy,
			Visibility: a.Visibility,
			Type:       a.Type,
		}
		apps[i].TaskInfo = &TaskInfo{
			LastRunAt:       a.LastRunAt,
			LastRunSuccess:  a.LastRunSuccess,
			LastRunDuration: a.LastRunDuration,
			NextRunAt:       a.NextRunAt,
		}
		if app.NoPublicSharing && apps[i].Visibility != nil && *apps[i].Visibility == "public" {
			apps[i].Visibility = nil
		}
		if app.NoPasswordProtectedSharing && apps[i].Visibility != nil && *apps[i].Visibility == "password-protected" {
			apps[i].Visibility = nil
		}
	}
	return AppListResponse{Apps: apps}, err
}
