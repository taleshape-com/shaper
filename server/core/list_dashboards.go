package core

import (
	"context"
	"fmt"
)

func ListDashboards(app *App, ctx context.Context, sort string, order string) (ListResult, error) {
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

	dashboards := []Dashboard{}
	err := app.DB.SelectContext(ctx, &dashboards,
		fmt.Sprintf(`SELECT *
		 FROM %s.apps
		 ORDER BY %s %s`, app.Schema, orderBy, order))
	if err != nil {
		err = fmt.Errorf("error listing dashboards: %w", err)
	}
	if app.NoPublicSharing {
		for i := range dashboards {
			dashboards[i].Visibility = nil
		}
	}
	return ListResult{Dashboards: dashboards}, err
}
