package core

import (
	"context"
	"fmt"
)

func DeleteDashboard(app *App, ctx context.Context, id string) error {
	result, err := app.db.ExecContext(ctx,
		`DELETE FROM `+app.Schema+`.dashboards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
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
