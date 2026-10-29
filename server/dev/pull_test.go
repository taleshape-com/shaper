// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"os"
	"path/filepath"
	"shaper/server/api"
	"testing"
	"time"
)

func TestValidateInsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()

	tests := []struct {
		name      string
		target    string
		expectErr bool
	}{
		{name: "same directory", target: baseDir, expectErr: false},
		{name: "subdirectory", target: filepath.Join(baseDir, "sub", "dir"), expectErr: false},
		{name: "parent traversal", target: filepath.Join(baseDir, "..", "escape"), expectErr: true},
		{name: "double parent traversal", target: filepath.Join(baseDir, "..", "..", "escape"), expectErr: true},
		{name: "nested then escape", target: filepath.Join(baseDir, "sub", "..", "..", "escape"), expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInsideBaseDir(baseDir, tc.target)
			if tc.expectErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWriteAppFilePathTraversal(t *testing.T) {
	baseDir := t.TempDir()

	app := api.App{
		ID:        "test-id",
		Name:      "test",
		Type:      "dashboard",
		Path:      "/../../etc",
		Content:   "SELECT 1",
		UpdatedAt: time.Now(),
	}

	err := writeAppFile(baseDir, app)
	if err == nil {
		t.Fatal("expected path traversal to be rejected")
	}

	escapedDir := filepath.Join(baseDir, "..", "..", "etc")
	if _, statErr := os.Stat(escapedDir); statErr == nil {
		t.Fatal("directory was created outside base dir")
	}
}

func TestWriteAppFileSafePath(t *testing.T) {
	baseDir := t.TempDir()

	app := api.App{
		ID:        "test-id",
		Name:      "test",
		Type:      "dashboard",
		Path:      "/dashboards/analytics",
		Content:   "SELECT 1",
		UpdatedAt: time.Now(),
	}

	err := writeAppFile(baseDir, app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filePath := filepath.Join(baseDir, "dashboards", "analytics", "test"+DASHBOARD_SUFFIX)
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist at %s: %v", filePath, err)
	}
}
