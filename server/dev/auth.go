package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SystemConfig struct {
	LoginRequired bool `json:"loginRequired"`
}

func fetchSystemConfig(ctx context.Context, baseURL string) (SystemConfig, error) {
	var cfg SystemConfig
	base := strings.TrimSuffix(baseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/system/config", nil)
	if err != nil {
		return cfg, fmt.Errorf("failed to build system config request: %w", err)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return cfg, fmt.Errorf("failed to fetch system config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return cfg, fmt.Errorf("system config request failed with status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to decode system config: %w", err)
	}
	return cfg, nil
}

type AuthManager struct {
	ctx           context.Context
	baseURL       string
	authFile      string
	loginRequired bool
	logger        *slog.Logger
	sessionToken  string
	mu            sync.Mutex
}

func NewAuthManager(ctx context.Context, baseURL, authFile string, loginRequired bool, logger *slog.Logger) *AuthManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthManager{
		ctx:           ctx,
		baseURL:       strings.TrimSuffix(baseURL, "/"),
		authFile:      authFile,
		loginRequired: loginRequired,
		logger:        logger,
	}
}

func (a *AuthManager) LoginRequired() bool {
	return a.loginRequired
}

func (a *AuthManager) EnsureSession() error {
	if !a.loginRequired {
		return nil
	}
	_, err := a.getOrPromptSessionTokenLocked(true)
	return err
}

func (a *AuthManager) SessionToken() (string, error) {
	if !a.loginRequired {
		return "", nil
	}
	return a.getOrPromptSessionTokenLocked(true)
}

func (a *AuthManager) RequireInteractiveLogin() error {
	if !a.loginRequired {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionToken = ""
	_, err := a.promptForLoginLocked()
	return err
}

func (a *AuthManager) getOrPromptSessionTokenLocked(allowPrompt bool) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.sessionToken == "" {
		if token, err := os.ReadFile(a.authFile); err == nil {
			if trimmed := strings.TrimSpace(string(token)); trimmed != "" {
				a.sessionToken = trimmed
			}
		}
	}

	if a.sessionToken != "" {
		return a.sessionToken, nil
	}

	if !allowPrompt {
		return "", errors.New("authentication required")
	}

	token, err := a.promptForLoginLocked()
	if err != nil {
		return "", err
	}
	a.sessionToken = token
	return a.sessionToken, nil
}

func (a *AuthManager) promptForLoginLocked() (string, error) {
	fmt.Fprintln(os.Stdout, "Authentication is required to use shaper dev.")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start temporary auth server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	tokenCh := make(chan string, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setCORSHeaders(w)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.Method != http.MethodPost || r.URL.Path != "/token" {
				http.NotFound(w, r)
				return
			}
			var payload struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}
			token := strings.TrimSpace(payload.Token)
			if token == "" {
				http.Error(w, "token required", http.StatusBadRequest)
				return
			}
			select {
			case tokenCh <- token:
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			default:
				http.Error(w, "already authenticated", http.StatusGone)
			}
		}),
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("Auth callback server error", slog.Any("error", err))
		}
	}()

	loginURL := fmt.Sprintf("%s/dev-login?port=%d", a.baseURL, port)
	fmt.Fprintf(os.Stdout, "Opening %s ...\n", loginURL)
	if err := OpenURL(loginURL); err != nil {
		fmt.Fprintf(os.Stdout, "Failed to open browser automatically: %v\nPlease open the URL manually.\n", err)
	}

	var token string
	select {
	case token = <-tokenCh:
	case <-a.ctx.Done():
		_ = server.Shutdown(context.Background())
		return "", errors.New("authentication cancelled")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	if err := a.saveTokenLocked(token); err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stdout, "Authentication successful. Token saved to %s\n", a.authFile)
	return token, nil
}

func (a *AuthManager) saveTokenLocked(token string) error {
	dir := filepath.Dir(a.authFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create auth directory: %w", err)
		}
	}
	if err := os.WriteFile(a.authFile, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("failed to save auth token: %w", err)
	}
	return nil
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
