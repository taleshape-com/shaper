// SPDX-License-Identifier: MPL-2.0

package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type customCSSState struct {
	content      []byte
	etag         string
	modTime      time.Time
	lastModified string
}

// CustomCSSStore stores the current custom CSS and allows atomic updates (e.g. on SIGHUP).
type CustomCSSStore struct {
	filePath  string
	inlineCSS string
	logger    *slog.Logger
	state     atomic.Pointer[customCSSState]
}

// NewCustomCSSStore creates a store with initial CSS and modification time.
func NewCustomCSSStore(filePath string, inlineCSS string, initialContent string, modTime time.Time, logger *slog.Logger) *CustomCSSStore {
	store := &CustomCSSStore{
		filePath:  filePath,
		inlineCSS: inlineCSS,
		logger:    logger,
	}
	store.Set(initialContent, modTime)
	return store
}

// Set atomically updates the custom CSS content and its modification time.
func (s *CustomCSSStore) Set(css string, modTime time.Time) {
	cssBytes := []byte(css)
	etag := generateContentETag(cssBytes)
	s.state.Store(&customCSSState{
		content:      cssBytes,
		etag:         etag,
		modTime:      modTime,
		lastModified: modTime.UTC().Format(http.TimeFormat),
	})
}

// Get returns the current custom CSS state.
func (s *CustomCSSStore) Get() (content []byte, etag string, modTime time.Time, lastModified string) {
	st := s.state.Load()
	if st == nil {
		return nil, "", time.Time{}, ""
	}
	return st.content, st.etag, st.modTime, st.lastModified
}

// Content returns the current custom CSS string.
func (s *CustomCSSStore) Content() string {
	st := s.state.Load()
	if st == nil {
		return ""
	}
	return string(st.content)
}

// FilePath returns the configured custom CSS file path.
func (s *CustomCSSStore) FilePath() string {
	return s.filePath
}

// Reload re-reads the custom CSS from disk if a file path is configured.
// It combines the inline CSS (if any) with the file contents.
func (s *CustomCSSStore) Reload() error {
	if s.filePath == "" {
		if s.logger != nil {
			s.logger.Info("SIGHUP received, but no custom CSS file configured (--css-file)")
		}
		return nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("Failed to read custom CSS file on reload", slog.String("path", s.filePath), slog.Any("error", err))
		}
		return fmt.Errorf("reading custom CSS file %s: %w", s.filePath, err)
	}

	stat, err := os.Stat(s.filePath)
	modTime := time.Now()
	if err == nil {
		modTime = stat.ModTime()
	}

	content := string(data)
	if s.inlineCSS != "" {
		content = s.inlineCSS + "\n" + content
	}

	s.Set(content, modTime)

	if s.logger != nil {
		s.logger.Info("Reloaded custom CSS file", slog.String("path", s.filePath))
	}
	return nil
}
