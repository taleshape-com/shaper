// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// expandUserPath resolves leading ~ references to the current user's home directory.
func expandUserPath(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	if p[0] != '~' {
		return p, nil
	}
	if len(p) > 1 && p[1] != '/' && p[1] != '\\' {
		return "", fmt.Errorf("cannot expand home directory in path %q", p)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	if len(p) == 1 {
		return homeDir, nil
	}

	suffix := strings.TrimLeft(p[1:], "/\\")
	if suffix == "" {
		return homeDir, nil
	}

	normalized := strings.ReplaceAll(suffix, "\\", "/")
	normalized = filepath.FromSlash(normalized)

	return filepath.Join(homeDir, normalized), nil
}

// resolveAbsolutePath expands ~ and returns an absolute path.
func resolveAbsolutePath(p string) (string, error) {
	expanded, err := expandUserPath(p)
	if err != nil {
		return "", err
	}
	if expanded == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

// resolvePathRelativeToConfig resolves a path relative to the given config file path.
func resolvePathRelativeToConfig(p string, configPath string) (string, error) {
	expanded, err := expandUserPath(p)
	if err != nil {
		return "", err
	}
	if expanded == "" {
		return "", nil
	}

	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}

	absConfigPath, err := resolveAbsolutePath(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve config path: %w", err)
	}

	configDir := filepath.Dir(absConfigPath)
	return filepath.Join(configDir, expanded), nil
}
