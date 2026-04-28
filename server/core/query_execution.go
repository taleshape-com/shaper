// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/nrednav/cuid2"
)

type QueryExecutionStatus string

const (
	QueryStatusPending    QueryExecutionStatus = "pending"
	QueryStatusRunning    QueryExecutionStatus = "running"
	QueryStatusSuccess    QueryExecutionStatus = "success"
	QueryStatusFailed     QueryExecutionStatus = "failed"
	QueryStatusCancelled  QueryExecutionStatus = "cancelled"
	QueryStatusTimedOut   QueryExecutionStatus = "timed_out"
)

type QueryExecutionType string

const (
	QueryTypeDashboard    QueryExecutionType = "dashboard"
	QueryTypeTask         QueryExecutionType = "task"
	QueryTypeSQLAPI       QueryExecutionType = "sql_api"
	QueryTypeDownload     QueryExecutionType = "download"
)

type QueryExecution struct {
	ID            string                  `json:"id"`
	Type          QueryExecutionType      `json:"type"`
	DashboardID   *string                 `json:"dashboardId,omitempty"`
	TaskID        *string                 `json:"taskId,omitempty"`
	UserID        *string                 `json:"userId,omitempty"`
	APIKeyID      *string                 `json:"apiKeyId,omitempty"`
	QueryIndex    *int                    `json:"queryIndex,omitempty"`
	Query         string                  `json:"query,omitempty"`
	Status        QueryExecutionStatus    `json:"status"`
	StartedAt     time.Time               `json:"startedAt"`
	EndedAt       *time.Time              `json:"endedAt,omitempty"`
	DurationMs    *int64                  `json:"durationMs,omitempty"`
	RowCount      *int64                  `json:"rowCount,omitempty"`
	Error         *string                 `json:"error,omitempty"`
	IsSlowQuery   bool                    `json:"isSlowQuery"`
}

const SLOW_QUERY_THRESHOLD_MS = 1000

type QueryExecutionTracker struct {
	mu          sync.RWMutex
	executions  map[string]*QueryExecution
	maxCapacity int
	onUpdate    func(*QueryExecution)
}

var globalQueryTracker *QueryExecutionTracker
var globalQueryTrackerOnce sync.Once

func GetQueryTracker() *QueryExecutionTracker {
	globalQueryTrackerOnce.Do(func() {
		globalQueryTracker = NewQueryExecutionTracker(1000)
	})
	return globalQueryTracker
}

func NewQueryExecutionTracker(maxCapacity int) *QueryExecutionTracker {
	return &QueryExecutionTracker{
		executions:  make(map[string]*QueryExecution),
		maxCapacity: maxCapacity,
	}
}

func (t *QueryExecutionTracker) SetOnUpdate(fn func(*QueryExecution)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onUpdate = fn
}

func (t *QueryExecutionTracker) Start(
	ctx context.Context,
	queryType QueryExecutionType,
	dashboardID, taskID *string,
	queryIndex *int,
	query string,
) *QueryExecution {
	actor := ActorFromContext(ctx)
	exec := &QueryExecution{
		ID:          cuid2.Generate(),
		Type:        queryType,
		DashboardID: dashboardID,
		TaskID:      taskID,
		QueryIndex:  queryIndex,
		Query:       truncateQuery(query, 500),
		Status:      QueryStatusRunning,
		StartedAt:   time.Now(),
	}

	if actor != nil {
		if actor.Type == ActorUser {
			exec.UserID = &actor.ID
		} else if actor.Type == ActorAPIKey {
			exec.APIKeyID = &actor.ID
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.executions) >= t.maxCapacity {
		t.cleanupOldest()
	}

	t.executions[exec.ID] = exec

	if t.onUpdate != nil {
		t.onUpdate(exec)
	}

	return exec
}

func (t *QueryExecutionTracker) Complete(exec *QueryExecution, rowCount int64, err error) {
	if exec == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	existing, ok := t.executions[exec.ID]
	if !ok {
		return
	}

	now := time.Now()
	existing.EndedAt = &now
	duration := now.Sub(existing.StartedAt).Milliseconds()
	existing.DurationMs = &duration

	if err != nil {
		existing.Status = QueryStatusFailed
		errMsg := err.Error()
		existing.Error = &errMsg
	} else {
		existing.Status = QueryStatusSuccess
		existing.RowCount = &rowCount
	}

	existing.IsSlowQuery = duration >= SLOW_QUERY_THRESHOLD_MS

	if t.onUpdate != nil {
		t.onUpdate(existing)
	}
}

func (t *QueryExecutionTracker) Cancel(exec *QueryExecution) {
	if exec == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	existing, ok := t.executions[exec.ID]
	if !ok {
		return
	}

	now := time.Now()
	existing.EndedAt = &now
	duration := now.Sub(existing.StartedAt).Milliseconds()
	existing.DurationMs = &duration
	existing.Status = QueryStatusCancelled
	existing.IsSlowQuery = duration >= SLOW_QUERY_THRESHOLD_MS

	if t.onUpdate != nil {
		t.onUpdate(existing)
	}
}

func (t *QueryExecutionTracker) Timeout(exec *QueryExecution) {
	if exec == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	existing, ok := t.executions[exec.ID]
	if !ok {
		return
	}

	now := time.Now()
	existing.EndedAt = &now
	duration := now.Sub(existing.StartedAt).Milliseconds()
	existing.DurationMs = &duration
	existing.Status = QueryStatusTimedOut
	existing.IsSlowQuery = true

	if t.onUpdate != nil {
		t.onUpdate(existing)
	}
}

func (t *QueryExecutionTracker) GetRecentExecutions(limit int) []*QueryExecution {
	t.mu.RLock()
	defer t.mu.RUnlock()

	execs := make([]*QueryExecution, 0, len(t.executions))
	for _, exec := range t.executions {
		execs = append(execs, exec)
	}

	return execs
}

func (t *QueryExecutionTracker) GetSlowQueries(limit int) []*QueryExecution {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var slowExecs []*QueryExecution
	for _, exec := range t.executions {
		if exec.IsSlowQuery {
			slowExecs = append(slowExecs, exec)
		}
	}

	return slowExecs
}

func (t *QueryExecutionTracker) GetByID(id string) (*QueryExecution, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	exec, ok := t.executions[id]
	return exec, ok
}

func (t *QueryExecutionTracker) cleanupOldest() {
	var oldestID string
	var oldestTime time.Time

	for id, exec := range t.executions {
		if oldestTime.IsZero() || exec.StartedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = exec.StartedAt
		}
	}

	if oldestID != "" {
		delete(t.executions, oldestID)
	}
}

func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}

func (exec *QueryExecution) GetDuration() time.Duration {
	if exec.DurationMs == nil {
		return time.Since(exec.StartedAt)
	}
	return time.Duration(*exec.DurationMs) * time.Millisecond
}

func (exec *QueryExecution) IsTerminal() bool {
	return exec.Status == QueryStatusSuccess ||
		exec.Status == QueryStatusFailed ||
		exec.Status == QueryStatusCancelled ||
		exec.Status == QueryStatusTimedOut
}

type QueryExecutionOptions struct {
	DashboardID *string
	TaskID      *string
	QueryIndex  *int
	Query       string
}

func StartQueryExecution(
	ctx context.Context,
	queryType QueryExecutionType,
	opts QueryExecutionOptions,
) *QueryExecution {
	tracker := GetQueryTracker()
	return tracker.Start(
		ctx,
		queryType,
		opts.DashboardID,
		opts.TaskID,
		opts.QueryIndex,
		opts.Query,
	)
}

func CompleteQueryExecution(exec *QueryExecution, rowCount int64, err error) {
	tracker := GetQueryTracker()
	if err != nil {
		if IsContextCancelledError(err) {
			tracker.Cancel(exec)
		} else if IsContextTimeoutError(err) {
			tracker.Timeout(exec)
		} else {
			tracker.Complete(exec, rowCount, err)
		}
	} else {
		tracker.Complete(exec, rowCount, nil)
	}
}

func IsContextCancelledError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "context canceled") ||
		strings.Contains(errMsg, "context was canceled")
}

func IsContextTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "context deadline exceeded") ||
		strings.Contains(errMsg, "timeout")
}
