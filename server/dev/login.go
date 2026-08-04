// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"fmt"
)

func RunLoginCommand(ctx context.Context, configPath, urlOverride, authFile string, noOpen bool) error {
	cfg, err := loadOrPromptConfig(configPath, urlOverride)
	if err != nil {
		return err
	}

	if authFile == "" {
		authFile = defaultAuthFile
	}
	authFilePath, err := resolvePathRelativeToConfig(authFile, configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	fmt.Println("Connecting to Shaper at: " + cfg.URL)

	systemCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authManager := NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired)
	authManager.SetNoOpen(noOpen)

	// Check if already logged in with a valid session
	if _, err := NewAPIClient(ctx, cfg.URL, authManager); err == nil {
		fmt.Println("Already logged in.")
		return nil
	}

	return authManager.Login()
}
