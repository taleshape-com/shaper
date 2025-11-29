package dev

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
	Path      string    `json:"path"`
}

type AppsResponse struct {
	Apps       []App `json:"apps"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalCount int   `json:"totalCount"`
}

func RunPullCommand(ctx context.Context, configPath, authFile string, logger *slog.Logger) error {
	cfg, err := LoadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	watchDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	if err := EnsureDirExists(watchDir); err != nil {
		return err
	}

	authFilePath, err := filepath.Abs(authFile)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	systemCfg, err := FetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authManager := NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired, logger)
	if err := authManager.EnsureSession(); err != nil {
		return err
	}

	client, err := NewAPIClient(ctx, cfg.URL, logger, authManager)
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	// Fetch all dashboards from remote
	logger.Info("Fetching dashboards from server...", slog.String("url", cfg.URL))
	remoteDashboards, err := fetchAllDashboards(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	logger.Info("Found remote dashboards", slog.Int("count", len(remoteDashboards)))

	// Scan local files for existing dashboard IDs
	localIDs, err := scanLocalDashboardIDs(watchDir)
	if err != nil {
		return fmt.Errorf("failed to scan local dashboards: %w", err)
	}
	logger.Info("Found local dashboards", slog.String("folder", watchDir), slog.Int("count", len(localIDs)))

	// Compare and categorize
	var toCreate, toUpdate []App
	var maxUpdatedAt time.Time

	// Track file paths to detect duplicates
	seenPaths := make(map[string]string) // path -> dashboard name (for error reporting)

	for _, dashboard := range remoteDashboards {
		if dashboard.UpdatedAt.After(maxUpdatedAt) {
			maxUpdatedAt = dashboard.UpdatedAt
		}

		// Check for duplicate file paths
		filePath := filepath.Join(strings.TrimPrefix(dashboard.Path, "/"), sanitizeFileName(dashboard.Name)+DASHBOARD_SUFFIX)
		if existingName, exists := seenPaths[filePath]; exists {
			return fmt.Errorf("duplicate dashboard name %q in folder %q (conflicts with %q) - please rename one of them before pulling", dashboard.Name, dashboard.Path, existingName)
		}
		seenPaths[filePath] = dashboard.Name

		// Skip dashboards older than lastPull
		if cfg.LastPull != nil && !dashboard.UpdatedAt.After(*cfg.LastPull) {
			continue
		}

		if _, exists := localIDs[dashboard.ID]; exists {
			toUpdate = append(toUpdate, dashboard)
		} else {
			toCreate = append(toCreate, dashboard)
		}
	}

	if len(toCreate) == 0 && len(toUpdate) == 0 {
		fmt.Println("No changes.")
		return nil
	}

	// Show summary
	fmt.Println()
	if len(toCreate) > 0 {
		fmt.Printf("Dashboards to create (%d):\n", len(toCreate))
		for _, d := range toCreate {
			fmt.Printf("  + %s%s\n", filepath.Join(strings.TrimPrefix(d.Path, "/"), d.Name), DASHBOARD_SUFFIX)
		}
	}
	if len(toUpdate) > 0 {
		fmt.Printf("Dashboards to update (%d):\n", len(toUpdate))
		for _, d := range toUpdate {
			fmt.Printf("  + %s%s\n", filepath.Join(strings.TrimPrefix(d.Path, "/"), d.Name), DASHBOARD_SUFFIX)
		}
	}
	fmt.Println()

	// Ask for confirmation
	fmt.Print("Proceed with pull? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		fmt.Println("Pull cancelled.")
		return nil
	}

	// Write dashboards to files
	var writeErrors []error
	for _, dashboard := range append(toCreate, toUpdate...) {
		if err := writeDashboardFile(watchDir, dashboard); err != nil {
			logger.Error("Failed to write dashboard", slog.String("name", dashboard.Name), slog.Any("error", err))
			writeErrors = append(writeErrors, err)
			continue
		}
		logger.Info("Wrote dashboard", slog.String("path", dashboard.Path+dashboard.Name+DASHBOARD_SUFFIX))
	}

	if len(writeErrors) > 0 {
		return fmt.Errorf("pull completed with %d error(s), lastPull not updated", len(writeErrors))
	}

	// Update lastPull timestamp
	cfg.LastPull = &maxUpdatedAt
	if err := SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config with lastPull: %w", err)
	}

	fmt.Printf("\nPull complete. Last pull timestamp: %s\n", maxUpdatedAt.Format(time.RFC3339))
	return nil
}

func fetchAllDashboards(ctx context.Context, requester appsRequester) ([]App, error) {
	var allDashboards []App
	page := 1
	pageSize := 100

	for {
		apps, totalCount, err := fetchAppsPage(ctx, requester, page, pageSize)
		if err != nil {
			return nil, err
		}

		for _, app := range apps {
			if app.Type == "dashboard" {
				allDashboards = append(allDashboards, app)
			}
		}

		if page*pageSize >= totalCount {
			break
		}
		page++
	}

	return allDashboards, nil
}

func fetchAppsPage(ctx context.Context, requester appsRequester, page, pageSize int) ([]App, int, error) {
	path := fmt.Sprintf("/api/apps?include_content=true&recursive=true&page=%d&pageSize=%d", page, pageSize)
	resp, err := requester.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, decodeAPIError(resp)
	}

	var result AppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode apps response: %w", err)
	}

	return result.Apps, result.TotalCount, nil
}

func scanLocalDashboardIDs(dir string) (map[string]string, error) {
	ids := make(map[string]string) // shaperID -> filePath

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), DASHBOARD_SUFFIX) {
			return nil
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		if id := extractShaperID(string(content)); id != "" {
			ids[id] = p
		}

		return nil
	})

	return ids, err
}

func extractShaperID(content string) string {
	if !strings.HasPrefix(content, shaperIDPrefix) {
		return ""
	}

	lineEnd := strings.IndexByte(content, '\n')
	firstLine := content
	if lineEnd != -1 {
		firstLine = content[:lineEnd]
	}

	id := strings.TrimPrefix(firstLine, shaperIDPrefix)
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, " \t\r") {
		return ""
	}

	return id
}

func sanitizeFileName(name string) string {
	// Escape slashes and backslashes to prevent path traversal
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return name
}

func writeDashboardFile(baseDir string, dashboard App) error {
	// Construct path
	dashPath := dashboard.Path
	if dashPath == "" {
		dashPath = "/"
	}
	// Remove leading slash for filepath.Join
	dashPath = strings.TrimPrefix(dashPath, "/")

	dirPath := filepath.Join(baseDir, dashPath)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	fileName := sanitizeFileName(dashboard.Name) + DASHBOARD_SUFFIX
	filePath := filepath.Join(dirPath, fileName)

	// Ensure content has shaper ID
	content := dashboard.Content
	if !hasLeadingShaperIDComment(content) {
		content = prependShaperIDComment(dashboard.ID, content)
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}
