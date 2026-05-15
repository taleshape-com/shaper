// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ValidateResponse struct {
	Valid    bool   `json:"valid"`
	Duration int64  `json:"duration"`
	Error    string `json:"error,omitempty"`
}

func RunValidateCommand(ctx context.Context, configPath string, args []string) error {
	cfg, err := loadOrPromptConfig(configPath)
	if err != nil {
		return err
	}

	systemCfg, err := fetchSystemConfig(ctx, cfg.URL)
	if err != nil {
		return err
	}

	apiKey := strings.TrimSpace(os.Getenv(deployAPIKeyEnv))

	var client appsRequester
	if apiKey != "" {
		client, err = newAPIKeyClient(cfg.URL, apiKey)
		if err != nil {
			return err
		}
	} else if systemCfg.LoginRequired {
		authFilePath, err := resolveAbsolutePath(defaultAuthFile)
		if err != nil {
			return err
		}
		authManager := NewAuthManager(ctx, cfg.URL, authFilePath, systemCfg.LoginRequired)
		if err := authManager.EnsureSession(); err != nil {
			return err
		}
		client, err = NewAPIClient(ctx, cfg.URL, authManager)
		if err != nil {
			return err
		}
	} else {
		client = newOpenDeployClient(cfg.URL)
	}

	// Figure out files to validate
	var filesToValidate []string
	if len(args) == 0 {
		// Scan directory
		watchDir, err := resolveConfigDirectory(cfg.Directory, configPath)
		if err != nil {
			return fmt.Errorf("failed to resolve directory: %w", err)
		}
		if err := ensureDirExists(watchDir); err != nil {
			return err
		}

		err = filepath.WalkDir(watchDir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			isDashboard := strings.HasSuffix(d.Name(), DASHBOARD_SUFFIX)
			if isDashboard {
				filesToValidate = append(filesToValidate, p)
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		// Walk provided paths
		for _, arg := range args {
			stat, err := os.Stat(arg)
			if err != nil {
				return fmt.Errorf("invalid path %s: %w", arg, err)
			}
			if stat.IsDir() {
				err = filepath.WalkDir(arg, func(p string, d fs.DirEntry, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if d.IsDir() {
						return nil
					}
					isDashboard := strings.HasSuffix(d.Name(), DASHBOARD_SUFFIX)
					if isDashboard {
						filesToValidate = append(filesToValidate, p)
					}
					return nil
				})
				if err != nil {
					return err
				}
			} else {
				isDashboard := strings.HasSuffix(stat.Name(), DASHBOARD_SUFFIX)
				isTask := strings.HasSuffix(stat.Name(), TASK_SUFFIX)
				if isTask {
					return fmt.Errorf("task validation is not supported yet: %s", arg)
				}
				if !isDashboard {
					return fmt.Errorf("file %s is not a dashboard (must end with %s)", arg, DASHBOARD_SUFFIX)
				}
				filesToValidate = append(filesToValidate, arg)
			}
		}
	}

	if len(filesToValidate) == 0 {
		fmt.Println("No dashboards found to validate.")
		return nil
	}

	fmt.Printf("Validating %d dashboards...\n\n", len(filesToValidate))

	hasErrors := false
	var errorsCount int

	for _, p := range filesToValidate {
		contentBytes, err := os.ReadFile(p)
		if err != nil {
			fmt.Printf("ERROR reading %s: %v\n", p, err)
			errorsCount++
			hasErrors = true
			continue
		}

		content := string(contentBytes)

		meta := extractAppMetadata(content)
		if meta.ID == "" {
			fmt.Printf("❌ %s\n   Error: missing shaperid comment. Run `shaper ids` to generate IDs for all apps.\n", p)
			errorsCount++
			hasErrors = true
			continue
		}

		reqBody, err := json.Marshal(map[string]string{
			"type": "dashboard",
			"sql":  content,
		})
		if err != nil {
			return err
		}

		resp, err := client.DoRequest(ctx, http.MethodPost, "/api/validate", reqBody)
		if err != nil {
			fmt.Printf("ERROR communicating with server for %s: %v\n", p, err)
			errorsCount++
			hasErrors = true
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("ERROR from server for %s: status %d\n", p, resp.StatusCode)
			resp.Body.Close()
			errorsCount++
			hasErrors = true
			continue
		}

		var valResp ValidateResponse
		if err := json.NewDecoder(resp.Body).Decode(&valResp); err != nil {
			fmt.Printf("ERROR decoding response for %s: %v\n", p, err)
			resp.Body.Close()
			errorsCount++
			hasErrors = true
			continue
		}
		resp.Body.Close()

		if valResp.Valid {
			fmt.Printf("✅ %s (%dms)\n", p, valResp.Duration)
		} else {
			fmt.Printf("❌ %s (%dms)\n   Error: %s\n", p, valResp.Duration, valResp.Error)
			errorsCount++
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed for %d dashboard(s)", errorsCount)
	}

	fmt.Println("\nAll dashboards are valid!")
	return nil
}
