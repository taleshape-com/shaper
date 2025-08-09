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
	err := app.DB.SelectContext(ctx, &apps,
		fmt.Sprintf(`SELECT *
		 FROM %s.apps
		 ORDER BY %s %s`, app.Schema, orderBy, order))
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
