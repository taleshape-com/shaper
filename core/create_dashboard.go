package core

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
)

func CreateDashboard(app *App, ctx context.Context, dashboardName string, content string) error {
	// Sanitize dashboard name
	dashboardName = strings.TrimSpace(dashboardName)
	if dashboardName == "" {
		return fmt.Errorf("dashboard name cannot be empty")
	}

	// Basic validation of dashboard name
	// Avoid directory traversal and ensure valid filename
	if strings.Contains(dashboardName, "/") || strings.Contains(dashboardName, "\\") {
		return fmt.Errorf("invalid dashboard name: must not contain path separators")
	}

	if !isValidDashboardName(dashboardName) {
		return fmt.Errorf("invalid dashboard name: use only letters, numbers, dashes, and underscores")
	}

	fileName := path.Join(app.DashboardDir, dashboardName+".sql")

	// Check if dashboard already exists
	if _, err := os.Stat(fileName); err == nil {
		return fmt.Errorf("dashboard already exists: %s", dashboardName)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking dashboard existence: %w", err)
	}

	// Create the dashboard file
	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to create dashboard file: %w", err)
	}

	return nil
}

func isValidDashboardName(name string) bool {
	// Only allow letters, numbers, dashes, and underscores
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_') {
			return false
		}
	}
	return true
}
