package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const deployAPIKeyEnv = "SHAPER_DEPLOY_API_KEY"

type LocalDashboard struct {
	ID       string
	Name     string
	Path     string
	Content  string
	FilePath string
}

type deployOperation struct {
	Operation string              `json:"operation"`
	Type      string              `json:"type"`
	Data      deployDashboardData `json:"data"`
}

type deployDashboardData struct {
	ID      string  `json:"id,omitempty"`
	Path    *string `json:"path,omitempty"`
	Name    *string `json:"name,omitempty"`
	Content *string `json:"content,omitempty"`
}

type deployRequest struct {
	Apps []deployOperation `json:"apps"`
}

func RunDeployCommand(ctx context.Context, configPath string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	apiKey := strings.TrimSpace(os.Getenv(deployAPIKeyEnv))
	if apiKey == "" {
		return fmt.Errorf("%s must be set to run shaper deploy", deployAPIKeyEnv)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	if cfg.LastPull == nil {
		return errors.New("config missing lastPull timestamp; run `shaper pull` before deploying")
	}

	watchDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	if err := EnsureDirExists(watchDir); err != nil {
		return err
	}

	localDashboards, err := loadLocalDashboards(watchDir)
	if err != nil {
		return err
	}
	logger.Info("Loaded local dashboards", slog.Int("count", len(localDashboards)))

	client, err := newAPIKeyClient(cfg.URL, apiKey, logger)
	if err != nil {
		return err
	}

	logger.Info("Fetching remote dashboards...", slog.String("url", cfg.URL))
	remoteDashboards, err := fetchAllDashboards(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	logger.Info("Loaded remote dashboards", slog.Int("count", len(remoteDashboards)))

	remoteDashboardsByID := make(map[string]App, len(remoteDashboards))
	for _, dashboard := range remoteDashboards {
		if dashboard.Type == "dashboard" {
			remoteDashboardsByID[dashboard.ID] = dashboard
		}
	}

	if err := ensureRemoteFreshness(remoteDashboards, *cfg.LastPull, client.actor); err != nil {
		return err
	}

	logger.Info("Deploy checkpoint established", slog.Time("last_pull", cfg.LastPull.UTC()))

	ops := buildDeployOperations(localDashboards, remoteDashboards)
	if len(ops) == 0 {
		logger.Info("No changes detected; nothing to deploy")
		return nil
	}

	var createCount, updateCount, deleteCount int
	for _, op := range ops {
		switch op.Operation {
		case "create":
			createCount++
		case "update":
			updateCount++
		case "delete":
			deleteCount++
		}
	}

	logger.Info("Submitting deploy request",
		slog.Int("operations", len(ops)),
		slog.Int("creates", createCount),
		slog.Int("updates", updateCount),
		slog.Int("deletes", deleteCount))

	logDeployChanges(logger, ops, localDashboards, remoteDashboardsByID)

	if err := submitDeploy(ctx, client, ops); err != nil {
		return err
	}

	logger.Info("Deploy completed", slog.Time("timestamp", time.Now().UTC()))
	return nil
}

func loadLocalDashboards(baseDir string) (map[string]LocalDashboard, error) {
	dashboards := make(map[string]LocalDashboard)
	err := filepath.WalkDir(baseDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), DASHBOARD_SUFFIX) {
			return nil
		}

		contentBytes, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", p, err)
		}
		content := string(contentBytes)
		id := extractShaperID(content)
		if id == "" {
			return fmt.Errorf("%s is missing a shaper id comment (-- shaperid:<id>)", p)
		}

		if _, exists := dashboards[id]; exists {
			return fmt.Errorf("duplicate dashboard id %s found in %s and %s", id, dashboards[id].FilePath, p)
		}

		relDir, err := filepath.Rel(baseDir, filepath.Dir(p))
		if err != nil {
			return fmt.Errorf("failed to determine relative path for %s: %w", p, err)
		}

		dashboards[id] = LocalDashboard{
			ID:       id,
			Name:     strings.TrimSuffix(d.Name(), DASHBOARD_SUFFIX),
			Path:     normalizeDashboardPath(relDir),
			Content:  content,
			FilePath: p,
		}
		return nil
	})

	return dashboards, err
}

func normalizeDashboardPath(relDir string) string {
	if relDir == "." {
		return "/"
	}
	normalized := filepath.ToSlash(relDir)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}
	return normalized
}

func ensureRemoteFreshness(remote []App, lastPull time.Time, actor string) error {
	for _, dashboard := range remote {
		if dashboard.Type != "dashboard" {
			continue
		}
		if dashboard.UpdatedAt.After(lastPull) && dashboard.UpdatedBy != actor {
			return fmt.Errorf("remote dashboard %s (%s) was updated after last pull by %s; run `shaper pull` first", dashboard.Name, dashboard.ID, dashboard.UpdatedBy)
		}
	}
	return nil
}

func buildDeployOperations(local map[string]LocalDashboard, remote []App) []deployOperation {
	remoteByID := make(map[string]App, len(remote))
	for _, r := range remote {
		if r.Type == "dashboard" {
			remoteByID[r.ID] = r
		}
	}

	var ops []deployOperation

	localList := make([]LocalDashboard, 0, len(local))
	for _, l := range local {
		localList = append(localList, l)
	}
	sort.Slice(localList, func(i, j int) bool {
		if localList[i].Path == localList[j].Path {
			return localList[i].Name < localList[j].Name
		}
		return localList[i].Path < localList[j].Path
	})

	for _, localDash := range localList {
		if remoteDash, ok := remoteByID[localDash.ID]; ok {
			if dashboardsDiffer(localDash, remoteDash) {
				name := localDash.Name
				path := localDash.Path
				content := localDash.Content
				ops = append(ops, deployOperation{
					Operation: "update",
					Type:      "dashboard",
					Data: deployDashboardData{
						ID:      localDash.ID,
						Name:    &name,
						Path:    &path,
						Content: &content,
					},
				})
			}
			continue
		}

		name := localDash.Name
		path := localDash.Path
		content := localDash.Content
		ops = append(ops, deployOperation{
			Operation: "create",
			Type:      "dashboard",
			Data: deployDashboardData{
				ID:      localDash.ID,
				Name:    &name,
				Path:    &path,
				Content: &content,
			},
		})
	}

	remoteList := make([]App, 0, len(remoteByID))
	for _, r := range remoteByID {
		remoteList = append(remoteList, r)
	}
	sort.Slice(remoteList, func(i, j int) bool {
		if remoteList[i].Path == remoteList[j].Path {
			return remoteList[i].Name < remoteList[j].Name
		}
		return remoteList[i].Path < remoteList[j].Path
	})

	for _, remoteDash := range remoteList {
		if _, ok := local[remoteDash.ID]; ok {
			continue
		}
		id := remoteDash.ID
		ops = append(ops, deployOperation{
			Operation: "delete",
			Type:      "dashboard",
			Data: deployDashboardData{
				ID: id,
			},
		})
	}

	return ops
}

func dashboardsDiffer(local LocalDashboard, remote App) bool {
	if local.Name != remote.Name {
		return true
	}
	if local.Path != normalizeDashboardPath(strings.TrimPrefix(remote.Path, "/")) {
		return true
	}
	return local.Content != remote.Content
}

func logDeployChanges(logger *slog.Logger, ops []deployOperation, local map[string]LocalDashboard, remote map[string]App) {
	for _, op := range ops {
		changeAttrs := []any{
			slog.String("operation", op.Operation),
			slog.String("id", op.Data.ID),
		}

		var (
			currentPath string
			currentName string
		)

		if localDash, ok := local[op.Data.ID]; ok {
			currentPath = localDash.Path
			currentName = localDash.Name
		} else {
			if op.Data.Path != nil {
				currentPath = *op.Data.Path
			}
			if op.Data.Name != nil {
				currentName = *op.Data.Name
			}
		}

		prev, hasPrev := remote[op.Data.ID]
		if currentPath == "" && hasPrev {
			currentPath = prev.Path
		}
		if currentName == "" && hasPrev {
			currentName = prev.Name
		}

		changeAttrs = append(changeAttrs,
			slog.String("path", currentPath),
			slog.String("name", currentName),
		)

		if hasPrev {
			changeAttrs = append(changeAttrs,
				slog.String("previous_path", prev.Path),
				slog.String("previous_name", prev.Name),
				slog.Time("previous_updated_at", prev.UpdatedAt),
			)
		}

		logger.Info("Deploy dashboard change", (changeAttrs)...)
	}
}

func submitDeploy(ctx context.Context, client *apiKeyClient, ops []deployOperation) error {
	body, err := json.Marshal(deployRequest{Apps: ops})
	if err != nil {
		return fmt.Errorf("failed to marshal deploy request: %w", err)
	}

	resp, err := client.DoRequest(ctx, http.MethodPost, "/api/deploy", body)
	if err != nil {
		return fmt.Errorf("deploy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}

	return nil
}

type apiKeyClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	actor      string
	logger     *slog.Logger
}

func newAPIKeyClient(baseURL, apiKey string, logger *slog.Logger) (*apiKeyClient, error) {
	keyID, actor, err := parseAPIKeyActor(apiKey)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("Using API key for deploy", slog.String("key_id", keyID))

	return &apiKeyClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
		actor:  actor,
		logger: logger,
	}, nil
}

func (c *apiKeyClient) DoRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func parseAPIKeyActor(key string) (string, string, error) {
	parts := strings.Split(key, ".")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid API key format; expected {prefix}.{key_id}.{random}")
	}
	keyID := strings.TrimSpace(parts[1])
	if keyID == "" {
		return "", "", fmt.Errorf("invalid API key format; missing key_id component")
	}
	return keyID, fmt.Sprintf("api_key:%s", keyID), nil
}
