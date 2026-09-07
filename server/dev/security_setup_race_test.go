package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"shaper/server/core"
	"shaper/server/web/handler"

	"github.com/labstack/echo/v4"
)

func performSetupRequest(app *core.App, email string) (int, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"name":     "Concurrent setup",
		"password": "Correct-Horse-Battery-Staple-123!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, recorder)
	err := handler.Setup(app)(ctx)
	return recorder.Code, err
}

func TestSecurityConcurrentSetupCreatesMultipleAdministrators(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	const requests = 8
	start := make(chan struct{})
	results := make(chan int, requests)
	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			status, err := performSetupRequest(app, fmt.Sprintf("attacker-%d@example.test", i))
			if err != nil {
				results <- 0
				return
			}
			results <- status
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	succeeded := 0
	conflicted := 0
	for status := range results {
		if status == http.StatusOK {
			succeeded++
		} else if status == http.StatusConflict {
			conflicted++
		}
	}

	var stored int
	if err := app.Sqlite.Get(&stored, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}

	t.Logf("concurrent unauthenticated setup attempts: %d; successful responses: %d; conflict responses: %d; stored administrators: %d",
		requests, succeeded, conflicted, stored)
	if succeeded != 1 {
		t.Fatalf("successful setup responses = %d, want 1", succeeded)
	}
	if conflicted != requests-1 {
		t.Fatalf("conflicted setup responses = %d, want %d", conflicted, requests-1)
	}
	if stored != 1 {
		t.Fatalf("stored users = %d, want 1", stored)
	}
}

func TestSecuritySequentialSetupRejectsSecondAdministrator(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	firstStatus, err := performSetupRequest(app, "first@example.test")
	if err != nil || firstStatus != http.StatusOK {
		t.Fatalf("first setup failed: status=%d err=%v", firstStatus, err)
	}
	secondStatus, err := performSetupRequest(app, "second@example.test")
	if err != nil || secondStatus != http.StatusConflict {
		t.Fatalf("second sequential setup status=%d err=%v, want 409", secondStatus, err)
	}

	var stored int
	if err := app.Sqlite.Get(&stored, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	t.Logf("negative control: sequential setup stored %d administrator and rejected the second attempt", stored)
	if stored != 1 {
		t.Fatalf("stored users = %d, want 1", stored)
	}
}

func TestSecuritySetupRollbackOnFailure(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Send request with password > 72 bytes (bcrypt error), which triggers rollback of setup claim
	tooLongPassword := string(bytes.Repeat([]byte("a"), 100))
	body, _ := json.Marshal(map[string]string{
		"email":    "failed@example.test",
		"name":     "Failed setup",
		"password": tooLongPassword,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, recorder)
	_ = handler.Setup(app)(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for invalid password, got %d", recorder.Code)
	}

	var stored int
	if err := app.Sqlite.Get(&stored, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Fatalf("stored users = %d, want 0 after failed setup", stored)
	}

	// 2. A subsequent valid setup request must succeed because the claim was safely rolled back
	status, err := performSetupRequest(app, "recovered@example.test")
	if err != nil || status != http.StatusOK {
		t.Fatalf("subsequent setup failed: status=%d err=%v", status, err)
	}

	if err := app.Sqlite.Get(&stored, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if stored != 1 {
		t.Fatalf("stored users = %d, want 1", stored)
	}
}

