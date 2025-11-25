package dev

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const defaultServerAddr = "localhost:5454"
const defaultServerURL = "http://localhost:5454"

func RunCommand(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dir := fs.String("dir", "", "Directory containing *.dashboard.sql files to watch")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: shaper dev --dir <path>\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dir == "" && fs.NArg() > 0 {
		*dir = fs.Arg(0)
	}
	if *dir == "" {
		return fmt.Errorf("--dir is required")
	}

	watchDir, err := filepath.Abs(*dir)
	if err != nil {
		return fmt.Errorf("failed to resolve watch directory: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := NewAPIClient(ctx, defaultServerURL, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	watcher, err := Watch(WatchConfig{
		WatchDirPath: watchDir,
		Client:       client,
		Logger:       logger,
		Addr:         defaultServerAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	logger.Info("Watching dashboards; press Ctrl+C to stop", slog.String("dir", watchDir))

	<-ctx.Done()
	watcher.Stop()
	logger.Info("Stopped dev watcher")
	return nil
}
