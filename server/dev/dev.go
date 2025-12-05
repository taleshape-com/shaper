// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"fmt"
)

func RunDevCommand(ctx context.Context, configPath, authFile string) error {
	fmt.Printf("Starting Shaper Dev File Watcher...\n\n")

	cfg, err := loadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	watchDir, err := resolveAbsolutePath(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve watch directory: %w", err)
	}
	if err := ensureDirExists(watchDir); err != nil {
		return err
	}

	if authFile == "" {
		authFile = ".shaper-auth"
	}
	authFilePath, err := resolveAbsolutePath(authFile)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	fmt.Println("Connecting to Shaper at: " + cfg.URL)

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

	watcher, err := Watch(WatchConfig{
		WatchDirPath: watchDir,
		Client:       client,
		BaseURL:      cfg.URL,
	})
	if err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	fmt.Println("\nPress Ctrl+C to stop")

	<-ctx.Done()
	watcher.Stop()
	fmt.Println("Stopped dev watcher")
	return nil
}
