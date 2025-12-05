// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	defaultServerURL   = "http://localhost:5454"
	defaultWatchFolder = "."
	defaultConfigPath  = "./shaper.json"
	defaultAuthFile    = ".shaper-auth"
)

type Config struct {
	URL       string     `json:"url"`
	Directory string     `json:"directory"`
	LastPull  *time.Time `json:"lastPull,omitempty"`
}

var ErrConfigNotFound = errors.New("config file not found")
var ErrInterrupted = errors.New("interrupted by user")

func loadOrPromptConfig(path string) (Config, error) {
	cfg, err := LoadConfig(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, ErrConfigNotFound) {
		return Config{}, err
	}
	return promptAndSaveConfig(path)
}

func LoadConfig(path string) (Config, error) {
	resolvedPath, err := expandUserPath(path)
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}
	return parseConfig(data)
}

func parseConfig(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return normalizeConfig(cfg)
}

func normalizeConfig(cfg Config) (Config, error) {
	cfg.URL = strings.TrimSpace(cfg.URL)
	if cfg.URL == "" {
		cfg.URL = defaultServerURL
	}
	if _, err := url.ParseRequestURI(cfg.URL); err != nil {
		return Config{}, fmt.Errorf("invalid url %q: %w", cfg.URL, err)
	}

	cfg.Directory = strings.TrimSpace(cfg.Directory)
	if cfg.Directory == "" {
		cfg.Directory = defaultWatchFolder
	}

	return cfg, nil
}

func promptAndSaveConfig(path string) (Config, error) {
	fmt.Printf("Config file %s not found. Let's create it.\n\n", path)
	reader := bufio.NewReader(os.Stdin)

	urlVal, err := prompt(reader, fmt.Sprintf("Server URL [Press ENTER to use %s]: ", defaultServerURL))
	if err != nil {
		return Config{}, err
	}
	if urlVal == "" {
		urlVal = defaultServerURL
	}
	dirVal, err := prompt(reader, fmt.Sprintf("Directory to watch [Press ENTER to use %s]: ", defaultWatchFolder))
	if err != nil {
		return Config{}, err
	}
	if dirVal == "" {
		dirVal = defaultWatchFolder
	}

	cfg, err := normalizeConfig(Config{
		URL:       urlVal,
		Directory: dirVal,
	})
	if err != nil {
		return Config{}, err
	}
	if err := SaveConfig(path, cfg); err != nil {
		return Config{}, err
	}
	fmt.Printf("Saved config to %s\n", path)
	return cfg, nil
}

func prompt(reader *bufio.Reader, msg string) (string, error) {
	fmt.Print(msg)

	// Set up signal handling for CTRL-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT)
	defer signal.Stop(sigChan)

	// Channel to receive input
	inputChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Read input in a goroutine
	go func() {
		input, err := reader.ReadString('\n')
		if err != nil {
			errChan <- err
			return
		}
		inputChan <- input
	}()

	// Wait for either input or signal
	select {
	case input := <-inputChan:
		return strings.TrimSpace(input), nil
	case <-sigChan:
		fmt.Print("\n\nInterrupted\n\n")
		return "", ErrInterrupted
	case err := <-errChan:
		return "", err
	}
}

func SaveConfig(path string, cfg Config) error {
	resolvedPath, err := expandUserPath(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(resolvedPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	if err := os.WriteFile(resolvedPath, append(data, '\n'), 0o644); err != nil {
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
			return fmt.Errorf("failed to create directory: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("failed to access directory: %w", err)
	}
}
