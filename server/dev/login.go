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

	systemCfg, sysErr := fetchSystemConfig(ctx, cfg.URL, authFilePath)
	if sysErr != nil && !isAuthOrProxyBlockError(sysErr) {
		return sysErr
	}

	loginRequired := true
	if sysErr == nil {
		loginRequired = systemCfg.LoginRequired
	}

	authManager := NewAuthManager(ctx, cfg.URL, authFilePath, loginRequired)
	authManager.SetNoOpen(noOpen)

	if !loginRequired {
		fmt.Println("Already logged in.")
		return nil
	}

	return authManager.Login()
}
