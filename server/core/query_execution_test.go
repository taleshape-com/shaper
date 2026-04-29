// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestQueryExecutionTracker_StartAndComplete(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	if exec.Status != QueryStatusRunning {
		t.Errorf("Expected status running, got %s", exec.Status)
	}
	if exec.ID == "" {
		t.Error("Expected non-empty ID")
	}

	tracker.Complete(exec, 10, nil)

	if exec.Status != QueryStatusSuccess {
		t.Errorf("Expected status success, got %s", exec.Status)
	}
	if exec.RowCount == nil || *exec.RowCount != 10 {
		t.Errorf("Expected rowCount 10, got %v", exec.RowCount)
	}
	if exec.DurationMs == nil {
		t.Error("Expected durationMs to be set")
	}
}

func TestQueryExecutionTracker_CompleteWithError(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	tracker.Complete(exec, 0, context.DeadlineExceeded)

	if exec.Status != QueryStatusFailed {
		t.Errorf("Expected status failed, got %s", exec.Status)
	}
	if exec.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestQueryExecutionTracker_Cancel(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	tracker.Cancel(exec)

	if exec.Status != QueryStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", exec.Status)
	}
}

func TestQueryExecutionTracker_Timeout(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	tracker.Timeout(exec)

	if exec.Status != QueryStatusTimedOut {
		t.Errorf("Expected status timed_out, got %s", exec.Status)
	}
	if !exec.IsSlowQuery {
		t.Error("Expected isSlowQuery to be true for timeout")
	}
}

func TestQueryExecutionTracker_DuplicateCompleteIgnored(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	tracker.Complete(exec, 10, nil)
	originalEndedAt := exec.EndedAt
	originalDuration := exec.DurationMs

	time.Sleep(10 * time.Millisecond)
	tracker.Complete(exec, 20, context.DeadlineExceeded)

	if exec.Status != QueryStatusSuccess {
		t.Errorf("Expected status to remain success, got %s", exec.Status)
	}
	if exec.RowCount == nil || *exec.RowCount != 10 {
		t.Errorf("Expected rowCount to remain 10, got %v", exec.RowCount)
	}
	if exec.EndedAt != originalEndedAt {
		t.Error("Expected EndedAt to remain unchanged")
	}
	if exec.DurationMs != originalDuration {
		t.Error("Expected DurationMs to remain unchanged")
	}
}

func TestQueryExecutionTracker_DuplicateCancelIgnored(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	tracker.Cancel(exec)
	originalEndedAt := exec.EndedAt

	time.Sleep(10 * time.Millisecond)
	tracker.Complete(exec, 10, nil)

	if exec.Status != QueryStatusCancelled {
		t.Errorf("Expected status to remain cancelled, got %s", exec.Status)
	}
	if exec.EndedAt != originalEndedAt {
		t.Error("Expected EndedAt to remain unchanged")
	}
}

func TestQueryExecutionTracker_GetRecentExecutions(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()

	exec1 := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
	time.Sleep(2 * time.Millisecond)
	exec2 := tracker.Start(ctx, QueryTypeTask, nil, nil, nil, "SELECT 2")
	time.Sleep(2 * time.Millisecond)
	exec3 := tracker.Start(ctx, QueryTypeSQLAPI, nil, nil, nil, "SELECT 3")

	tracker.Complete(exec1, 1, nil)
	tracker.Complete(exec2, 2, nil)
	tracker.Complete(exec3, 3, nil)

	filter := QueryFilter{Limit: 2}
	execs := tracker.GetRecentExecutions(filter)

	if len(execs) != 2 {
		t.Errorf("Expected 2 executions, got %d", len(execs))
	}

	if execs[0].ID != exec3.ID {
		t.Error("Expected most recent execution first")
	}
	if execs[1].ID != exec2.ID {
		t.Error("Expected second most recent execution second")
	}
}

func TestQueryExecutionTracker_GetSlowQueries(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()

	exec1 := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
	exec2 := tracker.Start(ctx, QueryTypeTask, nil, nil, nil, "SELECT 2")

	time.Sleep(time.Duration(SLOW_QUERY_THRESHOLD_MS)*time.Millisecond + 10*time.Millisecond)

	tracker.Complete(exec1, 1, nil)
	tracker.Complete(exec2, 2, nil)

	filter := QueryFilter{Limit: 10}
	slowExecs := tracker.GetSlowQueries(filter)

	if len(slowExecs) != 2 {
		t.Errorf("Expected 2 slow queries, got %d", len(slowExecs))
	}

	for _, exec := range slowExecs {
		if !exec.IsSlowQuery {
			t.Error("Expected isSlowQuery to be true")
		}
	}
}

func TestQueryExecutionTracker_FilterByType(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()

	exec1 := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
	exec2 := tracker.Start(ctx, QueryTypeTask, nil, nil, nil, "SELECT 2")
	exec3 := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 3")

	tracker.Complete(exec1, 1, nil)
	tracker.Complete(exec2, 2, nil)
	tracker.Complete(exec3, 3, nil)

	filter := QueryFilter{
		Types: []QueryExecutionType{QueryTypeDashboard},
		Limit: 10,
	}
	execs := tracker.GetRecentExecutions(filter)

	if len(execs) != 2 {
		t.Errorf("Expected 2 dashboard executions, got %d", len(execs))
	}

	for _, exec := range execs {
		if exec.Type != QueryTypeDashboard {
			t.Errorf("Expected type dashboard, got %s", exec.Type)
		}
	}
}

func TestQueryExecutionTracker_FilterByStatus(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()

	exec1 := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
	exec2 := tracker.Start(ctx, QueryTypeTask, nil, nil, nil, "SELECT 2")
	exec3 := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 3")

	tracker.Complete(exec1, 1, nil)
	tracker.Cancel(exec2)

	filter := QueryFilter{
		Status: []QueryExecutionStatus{QueryStatusRunning},
		Limit:  10,
	}
	execs := tracker.GetRecentExecutions(filter)

	if len(execs) != 1 {
		t.Errorf("Expected 1 running execution, got %d", len(execs))
	}

	if execs[0].ID != exec3.ID {
		t.Error("Expected exec3 to be the running execution")
	}
}

func TestQueryExecutionTracker_IsTerminal(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()

	execRunning := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
	if execRunning.IsTerminal() {
		t.Error("Running execution should not be terminal")
	}

	execSuccess := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 2")
	tracker.Complete(execSuccess, 1, nil)
	if !execSuccess.IsTerminal() {
		t.Error("Success execution should be terminal")
	}

	execFailed := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 3")
	tracker.Complete(execFailed, 0, context.DeadlineExceeded)
	if !execFailed.IsTerminal() {
		t.Error("Failed execution should be terminal")
	}

	execCancelled := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 4")
	tracker.Cancel(execCancelled)
	if !execCancelled.IsTerminal() {
		t.Error("Cancelled execution should be terminal")
	}

	execTimedOut := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 5")
	tracker.Timeout(execTimedOut)
	if !execTimedOut.IsTerminal() {
		t.Error("Timed out execution should be terminal")
	}
}

func TestQueryExecutionTracker_SanitizeForResponse(t *testing.T) {
	userID := "test-user"
	apiKeyID := "test-apikey"
	dashboardID := "test-dashboard"
	queryIndex := 0

	exec := &QueryExecution{
		ID:          "test-id",
		Type:        QueryTypeDashboard,
		DashboardID: &dashboardID,
		UserID:      &userID,
		APIKeyID:    &apiKeyID,
		QueryIndex:  &queryIndex,
		Query:       "SELECT * FROM users WHERE id = 'secret'",
		Status:      QueryStatusSuccess,
		RowCount:    ptrInt64(10),
		DurationMs:  ptrInt64(500),
		IsSlowQuery: false,
	}

	sanitized := exec.SanitizeForResponse()

	if sanitized.Query != "" {
		t.Error("Query should be sanitized (empty)")
	}
	if sanitized.UserID != nil {
		t.Error("UserID should be sanitized (nil)")
	}
	if sanitized.APIKeyID != nil {
		t.Error("APIKeyID should be sanitized (nil)")
	}

	if sanitized.ID != exec.ID {
		t.Error("ID should be preserved")
	}
	if sanitized.Type != exec.Type {
		t.Error("Type should be preserved")
	}
	if sanitized.DashboardID == nil || *sanitized.DashboardID != *exec.DashboardID {
		t.Error("DashboardID should be preserved")
	}
	if sanitized.QueryIndex == nil || *sanitized.QueryIndex != *exec.QueryIndex {
		t.Error("QueryIndex should be preserved")
	}
	if sanitized.Status != exec.Status {
		t.Error("Status should be preserved")
	}
	if sanitized.RowCount == nil || *sanitized.RowCount != *exec.RowCount {
		t.Error("RowCount should be preserved")
	}
	if sanitized.DurationMs == nil || *sanitized.DurationMs != *exec.DurationMs {
		t.Error("DurationMs should be preserved")
	}
	if sanitized.IsSlowQuery != exec.IsSlowQuery {
		t.Error("IsSlowQuery should be preserved")
	}
}

func TestQueryExecutionTracker_ToSummary(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	time.Sleep(2 * time.Millisecond)
	tracker.Complete(exec, 42, nil)

	summary := exec.ToSummary()

	if summary.DurationMs == 0 {
		t.Error("Expected durationMs to be set")
	}
	if summary.RowCount != 42 {
		t.Errorf("Expected rowCount 42, got %d", summary.RowCount)
	}
	if summary.Status != "success" {
		t.Errorf("Expected status success, got %s", summary.Status)
	}
}

func TestIsContextCancelledError(t *testing.T) {
	if !IsContextCancelledError(context.Canceled) {
		t.Error("Expected context.Canceled to be detected as cancelled error")
	}

	if IsContextCancelledError(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded not to be detected as cancelled error")
	}

	if IsContextCancelledError(nil) {
		t.Error("Expected nil not to be detected as cancelled error")
	}
}

func TestIsContextTimeoutError(t *testing.T) {
	if !IsContextTimeoutError(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be detected as timeout error")
	}

	if IsContextTimeoutError(context.Canceled) {
		t.Error("Expected context.Canceled not to be detected as timeout error")
	}

	if IsContextTimeoutError(nil) {
		t.Error("Expected nil not to be detected as timeout error")
	}
}

func TestQueryExecutionTracker_CapacityLimit(t *testing.T) {
	ResetQueryTracker()
	capacity := 5
	tracker := NewQueryExecutionTracker(capacity)

	ctx := context.Background()

	for i := 0; i < capacity*2; i++ {
		exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
		tracker.Complete(exec, int64(i), nil)
	}

	if tracker.Len() > capacity {
		t.Errorf("Expected tracker length <= %d, got %d", capacity, tracker.Len())
	}
}

func TestQueryExecutionTracker_GetByID(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(100)

	ctx := context.Background()
	exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")

	found, ok := tracker.GetByID(exec.ID)
	if !ok {
		t.Error("Expected to find execution by ID")
	}
	if found.ID != exec.ID {
		t.Error("Expected found execution to match")
	}

	_, ok = tracker.GetByID("non-existent-id")
	if ok {
		t.Error("Expected not to find non-existent execution")
	}
}

func TestQueryExecutionTracker_ConcurrentAccess(t *testing.T) {
	ResetQueryTracker()
	tracker := NewQueryExecutionTracker(1000)

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 100
	numOpsPerGoroutine := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				exec := tracker.Start(ctx, QueryTypeDashboard, nil, nil, nil, "SELECT 1")
				tracker.Complete(exec, 1, nil)
			}
		}()
	}

	wg.Wait()

	filter := QueryFilter{Limit: numGoroutines * numOpsPerGoroutine}
	execs := tracker.GetRecentExecutions(filter)

	if len(execs) != numGoroutines*numOpsPerGoroutine {
		t.Errorf("Expected %d executions, got %d", numGoroutines*numOpsPerGoroutine, len(execs))
	}
}

func TestQueryTimeoutConfig(t *testing.T) {
	originalTimeout := GetQueryTimeoutMs()
	defer SetQueryTimeoutMs(originalTimeout)

	newTimeout := 60000
	SetQueryTimeoutMs(newTimeout)

	if GetQueryTimeoutMs() != newTimeout {
		t.Errorf("Expected timeout %d, got %d", newTimeout, GetQueryTimeoutMs())
	}

	SetQueryTimeoutMs(-1)
	if GetQueryTimeoutMs() != newTimeout {
		t.Error("Expected timeout to remain unchanged when setting negative value")
	}
}

func TestWithQueryTimeout(t *testing.T) {
	originalTimeout := GetQueryTimeoutMs()
	defer SetQueryTimeoutMs(originalTimeout)

	SetQueryTimeoutMs(100)

	ctx := context.Background()
	timeoutCtx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Error("Expected deadline to be set")
	}

	if deadline.Before(time.Now()) {
		t.Error("Expected deadline to be in the future")
	}
}

func TestWithQueryTimeout_Disabled(t *testing.T) {
	originalTimeout := GetQueryTimeoutMs()
	defer SetQueryTimeoutMs(originalTimeout)

	SetQueryTimeoutMs(0)

	ctx := context.Background()
	timeoutCtx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	_, ok := timeoutCtx.Deadline()
	if ok {
		t.Error("Expected no deadline when timeout is disabled")
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
