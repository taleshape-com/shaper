// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunPreviewCommand(ctx context.Context, configPath, authFile string, noOpen bool, dashboardPath string) error {
	if !strings.HasSuffix(dashboardPath, DASHBOARD_SUFFIX) {
		return fmt.Errorf("file %s is not a dashboard (must end with %s)", dashboardPath, DASHBOARD_SUFFIX)
	}

	cfg, err := loadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	if authFile == "" {
		authFile = defaultAuthFile
	}
	authFilePath, err := resolveAbsolutePath(authFile)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	systemCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authManager := NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired)
	if err := authManager.EnsureSession(); err != nil {
		return err
	}

	client, err := NewAPIClient(ctx, cfg.URL, authManager)
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	content, err := os.ReadFile(dashboardPath)
	if err != nil {
		return fmt.Errorf("failed to read dashboard file: %w", err)
	}

	name := strings.TrimSuffix(filepath.Base(dashboardPath), DASHBOARD_SUFFIX)

	dashboardID, err := client.CreateDashboard(ctx, name, string(content), "/")
	if err != nil {
		return fmt.Errorf("failed to create preview dashboard: %w", err)
	}

	url := fmt.Sprintf("%s/dashboards/%s", strings.TrimSuffix(cfg.URL, "/"), dashboardID)
	fmt.Printf("Preview created: %s\n", url)

	if !noOpen {
		fmt.Printf("Opening %s in browser...\n", url)
		if err := OpenURL(url); err != nil {
			fmt.Printf("WARNING: Failed to open browser: %v\n", err)
		}
	}

	return nil
}
