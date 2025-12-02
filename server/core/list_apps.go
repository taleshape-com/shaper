// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"shaper/server/api"
	"strings"
	"time"
)

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


type ListAppsOptions struct {
	Sort              string
	Order             string
	Path              string
	IncludeSubfolders bool
	IncludeContent    bool
	Limit             int
	Offset            int
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

func ListApps(app *App, ctx context.Context, opts ListAppsOptions) (api.AppsResponse, error) {
	sort := opts.Sort
	order := opts.Order
	path := opts.Path
	var orderColumn string
	switch sort {
	case "created":
		orderColumn = "created_at"
	case "name":
		orderColumn = "name"
	default:
		orderColumn = "updated_at"
	}

	if order != "asc" && order != "desc" {
		order = "desc"
	}

	dbApps := []AppDbRecord{}

	// Build folder filters
	var folderIDFilter string
	var folderIDArgs []interface{}

	if opts.IncludeSubfolders {
		if path == "/" || path == "" {
			folderIDFilter = "WHERE 1=1"
		} else {
			folderIDFilter = "WHERE (COALESCE(fp.path, '/') = ? OR COALESCE(fp.path, '/') LIKE ?)"
			folderIDArgs = append(folderIDArgs, path, path+"%")
		}
	} else {
		if path == "/" || path == "" {
			folderIDFilter = "WHERE a.folder_id IS NULL"
		} else {
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
	}

	if folderIDFilter == "" {
		folderIDFilter = "WHERE 1=1"
	}

	// Build type filter
	if app.NoTasks {
		if strings.Contains(folderIDFilter, "WHERE") {
			folderIDFilter += " AND a.type = 'dashboard'"
		} else {
			folderIDFilter = "WHERE a.type = 'dashboard'"
		}
	}

	var paginationClause string
	if opts.Limit > 0 {
		if opts.Offset > 0 {
			paginationClause = " LIMIT ? OFFSET ?"
		} else {
			paginationClause = " LIMIT ?"
		}
	} else if opts.Offset > 0 {
		paginationClause = " LIMIT -1 OFFSET ?"
	}

	// Build the query with recursive CTE for folder paths
	queryWithFolders := fmt.Sprintf(`
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
		ORDER BY type, %s %s%s`, folderIDFilter,
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
		}(), orderColumn, order, paginationClause)

	queryAppsOnly := fmt.Sprintf(`
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
		WHERE %s
		ORDER BY type, %s %s%s`, folderIDFilter, 
		func() string {
			// Build folder filter for recursive mode
			if path == "/" || path == "" {
				return "1=1" // Include all folders
			} else {
				// Include folders at the path or in subfolders
				return "(COALESCE(fp.path, '/') = ? OR COALESCE(fp.path, '/') LIKE ?)"
			}
		}(), orderColumn, order, paginationClause)

	// Prepare arguments
	args := folderIDArgs
	if opts.IncludeSubfolders && path != "/" && path != "" {
		// For recursive mode, we need to duplicate path args for the folder query part
		args = append(args, path, path+"%")
	} else if !opts.IncludeSubfolders && path != "/" && path != "" {
		args = append(args, path) // For the folder query in non-recursive mode
	}

	if opts.Limit > 0 {
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			args = append(args, opts.Offset)
		}
	} else if opts.Offset > 0 {
		args = append(args, opts.Offset)
	}

	var query string
	if opts.IncludeSubfolders {
		query = queryAppsOnly
	} else {
		query = queryWithFolders
	}

	err := app.Sqlite.SelectContext(ctx, &dbApps, query, args...)
	if err != nil {
		err = fmt.Errorf("error listing apps: %w", err)
	}

	apps := make([]api.App, len(dbApps))
	for i, a := range dbApps {
		apps[i] = api.App{
			ID:         a.ID,
			Path:       a.Path,
			FolderID:   a.FolderID,
			Name:       a.Name,
			CreatedAt:  a.CreatedAt,
			UpdatedAt:  a.UpdatedAt,
			CreatedBy:  a.CreatedBy,
			UpdatedBy:  a.UpdatedBy,
			Visibility: a.Visibility,
			Type:       a.Type,
		}
		if opts.IncludeContent {
			apps[i].Content = a.Content
		}
		apps[i].TaskInfo = &api.TaskInfo{
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

	// Calculate pagination info from limit/offset
	pageSize := opts.Limit
	if pageSize == 0 {
		pageSize = len(apps)
		if pageSize == 0 {
			pageSize = 1 // Avoid division by zero
		}
	}
	page := 1
	if pageSize > 0 && opts.Offset > 0 {
		page = (opts.Offset / pageSize) + 1
	}

	return api.AppsResponse{
		Apps:     apps,
		Page:     page,
		PageSize: pageSize,
	}, err
}
