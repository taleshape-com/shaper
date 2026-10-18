// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestDashboard_DashboardIDRestriction(t *testing.T) {
	e := echo.New()
	app := setupTestApp(t)

	t.Run("CreateDashboard", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPost, "/api/dashboards", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)

			h := CreateDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (validation error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("GetDashboardInfo", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-1/info", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := GetDashboardInfo(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-1/info", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := GetDashboardInfo(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("SaveDashboardName", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/name", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := SaveDashboardName(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/name", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := SaveDashboardName(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (validation error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("SaveDashboardVisibility", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/visibility", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := SaveDashboardVisibility(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/visibility", bytes.NewReader([]byte(`{}`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := SaveDashboardVisibility(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("SaveDashboardPassword", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/password", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := SaveDashboardPassword(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/password", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := SaveDashboardPassword(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (binding error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("SaveDashboardQuery", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/queries/1", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id", "query_id")
			c.SetParamValues("dash-1", "1")

			h := SaveDashboardQuery(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodPut, "/api/dashboards/dash-1/queries/1", bytes.NewReader([]byte(`invalid json`)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id", "query_id")
			c.SetParamValues("dash-1", "1")

			h := SaveDashboardQuery(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 (binding error), got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("DeleteDashboard", func(t *testing.T) {
		t.Run("jwt with dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-456")
			req := httptest.NewRequest(http.MethodDelete, "/api/dashboards/dash-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := DeleteDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodDelete, "/api/dashboards/dash-1", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-1")

			h := DeleteDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("GetDashboard", func(t *testing.T) {
		t.Run("jwt with mismatched dashboardId is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-allowed")
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-other", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-other")

			h := GetDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401 for mismatched dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt with matching dashboardId is not rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-allowed")
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-allowed", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-allowed")

			h := GetDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status for matching dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-allowed", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id")
			c.SetParamValues("dash-allowed")

			h := GetDashboard(app)
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status for user token without dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("RequestDashboardDownload", func(t *testing.T) {
		t.Run("jwt with mismatched dashboardId for csv is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-allowed")
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-other/download/report.csv", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id", "filename")
			c.SetParamValues("dash-other", "report.csv")

			h := RequestDashboardDownload(app, "http://localhost:8080", "2006-01-02")
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401 for mismatched dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt with matching dashboardId for csv is not rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-allowed")
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-allowed/download/report.csv", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id", "filename")
			c.SetParamValues("dash-allowed", "report.csv")

			h := RequestDashboardDownload(app, "http://localhost:8080", "2006-01-02")
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status for matching dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt without dashboardId is not rejected with 401", func(t *testing.T) {
			token := createUserToken()
			req := httptest.NewRequest(http.MethodGet, "/api/dashboards/dash-other/download/report.csv", nil)
			rec := httptest.NewRecorder()
			c := newContextWithUser(e, req, rec, token)
			c.SetParamNames("id", "filename")
			c.SetParamValues("dash-other", "report.csv")

			h := RequestDashboardDownload(app, "http://localhost:8080", "2006-01-02")
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status for user token without dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})

	t.Run("DownloadFileByKey", func(t *testing.T) {
		t.Run("jwt without dashboardId in intent is rejected with 401", func(t *testing.T) {
			token := createUserToken()
			signedJWT := signToken(token, app.JWTSecret)

			intent := DownloadIntent{
				Type:        "csv",
				DashboardID: "dash-target",
				QueryID:     -1,
				JWTToken:    signedJWT,
			}
			intentJSON, _ := json.Marshal(intent)
			_, _ = app.DownloadsKv.Put(context.Background(), "dl-key-1", intentJSON)

			req := httptest.NewRequest(http.MethodGet, "/api/download/dl-key-1/report.csv", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("key", "filename")
			c.SetParamValues("dl-key-1", "report.csv")

			h := DownloadFileByKey(app, "http://localhost:8080", "2006-01-02")
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt with mismatched dashboardId in intent is rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-other")
			signedJWT := signToken(token, app.JWTSecret)

			intent := DownloadIntent{
				Type:        "csv",
				DashboardID: "dash-target",
				QueryID:     -1,
				JWTToken:    signedJWT,
			}
			intentJSON, _ := json.Marshal(intent)
			_, _ = app.DownloadsKv.Put(context.Background(), "dl-key-2", intentJSON)

			req := httptest.NewRequest(http.MethodGet, "/api/download/dl-key-2/report.csv", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("key", "filename")
			c.SetParamValues("dl-key-2", "report.csv")

			h := DownloadFileByKey(app, "http://localhost:8080", "2006-01-02")
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})

		t.Run("jwt with matching dashboardId in intent is not rejected with 401", func(t *testing.T) {
			token := createDashboardScopedToken("dash-target")
			signedJWT := signToken(token, app.JWTSecret)

			intent := DownloadIntent{
				Type:        "csv",
				DashboardID: "dash-target",
				QueryID:     -1,
				JWTToken:    signedJWT,
			}
			intentJSON, _ := json.Marshal(intent)
			_, _ = app.DownloadsKv.Put(context.Background(), "dl-key-3", intentJSON)

			req := httptest.NewRequest(http.MethodGet, "/api/download/dl-key-3/report.csv", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("key", "filename")
			c.SetParamValues("dl-key-3", "report.csv")

			h := DownloadFileByKey(app, "http://localhost:8080", "2006-01-02")
			if err := h(c); err != nil {
				t.Fatalf("handler failed: %v", err)
			}
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("expected non-401 status for matching dashboardId, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	})
}

