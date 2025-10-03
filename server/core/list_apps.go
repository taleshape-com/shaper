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
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy *string   `json:"createdBy,omitempty"`
	UpdatedBy *string   `json:"updatedBy,omitempty"`
}

type FolderDbRecord struct {
	ID        string    `db:"id"`
	Path      string    `db:"path"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	CreatedBy *string   `db:"created_by"`
	UpdatedBy *string   `db:"updated_by"`
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

	// Build path filter
	pathFilter := ""
	if path != "" {
		pathFilter = fmt.Sprintf("WHERE path = '%s'", path)
	} else {
		// When at root, show only items at root level (path = '/' or path = '')
		pathFilter = "WHERE (path = '/' OR path = '')"
	}

	// Build type filter
	typeFilter := ""
	if app.NoTasks {
		if pathFilter != "" {
			typeFilter = " AND type = 'dashboard'"
		} else {
			typeFilter = "WHERE type = 'dashboard'"
		}
	}

	// Combine filters
	whereClause := ""
	if pathFilter != "" || typeFilter != "" {
		whereClause = pathFilter + typeFilter
	}

	// Starting type for folders with underscore to sort them always to the top
	err := app.Sqlite.SelectContext(ctx, &dbApps,
		fmt.Sprintf(`SELECT
			a.id,
			a.path,
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
			LEFT JOIN task_runs t ON t.task_id = a.id AND a.type = 'task'
			%s
		UNION ALL
		SELECT
			f.id,
			f.path,
			f.name,
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
			%s
			ORDER BY type, %s %s`, whereClause, whereClause, orderBy, order))
	if err != nil {
		err = fmt.Errorf("error listing apps: %w", err)
	}
	apps := make([]AppListItem, len(dbApps))
	for i, a := range dbApps {
		apps[i] = AppListItem{
			ID:         a.ID,
			Path:       a.Path,
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
