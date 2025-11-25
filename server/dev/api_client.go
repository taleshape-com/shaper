package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
	logger     *slog.Logger
}

func NewAPIClient(ctx context.Context, baseURL string, logger *slog.Logger) (*APIClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	client := &APIClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
	if err := client.refreshToken(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *APIClient) refreshToken(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{
		"token": "",
	})
	if err != nil {
		return fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth/token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %s", resp.Status)
	}

	var payload struct {
		JWT string `json:"jwt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}
	if payload.JWT == "" {
		return fmt.Errorf("token response missing jwt")
	}
	c.token = payload.JWT
	c.logger.Info("Obtained JWT for dev mode")
	return nil
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
		return "", c.decodeAPIError(resp)
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
		return c.decodeAPIError(resp)
	}
	return nil
}

func (c *APIClient) authedRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

func (c *APIClient) decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if len(body) == 0 {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}
	return fmt.Errorf("request failed with status %s: %s", resp.Status, strings.TrimSpace(string(body)))
}
