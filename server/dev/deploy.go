// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"bytes"
	"context"
	"encoding/json"
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

type LocalApp struct {
	ID            string
	Name          string
	Type          string
	Path          string
	Content       string
	FilePath      string
	SyncTimestamp *time.Time
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

	watchDir, err := resolveConfigDirectory(cfg.Directory, configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	if err := ensureDirExists(watchDir); err != nil {
		return err
	}

	fmt.Printf("Deploying dashboards and tasks...\n\n")
	fmt.Println("Current time: ", time.Now().Format(time.RFC3339))
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
	fmt.Println("Fetching remote apps from", cfg.URL)
	remoteApps, err := fetchAllApps(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch apps: %w", err)
	}
	fmt.Printf("Found %d remote apps.\n", len(remoteApps))

	fmt.Println("Loading apps from folder", watchDir)
	localApps, err := loadLocalApps(watchDir)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d local apps.\n", len(localApps))

	remoteAppsByID := make(map[string]api.App, len(remoteApps))
	for _, app := range remoteApps {
		if app.Type == "dashboard" || app.Type == "task" {
			remoteAppsByID[app.ID] = app
		}
	}

	if err := ensureRemoteFreshness(remoteApps, localApps, client.Actor()); err != nil {
		return err
	}

	ops := buildDeployOperations(localApps, remoteApps)
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
	logDeployChanges(ops, localApps, remoteAppsByID)

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

func loadLocalApps(baseDir string) (map[string]LocalApp, error) {
	apps := make(map[string]LocalApp)
	err := filepath.WalkDir(baseDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		isDashboard := strings.HasSuffix(d.Name(), DASHBOARD_SUFFIX)
		isTask := strings.HasSuffix(d.Name(), TASK_SUFFIX)

		if !isDashboard && !isTask {
			if strings.HasSuffix(d.Name(), ".sql") {
				fmt.Printf("WARNING: %s ends with .sql but not with %s or %s; ignoring\n", p, DASHBOARD_SUFFIX, TASK_SUFFIX)
			}
			return nil
		}

		contentBytes, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", p, err)
		}
		content := string(contentBytes)
		meta := extractAppMetadata(content)
		if meta.ID == "" {
			return fmt.Errorf("%s is missing a shaper id comment (run `shaper ids` to generate)", p)
		}

		if _, exists := apps[meta.ID]; exists {
			return fmt.Errorf("duplicate app id %s found in %s and %s", meta.ID, apps[meta.ID].FilePath, p)
		}

		relDir, err := filepath.Rel(baseDir, filepath.Dir(p))
		if err != nil {
			return fmt.Errorf("failed to determine relative path for %s: %w", p, err)
		}

		appType := "dashboard"
		name := strings.TrimSuffix(d.Name(), DASHBOARD_SUFFIX)
		if isTask {
			appType = "task"
			name = strings.TrimSuffix(d.Name(), TASK_SUFFIX)
		}

		apps[meta.ID] = LocalApp{
			ID:            meta.ID,
			Name:          name,
			Type:          appType,
			Path:          normalizeDashboardPath(relDir),
			Content:       content,
			FilePath:      p,
			SyncTimestamp: meta.SyncTimestamp,
		}
		return nil
	})

	return apps, err
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

// stripAppMetadata removes the ID and SyncTime prefixes, their newlines, and the following empty line from content.
func stripAppMetadata(content string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	inMetadata := true
	hadMetadata := false
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if inMetadata && (strings.HasPrefix(trimmedLine, shaperIDPrefix) || strings.HasPrefix(trimmedLine, shaperSyncPrefix)) {
			hadMetadata = true
			continue
		}
		if inMetadata {
			inMetadata = false
			// Only skip the empty line if we actually stripped some metadata beforehand
			if hadMetadata && trimmedLine == "" && len(newLines) == 0 && i < len(lines)-1 {
				continue
			}
		}
		newLines = append(newLines, line)
	}
	return strings.Join(newLines, "\n")
}

func ensureRemoteFreshness(remote []api.App, local map[string]LocalApp, actor string) error {
	for _, app := range remote {
		if app.Type != "dashboard" && app.Type != "task" {
			continue
		}

		localApp, exists := local[app.ID]
		willBeModified := !exists || appsDiffer(localApp, app)
		if !willBeModified {
			continue
		}

		updatedBy := ""
		if app.UpdatedBy != nil {
			updatedBy = *app.UpdatedBy
		}

		// If the app was updated after our local sync timestamp, we normally want to force a pull first.
		// If no sync timestamp exists locally, we MUST assume it's stale if we want to protect against overwriting Prod.
		isStale := false
		if localApp.SyncTimestamp != nil {
			isStale = app.UpdatedAt.Truncate(time.Second).After(*localApp.SyncTimestamp)
		} else {
			isStale = true
		}

		if isStale {
			return fmt.Errorf("remote app %s (%s) was updated in prod by %s; run `shaper pull` first", app.Name, app.ID, updatedBy)
		}
	}
	return nil
}

func buildDeployOperations(local map[string]LocalApp, remote []api.App) []api.AppRequest {
	remoteByID := make(map[string]api.App, len(remote))
	for _, r := range remote {
		if r.Type == "dashboard" || r.Type == "task" {
			remoteByID[r.ID] = r
		}
	}

	var createOps []api.AppRequest
	var updateOps []api.AppRequest
	var deleteOps []api.AppRequest

	localList := make([]LocalApp, 0, len(local))
	for _, l := range local {
		localList = append(localList, l)
	}
	sort.Slice(localList, func(i, j int) bool {
		if localList[i].Path == localList[j].Path {
			return localList[i].Name < localList[j].Name
		}
		return localList[i].Path < localList[j].Path
	})

	for _, localApp := range localList {
		if remoteApp, ok := remoteByID[localApp.ID]; ok {
			if appsDiffer(localApp, remoteApp) {
				name := localApp.Name
				path := localApp.Path
				content := stripAppMetadata(localApp.Content)
				id := localApp.ID
				updateOps = append(updateOps, api.AppRequest{
					Operation: "update",
					Type:      localApp.Type,
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

		name := localApp.Name
		path := localApp.Path
		content := stripAppMetadata(localApp.Content)
		id := localApp.ID
		createOps = append(createOps, api.AppRequest{
			Operation: "create",
			Type:      localApp.Type,
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

	for _, remoteApp := range remoteList {
		if _, ok := local[remoteApp.ID]; ok {
			continue
		}
		id := remoteApp.ID
		deleteOps = append(deleteOps, api.AppRequest{
			Operation: "delete",
			Type:      remoteApp.Type,
			Data: api.DashboardData{
				ID: &id,
			},
		})
	}

	var ops []api.AppRequest
	ops = append(ops, deleteOps...)
	ops = append(ops, updateOps...)
	ops = append(ops, createOps...)

	return ops
}

func appsDiffer(local LocalApp, remote api.App) bool {
	if local.Name != remote.Name {
		return true
	}
	if local.Path != normalizeDashboardPath(strings.TrimPrefix(remote.Path, "/")) {
		return true
	}
	// Compare content without the ID prefix (local has it, remote doesn't)
	localContent := stripAppMetadata(local.Content)
	return localContent != remote.Content
}

func logDeployChanges(ops []api.AppRequest, local map[string]LocalApp, remote map[string]api.App) {
	for _, op := range ops {
		var (
			currentPath string
			currentName string
			appType     string
		)

		if op.Data.ID != nil {
			if localApp, ok := local[*op.Data.ID]; ok {
				currentPath = localApp.Path
				currentName = localApp.Name
				appType = localApp.Type
			}
		}
		if currentPath == "" && op.Data.Path != nil {
			currentPath = *op.Data.Path
		}
		if currentName == "" && op.Data.Name != nil {
			currentName = *op.Data.Name
		}
		if appType == "" {
			appType = op.Type
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
		if appType == "" && hasPrev {
			appType = prev.Type
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

		suffix := DASHBOARD_SUFFIX
		if appType == "task" {
			suffix = TASK_SUFFIX
		}

		fmt.Printf("%s %s: %s%s%s%s\n", op.Operation, opID, currentPath, currentName, suffix, extra)
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
