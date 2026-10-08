// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveHealthcheckURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		addr      string
		tlsDomain string
		expected  string
		wantErr   bool
	}{
		{
			name:     "empty inputs defaults to localhost:5454/health",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "addr with leading colon :5454",
			addr:     ":5454",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "addr with 0.0.0.0:5454",
			addr:     "0.0.0.0:5454",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "addr with [::]:5454",
			addr:     "[::]:5454",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "addr with 127.0.0.1:8080",
			addr:     "127.0.0.1:8080",
			expected: "http://127.0.0.1:8080/health",
		},
		{
			name:     "addr with custom host:port",
			addr:     "my-host:9000",
			expected: "http://my-host:9000/health",
		},
		{
			name:      "tlsDomain used when provided",
			tlsDomain: "example.com",
			expected:  "https://example.com/health",
		},
		{
			name:     "rawURL without path appends /health",
			rawURL:   "http://localhost:5454",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "rawURL with slash appends health",
			rawURL:   "http://localhost:5454/",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "rawURL with /health preserved",
			rawURL:   "http://localhost:5454/health",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "rawURL without scheme adds http",
			rawURL:   "localhost:5454",
			expected: "http://localhost:5454/health",
		},
		{
			name:     "rawURL with custom path preserved",
			rawURL:   "http://localhost:5454/custom-health",
			expected: "http://localhost:5454/custom-health",
		},
		{
			name:      "rawURL overrides addr and tlsDomain",
			rawURL:    "http://other-host:8080/health",
			addr:      ":5454",
			tlsDomain: "example.com",
			expected:  "http://other-host:8080/health",
		},
		{
			name:    "invalid URL returns error",
			rawURL:  "://invalid-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveHealthcheckURL(tt.rawURL, tt.addr, tt.tlsDomain)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveHealthcheckURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ResolveHealthcheckURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRunHealthcheckCommand(t *testing.T) {
	t.Run("server returns 200 OK", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("expected path /health, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		err := RunHealthcheckCommand(context.Background(), ts.URL, "", "", 2*time.Second, true)
		if err != nil {
			t.Fatalf("expected healthy, got error: %v", err)
		}
	})

	t.Run("server returns 500 error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		err := RunHealthcheckCommand(context.Background(), ts.URL, "", "", 2*time.Second, true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("server unreachable", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		tsURL := ts.URL
		ts.Close() // Immediately close to simulate unreachable server

		err := RunHealthcheckCommand(context.Background(), tsURL, "", "", 2*time.Second, true)
		if err == nil {
			t.Fatal("expected error for closed server, got nil")
		}
	})

	t.Run("request times out", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		err := RunHealthcheckCommand(context.Background(), ts.URL, "", "", 50*time.Millisecond, true)
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
	})
}
