// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const headersEnv = "SHAPER_HEADERS"

type SystemConfig struct {
	LoginRequired bool `json:"loginRequired"`
}

var errSystemConfigBlocked = errors.New("system config blocked by auth or proxy")

func isAuthOrProxyBlockError(err error) bool {
	return errors.Is(err, errSystemConfigBlocked)
}

func parseHeadersEnv(raw string) (map[string]string, error) {
	headers := make(map[string]string)
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return headers, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse JSON headers: %w", err)
		}
		for k, v := range parsed {
			key := strings.TrimSpace(k)
			if key != "" {
				headers[key] = v
			}
		}
		return headers, nil
	}

	var parts []string
	if strings.Contains(trimmed, "\n") {
		parts = strings.Split(trimmed, "\n")
	} else {
		parts = strings.Split(trimmed, ",")
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("expected 'Header: Value' format, got %q", part)
		}
		key := strings.TrimSpace(k)
		if key == "" {
			return nil, fmt.Errorf("empty header name in %q", part)
		}
		headers[key] = strings.TrimSpace(v)
	}
	return headers, nil
}

func applyCustomHeaders(req *http.Request) error {
	raw := strings.TrimSpace(os.Getenv(headersEnv))
	if raw == "" {
		return nil
	}
	headers, err := parseHeadersEnv(raw)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", headersEnv, err)
	}
	for k, v := range headers {
		if strings.EqualFold(k, "Host") {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}
	return nil
}

func resolveAuthToken(authFile string) string {
	if authFile != "" {
		if tokenBytes, err := os.ReadFile(authFile); err == nil {
			if trimmed := strings.TrimSpace(string(tokenBytes)); trimmed != "" {
				return trimmed
			}
		}
	}
	if apiKey := strings.TrimSpace(os.Getenv(deployAPIKeyEnv)); apiKey != "" {
		return apiKey
	}
	return ""
}

func newCLIHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			orig := via[0].URL
			if !strings.EqualFold(req.URL.Scheme, orig.Scheme) || !strings.EqualFold(req.URL.Host, orig.Host) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

func fetchSystemConfig(ctx context.Context, baseURL string, authFile ...string) (SystemConfig, error) {
	var cfg SystemConfig
	base := strings.TrimSuffix(baseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/system/config", nil)
	if err != nil {
		return cfg, fmt.Errorf("failed to build system config request: %w", err)
	}
	if err := applyCustomHeaders(req); err != nil {
		return cfg, err
	}
	var authFilePath string
	if len(authFile) > 0 {
		authFilePath = authFile[0]
	}
	if token := resolveAuthToken(authFilePath); token != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := newCLIHTTPClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return cfg, fmt.Errorf("failed to fetch system config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return cfg, fmt.Errorf("%w: system config request failed with status 401 Unauthorized: please run 'shaper login'", errSystemConfigBlocked)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return cfg, fmt.Errorf("%w: system config request failed with status %s: %s", errSystemConfigBlocked, resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("%w: failed to decode system config: %w", errSystemConfigBlocked, err)
	}
	return cfg, nil
}

type AuthManager struct {
	ctx           context.Context
	baseURL       string
	allowedOrigin string
	authFile      string
	loginRequired bool
	noOpen        bool
	sessionToken  string
	mu            sync.Mutex
}

func NewAuthManager(ctx context.Context, baseURL, authFile string, loginRequired bool) *AuthManager {
	trimmedBase := strings.TrimSuffix(baseURL, "/")
	var allowedOrigin string
	if parsed, err := url.Parse(trimmedBase); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		allowedOrigin = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	} else if err != nil {
		fmt.Println("WARN: Unable to derive origin from dev base URL; falling back to permissive CORS")
		fmt.Println("Error: ", err)
	}
	return &AuthManager{
		ctx:           ctx,
		baseURL:       trimmedBase,
		allowedOrigin: allowedOrigin,
		authFile:      authFile,
		loginRequired: loginRequired,
	}
}

func (a *AuthManager) SetNoOpen(noOpen bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.noOpen = noOpen
}

func (a *AuthManager) LoginRequired() bool {
	return a.loginRequired
}

func (a *AuthManager) EnsureSession() error {
	if !a.loginRequired {
		return nil
	}
	_, err := a.SessionToken()
	return err
}

func (a *AuthManager) SessionToken() (string, error) {
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

	if !a.loginRequired {
		return "", nil
	}

	return "", errors.New("not authenticated: please run 'shaper login'")
}

func (a *AuthManager) Login() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.sessionToken = ""
	token, err := a.promptForLoginLocked()
	if err != nil {
		return err
	}
	a.sessionToken = token
	return nil
}

func (a *AuthManager) promptForLoginLocked() (string, error) {
	fmt.Fprintln(os.Stdout, "Logging in to Shaper...")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start temporary auth server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	tokenCh := make(chan string, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/token" {
				http.NotFound(w, r)
				return
			}

			// Support top-level GET redirect (/token?token=...) to avoid CSP connect-src restrictions
			if r.Method == http.MethodGet {
				token := strings.TrimSpace(r.URL.Query().Get("token"))
				if token == "" {
					http.Error(w, "token required", http.StatusBadRequest)
					return
				}
				select {
				case tokenCh <- token:
					redirectURL := fmt.Sprintf("%s/dev-login?port=%d&status=success", a.baseURL, port)
					http.Redirect(w, r, redirectURL, http.StatusFound)
				default:
					http.Error(w, "already authenticated", http.StatusGone)
				}
				return
			}

			origin := r.Header.Get("Origin")
			if a.allowedOrigin != "" {
				if origin != a.allowedOrigin {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
				setCORSHeaders(w, a.allowedOrigin)
			} else if origin != "" {
				setCORSHeaders(w, origin)
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.Method != http.MethodPost {
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
			fmt.Println("ERROR: Auth callback server failed: ", err)
		}
	}()

	loginURL := fmt.Sprintf("%s/dev-login?port=%d", a.baseURL, port)
	fmt.Fprintf(os.Stdout, "\nDev auth callback listening on port %d\n\n", port)
	if !a.noOpen {
		fmt.Fprintf(os.Stdout, "Opening %s ...\n", loginURL)
		if err := OpenURL(loginURL); err != nil {
			fmt.Fprintf(os.Stdout, "Failed to open browser automatically: %v\nPlease open the URL manually.\n", err)
		}
	} else {
		fmt.Fprintf(os.Stdout, "Please open the login URL manually:\n%s\n", loginURL)
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
	fmt.Fprintf(os.Stdout, "\nAuthentication successful!\n\nToken saved to %s\n", a.authFile)
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

func setCORSHeaders(w http.ResponseWriter, origin string) {
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Vary", "Origin")
}
