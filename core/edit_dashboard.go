package core

import (
	"context"
	"fmt"
	"os"
	"path"
)

func GetDashboardQuery(app *App, ctx context.Context, dashboardName string) (string, error) {
	fileName := path.Join(app.DashboardDir, dashboardName+".sql")
	content, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to read dashboard file: %w", err)
	}
	return string(content), nil
}

func SaveDashboardQuery(app *App, ctx context.Context, dashboardName string, content string) error {
	fileName := path.Join(app.DashboardDir, dashboardName+".sql")
	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write dashboard file: %w", err)
	}
	return nil
}
