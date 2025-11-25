package dev

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	defaultServerURL   = "http://localhost:5454"
	defaultWatchFolder = "."
	defaultConfigPath  = "./shaper.json"
	defaultAuthFile    = ".shaper-auth"
)

type DevConfig struct {
	URL       string `json:"url"`
	Directory string `json:"directory"`
}

func RunCommand(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	configPath := fs.String("config", defaultConfigPath, "Path to dev config file")
	authFileFlag := fs.String("auth-file", defaultAuthFile, "Path to dev CLI auth token file")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: shaper dev [--config path] [--auth-file path]\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadOrPromptConfig(*configPath)
	if err != nil {
		return err
	}

	watchDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve watch directory: %w", err)
	}
	if err := ensureDirExists(watchDir); err != nil {
		return err
	}

	authFile := *authFileFlag
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

func loadOrPromptConfig(path string) (DevConfig, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return parseConfig(data)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return DevConfig{}, fmt.Errorf("failed to read config: %w", err)
	}
	return promptAndSaveConfig(path)
}

func parseConfig(data []byte) (DevConfig, error) {
	var cfg DevConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DevConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return normalizeConfig(cfg)
}

func normalizeConfig(cfg DevConfig) (DevConfig, error) {
	cfg.URL = strings.TrimSpace(cfg.URL)
	if cfg.URL == "" {
		cfg.URL = defaultServerURL
	}
	if _, err := url.ParseRequestURI(cfg.URL); err != nil {
		return DevConfig{}, fmt.Errorf("invalid url %q: %w", cfg.URL, err)
	}

	cfg.Directory = strings.TrimSpace(cfg.Directory)
	if cfg.Directory == "" {
		cfg.Directory = defaultWatchFolder
	}

	return cfg, nil
}

func promptAndSaveConfig(path string) (DevConfig, error) {
	fmt.Printf("Dev config file %s not found. Let's create it.\n", path)
	reader := bufio.NewReader(os.Stdin)

	urlVal := prompt(reader, fmt.Sprintf("Server URL [%s]: ", defaultServerURL))
	if urlVal == "" {
		urlVal = defaultServerURL
	}
	dirVal := prompt(reader, fmt.Sprintf("Directory to watch [%s]: ", defaultWatchFolder))
	if dirVal == "" {
		dirVal = defaultWatchFolder
	}

	cfg, err := normalizeConfig(DevConfig{
		URL:       urlVal,
		Directory: dirVal,
	})
	if err != nil {
		return DevConfig{}, err
	}
	if err := saveConfig(path, cfg); err != nil {
		return DevConfig{}, err
	}
	fmt.Printf("Saved dev config to %s\n", path)
	return cfg, nil
}

func prompt(reader *bufio.Reader, msg string) string {
	fmt.Print(msg)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func saveConfig(path string, cfg DevConfig) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func ensureDirExists(path string) error {
	if stat, err := os.Stat(path); err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		return nil
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create watch directory: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("failed to access watch directory: %w", err)
	}
}
