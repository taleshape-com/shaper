package dev

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

func RunCommand(args []string) error {
	// Root command flags
	rootFlags := ff.NewFlagSet("shaper")
	rootCmd := &ff.Command{
		Name:  "shaper",
		Usage: "shaper <subcommand> [FLAGS]",
		Flags: rootFlags,
	}

	// dev subcommand
	devFlags := ff.NewFlagSet("dev").SetParent(rootFlags)
	devConfigPath := devFlags.StringLong("config", defaultConfigPath, "Path to config file")
	devAuthFile := devFlags.StringLong("auth-file", defaultAuthFile, "Path to auth token file")

	devCmd := &ff.Command{
		Name:      "dev",
		Usage:     "shaper dev [--config path] [--auth-file path]",
		ShortHelp: "watch local dashboard files and show preview",
		Flags:     devFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runDevCommand(ctx, *devConfigPath, *devAuthFile)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, devCmd)

	// pull subcommand
	pullFlags := ff.NewFlagSet("pull").SetParent(rootFlags)
	pullConfigPath := pullFlags.StringLong("config", defaultConfigPath, "Path to config file")
	pullAuthFile := pullFlags.StringLong("auth-file", defaultAuthFile, "Path to auth token file")

	pullCmd := &ff.Command{
		Name:      "pull",
		Usage:     "shaper pull [--config path] [--auth-file path]",
		ShortHelp: "pull dashboards from server to local files",
		Flags:     pullFlags,
		Exec: func(ctx context.Context, args []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
			return RunPullCommand(ctx, *pullConfigPath, *pullAuthFile, logger)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, pullCmd)

	// deploy subcommand
	deployFlags := ff.NewFlagSet("deploy").SetParent(rootFlags)
	deployConfigPath := deployFlags.StringLong("config", defaultConfigPath, "Path to config file")

	deployCmd := &ff.Command{
		Name:      "deploy",
		Usage:     "shaper deploy [--config path]",
		ShortHelp: "deploy dashboards from files using API key auth",
		Flags:     deployFlags,
		Exec: func(ctx context.Context, args []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
			return RunDeployCommand(ctx, *deployConfigPath, logger)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, deployCmd)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ParseAndRun(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", ffhelp.Command(rootCmd))
		return err
	}
	return nil
}

func runDevCommand(ctx context.Context, configPath, authFile string) error {
	cfg, err := LoadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	watchDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve watch directory: %w", err)
	}
	if err := EnsureDirExists(watchDir); err != nil {
		return err
	}

	if authFile == "" {
		authFile = defaultAuthFile
	}
	authFilePath, err := filepath.Abs(authFile)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	systemCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authManager := NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired, logger)
	if err := authManager.EnsureSession(); err != nil {
		return err
	}

	client, err := NewAPIClient(ctx, cfg.URL, logger, authManager)
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	watcher, err := Watch(WatchConfig{
		WatchDirPath: watchDir,
		Client:       client,
		Logger:       logger,
		BaseURL:      cfg.URL,
	})
	if err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	logger.Info("Watching dashboards; press Ctrl+C to stop", slog.String("dir", watchDir), slog.String("url", cfg.URL))

	<-ctx.Done()
	watcher.Stop()
	logger.Info("Stopped dev watcher")
	return nil
}
