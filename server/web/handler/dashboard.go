// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"shaper/server/core"
	"shaper/server/pdf"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

type DownloadIntent struct {
	Type        string     `json:"type"`
	DashboardID string     `json:"dashboardId"`
	QueryID     int        `json:"queryId"`
	QueryParams url.Values `json:"queryParams"`
	JWTToken    string     `json:"jwtToken"`
}

var downloadFileTypes = map[string]bool{
	"pdf":  true,
	"csv":  true,
	"xlsx": true,
}

func CreateDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		actor := core.ActorFromContext(c.Request().Context())

		if userToken, ok := c.Get("user").(*jwt.Token); ok {
			claims := userToken.Claims.(jwt.MapClaims)
			if _, hasId := claims["dashboardId"]; hasId {
				return c.JSONPretty(http.StatusUnauthorized,
					struct {
						Error string `json:"error"`
					}{Error: "Unauthorized"}, "  ")
			}
		}

		var request struct {
			Name      string `json:"name"`
			Content   string `json:"content"`
			Path      string `json:"path"`
			Temporary bool   `json:"temporary"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Invalid request"}, "  ")
		}

		// API keys can only create temporary dashboards
		if actor != nil && actor.Type == core.ActorAPIKey && !request.Temporary {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "API keys are only allowed to create temporary dashboards"}, "  ")
		}

		// Allow creating temporary dashboards even if editing is disabled. They are needed during development
		if app.NoEdit && !request.Temporary {
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Editing is disabled"}, "  ")
		}

		// Validate dashboard name
		if !request.Temporary && request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: "Dashboard name is required"}, "  ")
		}

		// Make sure folder exists
		if !request.Temporary {
			_, err := core.ResolveFolderPath(app, c.Request().Context(), request.Path)
			if err != nil {
				return c.JSONPretty(http.StatusBadRequest,
					struct {
						Error string `json:"error"`
					}{Error: err.Error()}, "  ")
			}
		}

		id, err := core.CreateDashboard(app, c.Request().Context(), request.Name, request.Content, request.Path, request.Temporary, "")
		if err != nil {
			c.Logger().Error("error creating dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
		}

		return c.JSONPretty(http.StatusCreated, struct {
			ID string `json:"id"`
		}{
			ID: id,
		}, "  ")
	}
}

func GetDashboardInfo(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		// Embedding JWTs that are fixed to a dashboardId are not allowed to edit the board
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		dashboard, err := core.GetDashboardInfo(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error getting dashboard query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, dashboard)
	}
}

func SaveDashboardName(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		if app.NoEdit {
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Editing is disabled"}, "  ")
		}

		var request struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		if request.Name == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Dashboard name is required"}, "  ")
		}

		err := core.SaveDashboardName(app, c.Request().Context(), c.Param("id"), request.Name)
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error saving dashboard name:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveDashboardVisibility(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		if app.NoEdit {
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Editing is disabled"}, "  ")
		}

		var request struct {
			Visibility string `json:"visibility"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		err := core.SaveDashboardVisibility(app, c.Request().Context(), c.Param("id"), request.Visibility)
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error saving visibility:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveDashboardPassword(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		if app.NoEdit {
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Editing is disabled"}, "  ")
		}

		var request struct {
			Password string `json:"password"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		if request.Password == "" {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Password is required"}, "  ")
		}

		err := core.SaveDashboardPassword(app, c.Request().Context(), c.Param("id"), request.Password)
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error saving dashboard password:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func SaveDashboardQuery(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		// Embedding JWTs that are fixed to a dashboardId are not allowed to edit the board
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		var request struct {
			Content string `json:"content"`
		}
		if err := c.Bind(&request); err != nil {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid request"}, "  ")
		}

		if app.NoEdit && !strings.HasPrefix(c.Param("id"), core.TMP_DASHBOARD_PREFIX) {
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Editing is disabled"}, "  ")
		}

		err := core.SaveDashboardQuery(app, c.Request().Context(), c.Param("id"), request.Content)
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error saving dashboard query:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func GetDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		idParam := c.Param("id")
		variables := map[string]any{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]any)
		}
		idClaim, hasId := claims["dashboardId"]
		if hasId && idClaim != idParam {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		result, err := core.GetDashboard(app, c.Request().Context(), idParam, c.QueryParams(), variables)
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error getting dashboard:", slog.Any("error", err))
			// If the JWT is restricted to a dashboardId, we don't return the actual error to the client.
			// But if the JWT is generic, we return it.
			// In practice this means that if you are logged in and editing dashboards you see error messages, but if a dashboard is embedded or shared publicly you don't.
			errMsg := err.Error()
			if hasId {
				errMsg = "error getting dashboard"
			}
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: errMsg}, "  ")
		}
		return c.JSONPretty(http.StatusOK, result, "  ")
	}
}

func DeleteDashboard(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := c.Get("user").(*jwt.Token).Claims.(jwt.MapClaims)
		if _, hasId := claims["dashboardId"]; hasId {
			return c.JSONPretty(http.StatusUnauthorized,
				struct {
					Error string `json:"error"`
				}{Error: "Unauthorized"}, "  ")
		}
		if app.NoEdit {
			return c.JSONPretty(http.StatusForbidden, struct {
				Error string `json:"error"`
			}{Error: "Editing is disabled"}, "  ")
		}

		err := core.DeleteDashboard(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			if errors.Is(err, core.ErrDashboardNotFound) {
				return c.JSONPretty(http.StatusNotFound, struct {
					Error string `json:"error"`
				}{Error: err.Error()}, "  ")
			}
			c.Logger().Error("error deleting dashboard:", slog.Any("error", err))
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: err.Error()}, "  ")
		}

		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func RequestDashboardDownload(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		jwtToken := c.Get("user").(*jwt.Token)
		claims := jwtToken.Claims.(jwt.MapClaims)
		idParam := c.Param("id")
		filename := c.Param("filename")
		queryVars := c.QueryParam("vars")
		queryId := c.QueryParam("query_id")
		varsAsQueryParams := url.Values{}
		if queryVars != "" {
			err := json.Unmarshal([]byte(queryVars), &varsAsQueryParams)
			if err != nil {
				c.Logger().Error("invalid vars query param:", slog.Any("error", err), slog.String("vars", queryVars))
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: "Invalid vars query parameter"}, "  ")
			}
		}
		if idClaim, hasId := claims["dashboardId"]; hasId && idClaim != idParam {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}

		// type is extension of filename
		fileType := ""
		if parts := strings.Split(filename, "."); len(parts) > 1 {
			fileType = parts[len(parts)-1]
		}
		// assert allowed file types
		if !downloadFileTypes[strings.ToLower(fileType)] {
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid file type"}, "  ")
		}
		queryIdInt := -1
		if queryId != "" {
			var err error
			queryIdInt, err = strconv.Atoi(queryId)
			if err != nil {
				c.Logger().Error("invalid query_id query param:", slog.Any("error", err), slog.String("query_id", queryId))
				return c.JSONPretty(http.StatusBadRequest, struct {
					Error string `json:"error"`
				}{Error: "Invalid query_id query parameter"}, "  ")
			}
		}
		// new claims are same as old, but we set a new exp time and set the dashboardId
		newClaims := jwt.MapClaims{}
		for k, v := range claims {
			newClaims[k] = v
		}
		newClaims["exp"] = time.Now().Add(app.JWTExp).Unix()
		newClaims["dashboardId"] = idParam
		downloadJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
		downloadJWTStr, err := downloadJWT.SignedString(app.JWTSecret)
		if err != nil {
			c.Logger().Error("failed to sign download JWT token:", slog.Any("error", err))
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: "Internal server error"}, "  ")
		}
		intent := DownloadIntent{
			Type:        fileType,
			DashboardID: idParam,
			QueryID:     queryIdInt,
			QueryParams: varsAsQueryParams,
			JWTToken:    downloadJWTStr,
		}

		j, err := json.Marshal(intent)
		if err != nil {
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: "Internal server error"}, "  ")
		}

		token, err := generateDownloadToken()
		if err != nil {
			c.Logger().Error("failed to generate download token:", slog.Any("error", err))
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: "Internal server error"}, "  ")
		}
		_, err = app.DownloadsKv.Put(c.Request().Context(), token, j)
		if err != nil {
			c.Logger().Error("failed to put download intent into KV:", slog.Any("error", err))
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: "Internal server error"}, "  ")
		}

		u := fmt.Sprintf("/api/download/%s/%s", token, filename)

		return c.JSON(http.StatusOK, struct {
			URL string `json:"url"`
		}{
			URL: u,
		})
	}
}

func DownloadFileByKey(app *core.App, internalUrl string, pdfDateFormat string) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := c.Param("key")
		filename := c.Param("filename")

		entry, err := app.DownloadsKv.Get(c.Request().Context(), key)
		if err != nil {
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "Download not found or expired"}, "  ")
		}

		var intent DownloadIntent
		if err := json.Unmarshal(entry.Value(), &intent); err != nil {
			c.Logger().Error("failed to unmarshal download intent:", slog.Any("error", err))
			return c.JSONPretty(http.StatusInternalServerError, struct {
				Error string `json:"error"`
			}{Error: "Internal server error"}, "  ")
		}

		// Set headers based on type
		contentType := "application/octet-stream"
		switch strings.ToLower(intent.Type) {
		case "pdf":
			contentType = "application/pdf"
		case "csv":
			contentType = "text/csv"
		case "xlsx":
			contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		}

		c.Response().Header().Set(echo.HeaderContentType, contentType)
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))

		// Disable response buffering
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		c.Response().Header().Set("Transfer-Encoding", "chunked")
		// Disable caching so CDNSs and such doesn't cache the one-time download URL including the token in the URL
		c.Response().Header().Set("Cache-Control", "public, max-age=0, must-revalidate")

		// Create a writer that writes to the response
		writer := c.Response().Writer

		token, err := jwt.Parse(intent.JWTToken, func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != echojwt.AlgorithmHS256 {
				return nil, &echojwt.TokenError{Token: token, Err: fmt.Errorf("unexpected jwt signing method=%v", token.Header["alg"])}
			}
			return app.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			c.Logger().Error("invalid JWT token in download intent:", slog.Any("error", err))
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		claims := token.Claims.(jwt.MapClaims)
		if id, hasId := claims["dashboardId"]; !hasId || id != intent.DashboardID {
			return c.JSONPretty(http.StatusUnauthorized, struct {
				Error string `json:"error"`
			}{Error: "Unauthorized"}, "  ")
		}
		variables := map[string]any{}
		if vars, hasVariables := claims["variables"]; hasVariables {
			variables = vars.(map[string]any)
		}

		var streamErr error
		switch strings.ToLower(intent.Type) {
		case "pdf":
			streamErr = pdf.StreamDashboardPdf(
				c.Request().Context(),
				app.Logger,
				writer,
				internalUrl,
				pdfDateFormat,
				intent.DashboardID,
				intent.QueryParams,
				variables,
				token,
			)
		case "csv":
			streamErr = core.StreamQueryCSV(
				app,
				c.Request().Context(),
				intent.DashboardID,
				intent.QueryParams,
				intent.QueryID,
				variables,
				writer,
			)
		case "xlsx":
			streamErr = core.StreamQueryXLSX(
				app,
				c.Request().Context(),
				intent.DashboardID,
				intent.QueryParams,
				intent.QueryID,
				variables,
				writer,
			)
		default:
			return c.JSONPretty(http.StatusBadRequest, struct {
				Error string `json:"error"`
			}{Error: "Invalid download type"}, "  ")
		}

		if streamErr != nil {
			// If headers haven't been sent yet, return JSON error
			if c.Response().Committed {
				// If we've already started streaming, log the error since we can't modify the response
				c.Logger().Error("streaming error after response started:", slog.Any("error", err))
				return streamErr
			}
			c.Logger().Error("error downloading file:", slog.Any("error", streamErr))
			return c.JSONPretty(
				http.StatusBadRequest,
				struct {
					Error string `json:"error"`
				}{Error: streamErr.Error()},
				"  ",
			)
		}

		return nil
	}
}

func GetDashboardStatus(app *core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		dashboard, err := core.GetDashboardInfo(app, c.Request().Context(), c.Param("id"))
		if err != nil {
			c.Logger().Error("error getting dashboard status:", slog.Any("error", err))
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "Dashboard Not Found"}, "  ")
		}
		if dashboard.Visibility == nil ||
			(*dashboard.Visibility == "private") ||
			(app.NoPublicSharing && *dashboard.Visibility == "public") ||
			(app.NoPasswordProtectedSharing && *dashboard.Visibility == "password-protected") {
			return c.JSONPretty(http.StatusNotFound, struct {
				Error string `json:"error"`
			}{Error: "Dashboard Not Found"}, "  ")
		}
		return c.JSON(http.StatusOK, struct {
			Visibility string `json:"visibility"`
		}{
			Visibility: *dashboard.Visibility,
		})
	}
}

func generateDownloadToken() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 64 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
