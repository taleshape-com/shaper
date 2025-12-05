// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"shaper/server/api"
	"sort"
	"strings"
	"time"
)

const (
	deployAPIKeyEnv = "SHAPER_DEPLOY_API_KEY"
	noAuthActor     = "no_auth"
)

type LocalDashboard struct {
	ID       string
	Name     string
	Path     string
	Content  string
	FilePath string
}

type deployHTTPClient interface {
	appsRequester
	Actor() string
}

func RunDeployCommand(ctx context.Context, configPath string, validateOnly bool) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	watchDir, err := resolveAbsolutePath(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	if err := ensureDirExists(watchDir); err != nil {
		return err
	}

	fmt.Printf("Deploying Shaper dashboards...\n\n")
	fmt.Println("Current time: ", time.Now().Format(time.RFC3339))
	if cfg.LastPull != nil {
		fmt.Println("Last pulled:  ", cfg.LastPull.Format(time.RFC3339))
	} else {
		fmt.Println("Last pulled:  (not set)")
	}
	fmt.Println()

	systemCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	apiKey := strings.TrimSpace(os.Getenv(deployAPIKeyEnv))

	var client deployHTTPClient
	switch {
	case apiKey != "":
		client, err = newAPIKeyClient(cfg.URL, apiKey)
		if err != nil {
			return err
		}
	case !systemCfg.LoginRequired:
		client = newOpenDeployClient(cfg.URL)
	default:
		return fmt.Errorf("%s must be set to run shaper deploy when login is required", deployAPIKeyEnv)
	}
	fmt.Println("Fetching remote dashboards from", cfg.URL)
	remoteDashboards, err := fetchAllDashboards(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	fmt.Printf("Found %d remote dashboards.\n", len(remoteDashboards))

	// Count dashboard-type apps
	remoteDashboardCount := 0
	for _, dashboard := range remoteDashboards {
		if dashboard.Type == "dashboard" {
			remoteDashboardCount++
		}
	}

	// Require lastPull only if remote has dashboards
	if remoteDashboardCount > 0 && cfg.LastPull == nil {
		return errors.New("config missing lastPull timestamp; run `shaper pull` before deploying (remote has existing dashboards)")
	}

	fmt.Println("Loading dashboards from folder", watchDir)
	localDashboards, err := loadLocalDashboards(watchDir)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d local dashboards.\n", len(localDashboards))

	remoteDashboardsByID := make(map[string]api.App, len(remoteDashboards))
	for _, dashboard := range remoteDashboards {
		if dashboard.Type == "dashboard" {
			remoteDashboardsByID[dashboard.ID] = dashboard
		}
	}

	// Only check freshness if we have a lastPull timestamp
	if cfg.LastPull != nil {
		if err := ensureRemoteFreshness(remoteDashboards, *cfg.LastPull, client.Actor()); err != nil {
			return err
		}
	}

	ops := buildDeployOperations(localDashboards, remoteDashboards)
	if len(ops) == 0 {
		fmt.Printf("\nNo changes detected; nothing to deploy.\n")
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

	fmt.Printf("\nChanges: create=%d, update=%d, delete=%d\n\n", createCount, updateCount, deleteCount)
	logDeployChanges(ops, localDashboards, remoteDashboardsByID)

	if validateOnly {
		fmt.Printf("\nValidation successful. No changes have been applied (validate-only mode).\n")
		return nil
	}

	if err := submitDeploy(ctx, client, ops); err != nil {
		return err
	}

	fmt.Printf("\nDeploy completed.\n")
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

// stripShaperIDPrefix removes the ID prefix, the newline after it, and the following empty line from content.
// The ID prefix format is "-- shaperid:<id>\n\n" at the start of the content.
func stripShaperIDPrefix(content string) string {
	if !strings.HasPrefix(content, shaperIDPrefix) {
		return content
	}

	// Find the end of the first line (where the newline is)
	lineEnd := strings.IndexByte(content, '\n')
	if lineEnd == -1 {
		// No newline found, return empty string (content is just the ID line)
		return ""
	}

	// Skip the ID line's newline
	remaining := content[lineEnd+1:]

	// Check if there's an empty line after the ID line and skip it too
	if len(remaining) > 0 && remaining[0] == '\n' {
		remaining = remaining[1:]
	}

	return remaining
}

func ensureRemoteFreshness(remote []api.App, lastPull time.Time, actor string) error {
	for _, dashboard := range remote {
		if dashboard.Type != "dashboard" {
			continue
		}
		updatedBy := ""
		if dashboard.UpdatedBy != nil {
			updatedBy = *dashboard.UpdatedBy
		}
		if dashboard.UpdatedAt.After(lastPull) && updatedBy != actor {
			return fmt.Errorf("remote dashboard %s (%s) was updated after last pull by %s; run `shaper pull` first", dashboard.Name, dashboard.ID, updatedBy)
		}
	}
	return nil
}

func buildDeployOperations(local map[string]LocalDashboard, remote []api.App) []api.AppRequest {
	remoteByID := make(map[string]api.App, len(remote))
	for _, r := range remote {
		if r.Type == "dashboard" {
			remoteByID[r.ID] = r
		}
	}

	var ops []api.AppRequest

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
				content := stripShaperIDPrefix(localDash.Content)
				id := localDash.ID
				ops = append(ops, api.AppRequest{
					Operation: "update",
					Type:      "dashboard",
					Data: api.DashboardData{
						ID:      &id,
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
		content := stripShaperIDPrefix(localDash.Content)
		id := localDash.ID
		ops = append(ops, api.AppRequest{
			Operation: "create",
			Type:      "dashboard",
			Data: api.DashboardData{
				ID:      &id,
				Name:    &name,
				Path:    &path,
				Content: &content,
			},
		})
	}

	remoteList := make([]api.App, 0, len(remoteByID))
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
		ops = append(ops, api.AppRequest{
			Operation: "delete",
			Type:      "dashboard",
			Data: api.DashboardData{
				ID: &id,
			},
		})
	}

	return ops
}

func dashboardsDiffer(local LocalDashboard, remote api.App) bool {
	if local.Name != remote.Name {
		return true
	}
	if local.Path != normalizeDashboardPath(strings.TrimPrefix(remote.Path, "/")) {
		return true
	}
	// Compare content without the ID prefix (local has it, remote doesn't)
	localContent := stripShaperIDPrefix(local.Content)
	return localContent != remote.Content
}

func logDeployChanges(ops []api.AppRequest, local map[string]LocalDashboard, remote map[string]api.App) {
	for _, op := range ops {
		var (
			currentPath string
			currentName string
		)

		if op.Data.ID != nil {
			if localDash, ok := local[*op.Data.ID]; ok {
				currentPath = localDash.Path
				currentName = localDash.Name
			}
		}
		if currentPath == "" && op.Data.Path != nil {
			currentPath = *op.Data.Path
		}
		if currentName == "" && op.Data.Name != nil {
			currentName = *op.Data.Name
		}

		var prev api.App
		var hasPrev bool
		if op.Data.ID != nil {
			prev, hasPrev = remote[*op.Data.ID]
		}
		if currentPath == "" && hasPrev {
			currentPath = prev.Path
		}
		if currentName == "" && hasPrev {
			currentName = prev.Name
		}

		extra := ""
		if hasPrev && op.Operation != "delete" {
			extra = " (last_updated=" + prev.UpdatedAt.Format(time.RFC3339)
			if prev.Path != currentPath {
				extra += fmt.Sprintf(", previous_path=%s", prev.Path)
			}
			if prev.Name != currentName {
				extra += fmt.Sprintf(", previous_name=%s", prev.Name)
			}
			extra += ")"
		}

		opID := "unknown"
		if op.Data.ID != nil {
			opID = *op.Data.ID
		}
		fmt.Printf("%s %s: %s%s%s%s\n", op.Operation, opID, currentPath, currentName, DASHBOARD_SUFFIX, extra)
	}
}

func submitDeploy(ctx context.Context, client deployHTTPClient, ops []api.AppRequest) error {
	body, err := json.Marshal(api.Request{Apps: ops})
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
}

func newAPIKeyClient(baseURL, apiKey string) (*apiKeyClient, error) {
	_, actor, err := parseAPIKeyActor(apiKey)
	if err != nil {
		return nil, err
	}

	return &apiKeyClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
		actor:  actor,
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

func (c *apiKeyClient) Actor() string {
	return c.actor
}

type openDeployClient struct {
	baseURL    string
	httpClient *http.Client
}

func newOpenDeployClient(baseURL string) *openDeployClient {
	fmt.Printf("Using unauthenticated deploy client.\n\n")
	return &openDeployClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *openDeployClient) Actor() string {
	return noAuthActor
}

func (c *openDeployClient) DoRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
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
