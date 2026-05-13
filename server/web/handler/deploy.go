// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"shaper/server/core"
	"shaper/server/api"
	"strings"

	"github.com/labstack/echo/v4"
)

type deployResult struct {
	Operation string `json:"operation"`
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Status    string `json:"status"`
}

func Deploy(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req api.Request
		if err := c.Bind(&req); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "invalid request body"}, "  ")
		}

		if len(req.Apps) == 0 {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "apps array is required"}, "  ")
		}

		ctx := c.Request().Context()
		results := make([]deployResult, 0, len(req.Apps))

		for idx, item := range req.Apps {
			result, err := processDeployOperation(ctx, app, idx, item)
			if err != nil {
				c.Logger().Error("deploy operation failed",
					slog.Int("index", idx),
					slog.Any("error", err),
				)
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			results = append(results, result)
		}

		return c.JSONPretty(http.StatusOK, struct {
			Results []deployResult `json:"results"`
		}{Results: results}, "  ")
	}
}

func processDeployOperation(ctx context.Context, app *core.App, idx int, req api.AppRequest) (deployResult, error) {
	appType := strings.ToLower(strings.TrimSpace(req.Type))
	switch appType {
	case "dashboard", "task":
	default:
		return deployResult{}, fmt.Errorf("apps[%d]: unsupported type %q", idx, req.Type)
	}

	switch strings.ToLower(strings.TrimSpace(req.Operation)) {
	case "create":
		return handleDeployCreate(ctx, app, idx, req.Data, appType)
	case "update":
		return handleDeployUpdate(ctx, app, idx, req.Data, appType)
	case "delete":
		return handleDeployDelete(ctx, app, idx, req.Data, appType)
	default:
		return deployResult{}, fmt.Errorf("apps[%d]: unsupported operation %q", idx, req.Operation)
	}
}

func handleDeployCreate(ctx context.Context, app *core.App, idx int, data api.DashboardData, appType string) (deployResult, error) {
	if data.Name == nil || strings.TrimSpace(*data.Name) == "" {
		return deployResult{}, fmt.Errorf("apps[%d]: name is required for create operations", idx)
	}
	if data.Path == nil {
		return deployResult{}, fmt.Errorf("apps[%d]: path is required for create operations", idx)
	}
	if data.Content == nil {
		return deployResult{}, fmt.Errorf("apps[%d]: content is required for create operations", idx)
	}

	name := strings.TrimSpace(*data.Name)
	content := *data.Content
	path, err := ensureFolderPathExists(ctx, app, *data.Path)
	if err != nil {
		return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
	}

	var requestedID string
	if data.ID != nil {
		requestedID = strings.TrimSpace(*data.ID)
		if requestedID == "" {
			return deployResult{}, fmt.Errorf("apps[%d]: id cannot be empty when provided", idx)
		}
	}

	var id string
	if appType == "task" {
		id, err = core.CreateTask(app, ctx, name, content, path, requestedID)
	} else {
		id, err = core.CreateDashboard(app, ctx, name, content, path, false, requestedID)
	}
	if err != nil {
		return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
	}

	return deployResult{
		Operation: "create",
		Type:      appType,
		ID:        id,
		Status:    "created",
	}, nil
}

func handleDeployUpdate(ctx context.Context, app *core.App, idx int, data api.DashboardData, appType string) (deployResult, error) {
	if data.ID == nil || strings.TrimSpace(*data.ID) == "" {
		return deployResult{}, fmt.Errorf("apps[%d]: id is required for update operations", idx)
	}

	id := strings.TrimSpace(*data.ID)
	changed := false

	if data.Name != nil {
		name := strings.TrimSpace(*data.Name)
		if name == "" {
			return deployResult{}, fmt.Errorf("apps[%d]: name cannot be empty when provided", idx)
		}
		var err error
		if appType == "task" {
			err = core.SaveTaskName(app, ctx, id, name)
		} else {
			err = core.SaveDashboardName(app, ctx, id, name)
		}
		if err != nil {
			return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
		}
		changed = true
	}

	if data.Content != nil {
		var err error
		if appType == "task" {
			err = core.SaveTaskContent(app, ctx, id, *data.Content)
		} else {
			err = core.SaveDashboardQuery(app, ctx, id, *data.Content)
		}
		if err != nil {
			return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
		}
		changed = true
	}

	var appInfo core.Dashboard // using Dashboard struct to reuse existing path handling logic
	var haveAppInfo bool
	getAppInfo := func() (core.Dashboard, error) {
		if haveAppInfo {
			return appInfo, nil
		}
		var err error
		if appType == "task" {
			var task core.Task
			task, err = core.GetTask(app, ctx, id)
			appInfo.Path = task.Path
		} else {
			appInfo, err = core.GetDashboardInfo(app, ctx, id)
		}
		if err != nil {
			return core.Dashboard{}, err
		}
		haveAppInfo = true
		return appInfo, nil
	}

	if data.Path != nil {
		desiredPath, err := ensureFolderPathExists(ctx, app, *data.Path)
		if err != nil {
			return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
		}
		info, err := getAppInfo()
		if err != nil {
			return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
		}
		currentPath := normalizeFolderPath(info.Path)
		if desiredPath != currentPath {
			var err error
			moveReq := core.MoveItemsRequest{
				Apps: []string{id},
				Path: desiredPath,
			}
			err = core.MoveItems(app, ctx, moveReq)
			if err != nil {
				return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
			}
			changed = true
		}
	}

	if !changed {
		return deployResult{}, fmt.Errorf("apps[%d]: no updates provided", idx)
	}

	return deployResult{
		Operation: "update",
		Type:      appType,
		ID:        id,
		Status:    "updated",
	}, nil
}

func handleDeployDelete(ctx context.Context, app *core.App, idx int, data api.DashboardData, appType string) (deployResult, error) {
	if data.ID == nil || strings.TrimSpace(*data.ID) == "" {
		return deployResult{}, fmt.Errorf("apps[%d]: id is required for delete operations", idx)
	}

	id := strings.TrimSpace(*data.ID)
	var err error
	if appType == "task" {
		err = core.DeleteTask(app, ctx, id)
	} else {
		err = core.DeleteDashboard(app, ctx, id)
	}
	if err != nil {
		return deployResult{}, fmt.Errorf("apps[%d]: %w", idx, err)
	}

	return deployResult{
		Operation: "delete",
		Type:      appType,
		ID:        id,
		Status:    "deleted",
	}, nil
}

func ensureFolderPathExists(ctx context.Context, app *core.App, rawPath string) (string, error) {
	normalized := normalizeFolderPath(rawPath)
	if normalized == "/" {
		return normalized, nil
	}

	trimmed := strings.Trim(normalized, "/")
	if trimmed == "" {
		return normalized, nil
	}

	segments := strings.Split(trimmed, "/")
	for i := range segments {
		subPath := "/" + strings.Join(segments[:i+1], "/") + "/"
		if _, err := core.ResolveFolderPath(app, ctx, subPath); err == nil {
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}

		parentPath := "/"
		if i > 0 {
			parentPath = "/" + strings.Join(segments[:i], "/") + "/"
		}

		if _, err := core.CreateFolder(app, ctx, core.CreateFolderRequest{
			Name: segments[i],
			Path: parentPath,
		}); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "already exists") {
				continue
			}
			return "", err
		}
	}

	return normalized, nil
}

func normalizeFolderPath(rawPath string) string {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "/"
	}
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Trim(path, "/")
	if path == "" {
		return "/"
	}

	var segments []string
	for _, segment := range strings.Split(path, "/") {
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}

	if len(segments) == 0 {
		return "/"
	}

	return "/" + strings.Join(segments, "/") + "/"
}
