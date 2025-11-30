package dev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasLeadingShaperIDComment(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "missing comment",
			content:  "select 1",
			expected: false,
		},
		{
			name:     "valid comment with newline",
			content:  "-- shaperid:ckb0example12345678901234\nselect 1",
			expected: true,
		},
		{
			name:     "valid comment without newline",
			content:  "-- shaperid:ckb0example12345678901234",
			expected: true,
		},
		{
			name:     "comment with trailing spaces is invalid",
			content:  "-- shaperid:ckb0example12345678901234   \nselect 1",
			expected: false,
		},
		{
			name:     "comment with extra text is invalid",
			content:  "-- shaperid:ckb0example12345678901234 extra\nselect 1",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasLeadingShaperIDComment(tc.content); got != tc.expected {
				t.Fatalf("expected %v, got %v for content %q", tc.expected, got, tc.content)
			}
		})
	}
}

func TestPrependShaperIDComment(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "content without leading newline gets newline inserted",
			content:  "select 1",
			expected: "-- shaperid:testid\n\nselect 1",
		},
		{
			name:     "content with leading newline keeps newline",
			content:  "\nselect 1",
			expected: "-- shaperid:testid\n\nselect 1",
		},
		{
			name:     "empty content still ends with newline",
			content:  "",
			expected: "-- shaperid:testid\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := prependShaperIDComment("testid", tc.content)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestEnsureShaperIDForFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.dashboard.sql")
	if err := os.WriteFile(filePath, []byte("select 1"), 0o644); err != nil {
		t.Fatalf("failed writing temp file: %v", err)
	}

	contentBytes, updated, newID, err := ensureShaperIDForFile(filePath)
	if err != nil {
		t.Fatalf("ensureShaperIDForFile returned error: %v", err)
	}
	if !updated {
		t.Fatalf("expected file to be updated with new shaper ID")
	}
	if newID == "" {
		t.Fatalf("expected new shaper ID to be returned")
	}
	if !strings.HasPrefix(string(contentBytes), "-- shaperid:"+newID) {
		t.Fatalf("expected file content to start with new shaper ID comment, got %q", string(contentBytes))
	}

	onDisk, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed rereading updated file: %v", err)
	}
	if string(onDisk) != string(contentBytes) {
		t.Fatalf("expected file contents to match return value")
	}

	_, updated, _, err = ensureShaperIDForFile(filePath)
	if err != nil {
		t.Fatalf("ensureShaperIDForFile returned error on already tagged file: %v", err)
	}
	if updated {
		t.Fatalf("did not expect second invocation to rewrite file")
	}
}

func TestEnsureShaperIDsForDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed creating nested dir: %v", err)
	}

	withIDPath := filepath.Join(tmpDir, "with.dashboard.sql")
	if err := os.WriteFile(withIDPath, []byte("-- shaperid:test123\nselect 1"), 0o644); err != nil {
		t.Fatalf("failed writing withID file: %v", err)
	}

	withoutIDPath := filepath.Join(nestedDir, "without.dashboard.sql")
	if err := os.WriteFile(withoutIDPath, []byte("select 2"), 0o644); err != nil {
		t.Fatalf("failed writing withoutID file: %v", err)
	}

	if err := ensureShaperIDsForDir(tmpDir); err != nil {
		t.Fatalf("ensureShaperIDsForDir returned error: %v", err)
	}

	content, err := os.ReadFile(withoutIDPath)
	if err != nil {
		t.Fatalf("failed reading ensured file: %v", err)
	}
	if !strings.HasPrefix(string(content), "-- shaperid:") {
		t.Fatalf("expected ensured file to start with shaper ID comment, got %q", string(content))
	}
}
