package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nrednav/cuid2"
)

func CreateDashboard(app *App, ctx context.Context, name string, content string) (string, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("dashboard name cannot be empty")
	}

	// Generate unique ID
	id := cuid2.Generate()
	now := time.Now()

	// Insert into DB
	_, err := app.db.ExecContext(ctx,
		`INSERT INTO `+app.Schema+`.dashboards (
			id, path, name, content, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, "/", name, content, now, now,
	)
	if err != nil {
		return id, fmt.Errorf("failed to create dashboard: %w", err)
	}

	return id, nil
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
