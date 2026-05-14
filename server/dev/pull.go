// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"shaper/server/api"
	"strings"
	"syscall"
	"time"
)

func RunPullCommand(ctx context.Context, configPath, authFile string, skipConfirm bool) error {
	cfg, err := loadOrPromptConfig(configPath)
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

	authFilePath, err := resolveAbsolutePath(authFile)
	if err != nil {
		return fmt.Errorf("failed to resolve auth file path: %w", err)
	}

	fmt.Printf("Pulling app changes...\n\n")

	systemCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	authManager := NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired)
	if err := authManager.EnsureSession(); err != nil {
		return err
	}

	client, err := NewAPIClient(ctx, cfg.URL, authManager)
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	// Fetch all apps and folders from remote
	fmt.Println("Fetching apps and folders from", cfg.URL)
	remoteApps, folders, err := fetchAllAppsAndFolders(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch apps: %w", err)
	}
	fmt.Printf("Found %d remote apps\n", len(remoteApps))
	fmt.Printf("Found %d remote folders\n", len(folders))

	// Build a map of folder paths to their updatedAt timestamps
	folderUpdatedAt := make(map[string]time.Time) // path -> updatedAt
	for _, folder := range folders {
		folderUpdatedAt[folder.Path+folder.Name+"/"] = folder.UpdatedAt
	}

	// Scan local files for existing app IDs
	fmt.Println("Loading apps from folder", watchDir)
	localIDs, err := scanLocalAppIDs(watchDir)
	if err != nil {
		return fmt.Errorf("failed to scan local apps: %w", err)
	}
	fmt.Printf("Found %d local apps.\n", len(localIDs))

	// Compare and categorize
	var toCreate, toUpdate []api.App
	
	// Track file paths to detect duplicates
	seenPaths := make(map[string]string) // path -> app name (for error reporting)

	for _, app := range remoteApps {
		// Check for duplicate file paths
		suffix := DASHBOARD_SUFFIX
		if app.Type == "task" {
			suffix = TASK_SUFFIX
		}
		filePath := filepath.Join(strings.TrimPrefix(app.Path, "/"), sanitizeFileName(app.Name)+suffix)
		if existingName, exists := seenPaths[filePath]; exists {
			return fmt.Errorf("duplicate app name %q in folder %q (conflicts with %q) - please rename one of them before pulling", app.Name, app.Path, existingName)
		}
		seenPaths[filePath] = app.Name

		localPath, existsLocally := localIDs[app.ID]

		if !existsLocally {
			toCreate = append(toCreate, app)
			continue
		}

		// Read local file to check SyncTimestamp and content
		contentBytes, err := os.ReadFile(localPath)
		if err != nil {
			return fmt.Errorf("failed to read local file %s: %w", localPath, err)
		}
		
		content := string(contentBytes)
		meta := extractAppMetadata(content)
		
		// Reconstruct what the local app looks like to check if it differs
		relDir, err := filepath.Rel(watchDir, filepath.Dir(localPath))
		if err != nil {
			return fmt.Errorf("failed to determine relative path for %s: %w", localPath, err)
		}
		
		name := strings.TrimSuffix(filepath.Base(localPath), DASHBOARD_SUFFIX)
		if app.Type == "task" {
			name = strings.TrimSuffix(filepath.Base(localPath), TASK_SUFFIX)
		}

		localApp := LocalApp{
			ID:            meta.ID,
			Name:          name,
			Path:          normalizeDashboardPath(relDir),
			Content:       content,
			SyncTimestamp: meta.SyncTimestamp,
		}

		// We MUST update the local file if it lacks a SyncTimestamp, OR if it's stale, OR if it differs.
		// Without a SyncTimestamp, `deploy` cannot safely protect against overwriting manual Prod changes.
		isStale := meta.SyncTimestamp == nil || app.UpdatedAt.Truncate(time.Second).After(*meta.SyncTimestamp)
		if isStale || appsDiffer(localApp, app) {
			toUpdate = append(toUpdate, app)
		}

		delete(localIDs, app.ID)
	}

	if len(toCreate) == 0 && len(toUpdate) == 0 {
		fmt.Printf("\nNo updates.\n")
		return nil
	}

	// Show summary
	fmt.Println()
	if len(toCreate) > 0 {
		fmt.Printf("Apps to create (%d):\n", len(toCreate))
		for _, d := range toCreate {
			suffix := DASHBOARD_SUFFIX
			if d.Type == "task" {
				suffix = TASK_SUFFIX
			}
			fmt.Printf("  + %s%s\n", filepath.Join(strings.TrimPrefix(d.Path, "/"), d.Name), suffix)
		}
	}
	if len(toUpdate) > 0 {
		fmt.Printf("Apps to update (%d):\n", len(toUpdate))
		for _, d := range toUpdate {
			suffix := DASHBOARD_SUFFIX
			if d.Type == "task" {
				suffix = TASK_SUFFIX
			}
			fmt.Printf("  + %s%s\n", filepath.Join(strings.TrimPrefix(d.Path, "/"), d.Name), suffix)
		}
	}
	fmt.Println()

	// Ask for confirmation unless skipped
	if !skipConfirm {
		fmt.Print("Proceed with pull? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)

		// Set up signal handling for CTRL-C
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGINT)
		defer signal.Stop(sigChan)

		// Channel to receive input
		inputChan := make(chan string, 1)
		errChan := make(chan error, 1)

		// Read input in a goroutine
		go func() {
			input, err := reader.ReadString('\n')
			if err != nil {
				errChan <- err
				return
			}
			inputChan <- input
		}()

		// Wait for either input or signal
		var input string
		select {
		case input = <-inputChan:
			// Got input, continue
		case <-sigChan:
			fmt.Print("\n\nInterrupted\n\n")
			return ErrInterrupted
		case err := <-errChan:
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("\nPull cancelled.")
			return nil
		}
	}

	// Write apps to files
	var writeErrors []error
	expectedPaths := make(map[string]string) // app ID -> expected file path
	for _, app := range append(toCreate, toUpdate...) {
		expectedPath, err := getExpectedFilePath(watchDir, app)
		if err != nil {
			fmt.Printf("ERROR: Failed to determine file path for app '%s': %s\n", app.Name, err)
			writeErrors = append(writeErrors, err)
			continue
		}
		expectedPaths[app.ID] = expectedPath

		if err := writeAppFile(watchDir, app); err != nil {
			fmt.Printf("ERROR: Failed to write app '%s': %s\n", app.Name, err)
			writeErrors = append(writeErrors, err)
			continue
		}
		suffix := DASHBOARD_SUFFIX
		if app.Type == "task" {
			suffix = TASK_SUFFIX
		}
		fmt.Println("Wrote app:", app.Path+app.Name+suffix)
	}

	if len(writeErrors) > 0 {
		return fmt.Errorf("pull completed with %d error(s), lastPull not updated", len(writeErrors))
	}

	// Delete old files that have been moved or renamed
	var deleteErrors []error
	for appID, actualPath := range localIDs {
		expectedPath, exists := expectedPaths[appID]
		if !exists {
			// App was deleted remotely, skip deletion (user might want to keep it)
			continue
		}
		if actualPath != expectedPath {
			// App was moved/renamed, delete old file
			if err := os.Remove(actualPath); err != nil {
				fmt.Printf("ERROR: Failed to delete old app file '%s': %s\n", actualPath, err)
				deleteErrors = append(deleteErrors, err)
			} else {
				fmt.Printf("Deleted old app file: %s\n", actualPath)
			}
		}
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("pull completed with %d deletion error(s), lastPull not updated", len(deleteErrors))
	}

	fmt.Printf("\nPull complete.\n")
	return nil
}

func fetchAllApps(ctx context.Context, requester appsRequester) ([]api.App, error) {
	apps, _, err := fetchAllAppsAndFolders(ctx, requester)
	return apps, err
}

func fetchAllAppsAndFolders(ctx context.Context, requester appsRequester) ([]api.App, []api.App, error) {
	var allApps []api.App
	var allFolders []api.App
	limit := 100
	offset := 0

	for {
		apps, err := fetchAppsPage(ctx, requester, limit, offset)
		if err != nil {
			return nil, nil, err
		}

		for _, app := range apps {
			if app.Type == "dashboard" || app.Type == "task" {
				allApps = append(allApps, app)
			} else if app.Type == "_folder" {
				allFolders = append(allFolders, app)
			}
		}

		// If we got fewer apps than requested, we've reached the end
		if len(apps) < limit {
			break
		}
		offset += limit
	}

	return allApps, allFolders, nil
}

func fetchAppsPage(ctx context.Context, requester appsRequester, limit, offset int) ([]api.App, error) {
	path := fmt.Sprintf("/api/apps?include_content=true&recursive=true&limit=%d&offset=%d", limit, offset)
	resp, err := requester.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}

	var result api.AppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode apps response: %w", err)
	}

	return result.Apps, nil
}

func scanLocalAppIDs(dir string) (map[string]string, error) {
	ids := make(map[string]string) // shaperID -> filePath

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
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

		content, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		if id := extractShaperID(string(content)); id != "" {
			// Convert to absolute path for consistent comparison
			absPath, err := filepath.Abs(p)
			if err != nil {
				// If we can't get absolute path, use the original path
				ids[id] = p
			} else {
				ids[id] = absPath
			}
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

// isAnyParentFolderUpdated checks if any folder in the given path (including the path itself)
// was updated after the lastPull timestamp. It checks all parent paths.
// For example, for path "/folder1/folder2/", it checks:
// - "/folder1/folder2/"
// - "/folder1/"
// - "/"
func isAnyParentFolderUpdated(path string, folderUpdatedAt map[string]time.Time, lastPull time.Time) bool {
	// Normalize path to ensure it ends with /
	normalizedPath := path
	if normalizedPath != "/" && !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Check all parent paths
	currentPath := normalizedPath
	for {
		if updatedAt, exists := folderUpdatedAt[currentPath]; exists {
			if updatedAt.After(lastPull) {
				return true
			}
		}

		// Move to parent path
		if currentPath == "/" {
			break
		}
		// Remove the last segment
		currentPath = strings.TrimSuffix(currentPath, "/")
		lastSlash := strings.LastIndex(currentPath, "/")
		if lastSlash == -1 {
			currentPath = "/"
		} else {
			currentPath = currentPath[:lastSlash+1]
		}
	}

	return false
}

func getExpectedFilePath(baseDir string, app api.App) (string, error) {
	// Construct path (same logic as writeAppFile)
	dashPath := app.Path
	if dashPath == "" {
		dashPath = "/"
	}
	// Remove leading slash for filepath.Join
	dashPath = strings.TrimPrefix(dashPath, "/")

	dirPath := filepath.Join(baseDir, dashPath)
	suffix := DASHBOARD_SUFFIX
	if app.Type == "task" {
		suffix = TASK_SUFFIX
	}
	fileName := sanitizeFileName(app.Name) + suffix
	filePath := filepath.Join(dirPath, fileName)

	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	return absPath, nil
}

func writeAppFile(baseDir string, app api.App) error {
	// Construct path
	dashPath := app.Path
	if dashPath == "" {
		dashPath = "/"
	}
	// Remove leading slash for filepath.Join
	dashPath = strings.TrimPrefix(dashPath, "/")

	dirPath := filepath.Join(baseDir, dashPath)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	suffix := DASHBOARD_SUFFIX
	if app.Type == "task" {
		suffix = TASK_SUFFIX
	}
	fileName := sanitizeFileName(app.Name) + suffix
	filePath := filepath.Join(dirPath, fileName)

	meta := extractAppMetadata(app.Content)
	if meta.ID == "" {
		meta.ID = app.ID
	}
	
	// We want to format the time without nanoseconds so that when we read it back and compare, 
	// it isn't automatically considered "stale" due to nanosecond precision loss during formatting.
	truncatedTime := app.UpdatedAt.Truncate(time.Second)
	content := prependAppMetadata(meta.ID, &truncatedTime, stripAppMetadata(app.Content))

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}
