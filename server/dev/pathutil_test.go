package dev

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUserPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{name: "empty path", input: "", want: ""},
		{name: "relative path", input: "./dashboards", want: "./dashboards"},
		{name: "home only", input: "~", want: homeDir},
		{name: "home slash", input: "~/dashboards/main", want: filepath.Join(homeDir, "dashboards", "main")},
		{name: "home backslash", input: "~\\nested\\file", want: filepath.Join(homeDir, "nested", "file")},
		{name: "unsupported user", input: "~someone/else", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandUserPath(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestResolveAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	relative := "./testdata"

	path := filepath.Join(tmpDir, relative)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	resolved, err := resolveAbsolutePath(relative)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(tmpDir, "testdata")
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}
