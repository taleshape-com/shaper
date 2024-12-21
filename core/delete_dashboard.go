package core

import (
	"context"
	"fmt"
	"os"
	"path"
)

func DeleteDashboard(app *App, ctx context.Context, dashboardName string) error {
    fileName := path.Join(app.DashboardDir, dashboardName+".sql")

    // Check if dashboard exists
    if _, err := os.Stat(fileName); os.IsNotExist(err) {
        return fmt.Errorf("dashboard does not exist: %s", dashboardName)
    }

    // Delete the dashboard file
    if err := os.Remove(fileName); err != nil {
        return fmt.Errorf("failed to delete dashboard: %w", err)
    }

    return nil
}
