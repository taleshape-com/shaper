package core

import (
	"context"
	"fmt"
)

func ListDashboards(app *App, ctx context.Context) (ListResult, error) {
	dashboards := []Dashboard{}
	err := app.db.SelectContext(ctx, &dashboards,
		`SELECT *
		 FROM `+app.Schema+`.dashboards
		 ORDER BY name`)
	if err != nil {
		return ListResult{Dashboards: dashboards}, fmt.Errorf("error listing dashboards: %w", err)
	}
	return ListResult{Dashboards: dashboards}, nil
}
