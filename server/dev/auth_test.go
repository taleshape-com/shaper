// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func makeTestJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(map[string]any{
		"userId": "user-1",
		"exp":    exp.Unix(),
	})
	if err != nil {
		t.Fatalf("failed to marshal jwt payload: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + ".sig"
}

func TestFetchSystemConfig_SendsAuthAndCustomHeaders(t *testing.T) {
	var gotAuth string
	var gotCustom string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/config" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotCustom = r.Header.Get("X-Proxy-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"loginRequired":true}`))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	authFile := filepath.Join(tmpDir, ".shaper-auth")
	testToken := makeTestJWT(t, time.Now().Add(time.Hour))
	if err := os.WriteFile(authFile, []byte(testToken+"\n"), 0o600); err != nil {
		t.Fatalf("failed to write auth file: %v", err)
	}

	t.Setenv("SHAPER_HEADERS", "X-Proxy-Token: proxy-secret-123")

	cfg, err := fetchSystemConfig(context.Background(), srv.URL, authFile)
	if err != nil {
		t.Fatalf("fetchSystemConfig failed: %v", err)
	}
	if !cfg.LoginRequired {
		t.Fatalf("expected LoginRequired=true")
	}
	if gotAuth != "Bearer "+testToken {
		t.Fatalf("expected Authorization header %q, got %q", "Bearer "+testToken, gotAuth)
	}
	if gotCustom != "proxy-secret-123" {
		t.Fatalf("expected X-Proxy-Token %q, got %q", "proxy-secret-123", gotCustom)
	}
}

func TestRunLoginCommand_ProceedsWhenSystemConfigBlockedByProxy(t *testing.T) {
	testCases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "401 Unauthorized from proxy",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			},
		},
		{
			name: "403 Forbidden from proxy",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Forbidden", http.StatusForbidden)
			},
		},
		{
			name: "200 HTML login page from proxy redirect",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>Login</body></html>`))
			},
		},
	}

	portRe := regexp.MustCompile(`Dev auth callback listening on port (\d+)`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			tmpDir := t.TempDir()
			cfgPath := filepath.Join(tmpDir, "shaper.json")
			authFile := filepath.Join(tmpDir, ".shaper-auth")
			if err := SaveConfig(cfgPath, Config{URL: srv.URL, Directory: tmpDir}); err != nil {
				t.Fatalf("SaveConfig failed: %v", err)
			}

			origStdout := os.Stdout
			rOut, wOut, err := os.Pipe()
			if err != nil {
				t.Fatalf("pipe failed: %v", err)
			}
			os.Stdout = wOut
			defer func() { os.Stdout = origStdout }()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			errCh := make(chan error, 1)
			go func() {
				errCh <- RunLoginCommand(ctx, cfgPath, "", authFile, true)
			}()

			// Read stdout until the callback port is printed, then post a token to it
			buf := make([]byte, 4096)
			var out strings.Builder
			var callbackPort string
			for callbackPort == "" {
				n, readErr := rOut.Read(buf)
				if n > 0 {
					out.Write(buf[:n])
					if m := portRe.FindStringSubmatch(out.String()); len(m) == 2 {
						callbackPort = m[1]
						break
					}
				}
				if readErr != nil {
					break
				}
			}

			if callbackPort == "" {
				cancel()
				_ = wOut.Close()
				loginErr := <-errCh
				t.Fatalf("expected login callback server to start despite blocked system config, got err=%v, output=%q", loginErr, out.String())
			}

			expectedToken := makeTestJWT(t, time.Now().Add(24*time.Hour))
			body, _ := json.Marshal(map[string]string{"token": expectedToken})
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:"+callbackPort+"/token", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("failed to build callback request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", srv.URL)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("callback request failed: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 OK from callback server, got %d", resp.StatusCode)
			}

			loginErr := <-errCh
			_ = wOut.Close()
			if loginErr != nil {
				t.Fatalf("RunLoginCommand returned error: %v", loginErr)
			}

			savedBytes, err := os.ReadFile(authFile)
			if err != nil {
				t.Fatalf("failed to read saved auth file: %v", err)
			}
			if strings.TrimSpace(string(savedBytes)) != expectedToken {
				t.Fatalf("expected saved token %q, got %q", expectedToken, strings.TrimSpace(string(savedBytes)))
			}
		})
	}
}

func TestParseHeadersEnv(t *testing.T) {
	t.Run("comma-separated Key: Value pairs", func(t *testing.T) {
		hdrs, err := parseHeadersEnv("CF-Access-Client-Id: client-id-1, CF-Access-Client-Secret: secret-123")
		if err != nil {
			t.Fatalf("parseHeadersEnv failed: %v", err)
		}
		if hdrs["CF-Access-Client-Id"] != "client-id-1" {
			t.Errorf("expected CF-Access-Client-Id='client-id-1', got %q", hdrs["CF-Access-Client-Id"])
		}
		if hdrs["CF-Access-Client-Secret"] != "secret-123" {
			t.Errorf("expected CF-Access-Client-Secret='secret-123', got %q", hdrs["CF-Access-Client-Secret"])
		}
	})

	t.Run("JSON format", func(t *testing.T) {
		hdrs, err := parseHeadersEnv(`{"CF-Access-Client-Id": "id-json", "Proxy-Authorization": "Bearer xyz"}`)
		if err != nil {
			t.Fatalf("parseHeadersEnv failed: %v", err)
		}
		if hdrs["CF-Access-Client-Id"] != "id-json" {
			t.Errorf("expected CF-Access-Client-Id='id-json', got %q", hdrs["CF-Access-Client-Id"])
		}
		if hdrs["Proxy-Authorization"] != "Bearer xyz" {
			t.Errorf("expected Proxy-Authorization='Bearer xyz', got %q", hdrs["Proxy-Authorization"])
		}
	})
}

func TestAPIClient_SendsCustomHeadersAndBearerTokenDirectlyWithoutTokenEndpoint(t *testing.T) {
	t.Setenv("SHAPER_HEADERS", "CF-Access-Client-Id: cf-id-42")
	cliJWT := makeTestJWT(t, time.Now().Add(time.Hour))

	var tokenEndpointCalled bool
	var apiReqCustomHeader string
	var apiReqAuthHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/token":
			tokenEndpointCalled = true
			http.Error(w, "should not be called when stored token is already a JWT", http.StatusInternalServerError)
		case "/api/schema":
			apiReqCustomHeader = r.Header.Get("CF-Access-Client-Id")
			apiReqAuthHeader = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"databases":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	authFile := filepath.Join(tmpDir, ".shaper-auth")
	if err := os.WriteFile(authFile, []byte(cliJWT+"\n"), 0o600); err != nil {
		t.Fatalf("failed to write auth file: %v", err)
	}

	auth := NewAuthManager(context.Background(), srv.URL, authFile, true)

	client, err := NewAPIClient(context.Background(), srv.URL, auth)
	if err != nil {
		t.Fatalf("NewAPIClient failed: %v", err)
	}

	resp, err := client.DoRequest(context.Background(), http.MethodGet, "/api/schema", nil)
	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	resp.Body.Close()

	if tokenEndpointCalled {
		t.Errorf("expected /api/auth/token NOT to be called when .shaper-auth contains a JWT")
	}
	if apiReqCustomHeader != "cf-id-42" {
		t.Errorf("expected API request CF-Access-Client-Id='cf-id-42', got %q", apiReqCustomHeader)
	}
	if apiReqAuthHeader != "Bearer "+cliJWT {
		t.Errorf("expected API request Authorization=%q, got %q", "Bearer "+cliJWT, apiReqAuthHeader)
	}

	t.Run("expired JWT fails locally without calling /api/auth/token", func(t *testing.T) {
		tokenEndpointCalled = false
		expiredJWT := makeTestJWT(t, time.Now().Add(-time.Hour))
		expiredFile := filepath.Join(tmpDir, ".shaper-auth-expired")
		if err := os.WriteFile(expiredFile, []byte(expiredJWT+"\n"), 0o600); err != nil {
			t.Fatalf("failed to write expired auth file: %v", err)
		}
		expiredAuth := NewAuthManager(context.Background(), srv.URL, expiredFile, true)
		_, err := NewAPIClient(context.Background(), srv.URL, expiredAuth)
		if err == nil {
			t.Fatal("expected error for expired JWT, got nil")
		}
		if tokenEndpointCalled {
			t.Errorf("expected /api/auth/token NOT to be called for expired JWT")
		}
	})
}

func TestCLIHTTPClient_DoesNotLeakHeadersOnCrossOriginRedirect(t *testing.T) {
	t.Setenv("SHAPER_HEADERS", "CF-Access-Client-Secret: super-secret")

	var externalHit bool
	externalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer externalSrv.Close()

	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, externalSrv.URL+"/login", http.StatusFound)
	}))
	defer proxySrv.Close()

	_, err := fetchSystemConfig(context.Background(), proxySrv.URL, "")
	if err == nil {
		t.Fatal("expected error when proxy redirects to external login page")
	}
	if !isAuthOrProxyBlockError(err) {
		t.Fatalf("expected isAuthOrProxyBlockError(err)=true, got false (err=%v)", err)
	}
	if externalHit {
		t.Fatal("expected CLI HTTP client NOT to follow cross-origin redirect or leak custom headers")
	}
}

