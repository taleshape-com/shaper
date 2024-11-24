package core

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func ListDashboards(app *App, ctx context.Context) (ListResult, error) {
	result := ListResult{Dashboards: []string{}}
	files, err := os.ReadDir(app.DashboardDir)
	if err != nil {
		return result, fmt.Errorf("failed to read dashboards directory: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			dashboardName := strings.TrimSuffix(file.Name(), ".sql")
			result.Dashboards = append(result.Dashboards, dashboardName)
		}
	}
	return result, nil
}
