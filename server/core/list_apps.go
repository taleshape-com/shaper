// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"time"
)

type AppRecord struct {
	ID         string    `db:"id" json:"id"`
	Path       string    `db:"path" json:"path"`
	Name       string    `db:"name" json:"name"`
	Content    string    `db:"content" json:"content"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
	CreatedBy  *string   `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy  *string   `db:"updated_by" json:"updatedBy,omitempty"`
	Visibility *string   `db:"visibility" json:"visibility,omitempty"`
	TaskInfo   any       `db:"task_info" json:"taskInfo,omitempty"`
	Type       string    `db:"type" json:"type"`
}

type AppListResponse struct {
	Apps []AppRecord `json:"apps"`
}

func ListApps(app *App, ctx context.Context, sort string, order string) (AppListResponse, error) {
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

	apps := []AppRecord{}
	optionalFilter := ""
	if app.NoTasks {
		optionalFilter = "WHERE type = 'dashboard'"
	}
	err := app.DB.SelectContext(ctx, &apps,
		fmt.Sprintf(`SELECT a.*,
			CASE WHEN a.type = 'task' THEN
				{
					'lastRunAt': epoch_ms(t.last_run_at),
					'lastRunSuccess': t.last_run_success,
					'lastRunDuration': round(epoch(t.last_run_duration) * 1000),
					'nextRunAt': epoch_ms(t.next_run_at),
				}
			END AS task_info
			FROM %s.apps a
			LEFT JOIN %s.task_runs t ON t.task_id = a.id AND a.type = 'task'
			%s
			ORDER BY %s %s`, app.Schema, app.Schema, optionalFilter, orderBy, order))
	if err != nil {
		err = fmt.Errorf("error listing apps: %w", err)
	}
	if app.NoPublicSharing {
		for i := range apps {
			apps[i].Visibility = nil
		}
	}
	return AppListResponse{Apps: apps}, err
}
