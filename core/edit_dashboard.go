package core

import (
	"context"
	"fmt"
	"time"
)

func GetDashboardQuery(app *App, ctx context.Context, id string) (Dashboard, error) {
	var dashboard Dashboard
	err := app.db.GetContext(ctx, &dashboard,
		`SELECT * FROM `+app.Schema+`.dashboards WHERE id = $1`, id)
	if err != nil {
		return dashboard, fmt.Errorf("failed to get dashboard: %w", err)
	}
	return dashboard, nil
}

func SaveDashboardName(app *App, ctx context.Context, id string, name string) error {
	result, err := app.db.ExecContext(ctx,
		`UPDATE `+app.Schema+`.dashboards
			 SET name = $1, updated_at = $2
			 WHERE id = $3`,
		name, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to save dashboard name: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("dashboard not found: %s", id)
	}

	return nil
}

func SaveDashboardQuery(app *App, ctx context.Context, id string, content string) error {
	result, err := app.db.ExecContext(ctx,
		`UPDATE `+app.Schema+`.dashboards
		 SET content = $1, updated_at = $2
		 WHERE id = $3`,
		content, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to save dashboard: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("dashboard not found: %s", id)
	}

	return nil
}
