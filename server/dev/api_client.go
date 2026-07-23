// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type APIClient struct {
	baseURL     string
	httpClient  *http.Client
	token       string
	tokenExpiry time.Time
	auth        *AuthManager
}

func NewAPIClient(ctx context.Context, baseURL string, auth *AuthManager) (*APIClient, error) {
	client := &APIClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
		auth: auth,
	}
	if err := client.refreshToken(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func isJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && strings.HasPrefix(token, "ey")
}

func (c *APIClient) refreshToken(ctx context.Context) error {
	var storedToken string
	if c.auth != nil && c.auth.LoginRequired() {
		token, err := c.auth.SessionToken()
		if err != nil {
			return err
		}
		storedToken = token
	}

	for attempt := 0; attempt < 2; attempt++ {
		bodyMap := map[string]any{}
		if storedToken != "" && !isJWT(storedToken) {
			bodyMap["token"] = storedToken
		}

		body, err := json.Marshal(bodyMap)
		if err != nil {
			return fmt.Errorf("failed to marshal token request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth/token", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to build token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if storedToken != "" && isJWT(storedToken) {
			req.Header.Set("Authorization", "Bearer "+storedToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to request token: %w", err)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			storedToken = ""
			if c.auth != nil && c.auth.LoginRequired() {
				if err := c.auth.RequireInteractiveLogin(); err != nil {
					return err
				}
				newToken, err := c.auth.SessionToken()
				if err != nil {
					return err
				}
				storedToken = newToken
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			return fmt.Errorf("token request failed with status %s: %s", resp.Status, strings.TrimSpace(string(data)))
		}

		var payload struct {
			JWT string `json:"jwt"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode token response: %w", err)
		}
		resp.Body.Close()

		if payload.JWT == "" {
			return fmt.Errorf("token response missing jwt")
		}
		c.token = payload.JWT
		expiry, err := extractJWTExpiry(payload.JWT)
		if err != nil {
			return fmt.Errorf("failed to parse token expiry: %w", err)
		}
		c.tokenExpiry = expiry
		return nil
	}
	return fmt.Errorf("failed to refresh token after re-authentication")
}

func extractJWTExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode payload: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse payload: %w", err)
	}
	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("missing exp claim")
	}
	return time.Unix(claims.Exp, 0), nil
}

func (c *APIClient) CreateDashboard(ctx context.Context, name, content, folderPath string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"name":      name,
		"content":   content,
		"path":      folderPath,
		"temporary": true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal create dashboard request: %w", err)
	}
	resp, err := c.authedRequest(ctx, http.MethodPost, "/api/dashboards", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", decodeAPIError(resp)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode dashboard response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("dashboard response missing id")
	}
	return result.ID, nil
}

func (c *APIClient) SaveDashboardQuery(ctx context.Context, dashboardID, content string) error {
	body, err := json.Marshal(map[string]string{
		"content": content,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}
	path := fmt.Sprintf("/api/dashboards/%s/query", dashboardID)
	resp, err := c.authedRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	return nil
}

func (c *APIClient) authedRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		c.token = ""
		c.tokenExpiry = time.Time{}
		if c.auth != nil && c.auth.LoginRequired() {
			if err := c.auth.RequireInteractiveLogin(); err != nil {
				return nil, err
			}
			if err := c.refreshToken(ctx); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("request failed with status 401 Unauthorized")
		}

		reqRetry, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to build retry request: %w", err)
		}
		reqRetry.Header.Set("Authorization", "Bearer "+c.token)
		reqRetry.Header.Set("Content-Type", "application/json")
		return c.httpClient.Do(reqRetry)
	}
	return resp, nil
}

func (c *APIClient) DoRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	return c.authedRequest(ctx, method, path, body)
}

func (c *APIClient) ensureValidToken(ctx context.Context) error {
	if c.token == "" || time.Until(c.tokenExpiry) < time.Minute {
		return c.refreshToken(ctx)
	}
	return nil
}

func (c *APIClient) Actor() string {
	if c.token == "" {
		return "no_auth"
	}
	parts := strings.Split(c.token, ".")
	if len(parts) < 2 {
		return "no_auth"
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "no_auth"
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "no_auth"
	}

	if userID, ok := claims["userId"].(string); ok && userID != "" {
		return "user:" + userID
	}
	if apiKeyID, ok := claims["apiKeyId"].(string); ok && apiKeyID != "" {
		return "api_key:" + apiKeyID
	}
	return "no_auth"
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if len(body) == 0 {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}
	return fmt.Errorf("request failed with status %s: %s", resp.Status, strings.TrimSpace(string(body)))
}
