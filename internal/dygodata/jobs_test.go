package dygodata

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	jobstore "github.com/hapyco/dygo/internal/jobs/store"
	"github.com/hapyco/dygo/pkg/dygo"
)

func TestJobDataEnqueueMapsSDKOptions(t *testing.T) {
	runAfter := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	store := &fakeJobOperator{
		execution: jobstore.Execution{
			ID:       42,
			AppName:  "crm",
			JobName:  "send-email",
			Queue:    "default",
			Attempts: 0,
			Payload:  json.RawMessage(`{"email":"a@example.com"}`),
		},
	}

	execution, err := NewJobData(store).Enqueue(context.Background(), "crm", "send-email", json.RawMessage(`{"email":"a@example.com"}`), dygo.EnqueueOptions{
		IdempotencyKey: "email:1",
		Priority:       10,
		RunAfter:       runAfter,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v, want nil", err)
	}
	if store.appName != "crm" || store.jobName != "send-email" || string(store.payload) != `{"email":"a@example.com"}` {
		t.Fatalf("store call = %s/%s %s, want crm/send-email payload", store.appName, store.jobName, store.payload)
	}
	if store.options.IdempotencyKey != "email:1" || store.options.Priority != 10 || !store.options.RunAfter.Equal(runAfter) {
		t.Fatalf("store options = %+v, want SDK enqueue options", store.options)
	}
	if execution.ID != 42 || execution.AppName != "crm" || execution.JobName != "send-email" || execution.Queue != "default" || string(execution.Payload) != `{"email":"a@example.com"}` || execution.Jobs == nil {
		t.Fatalf("SDK execution = %+v, want mapped execution with Jobs service", execution)
	}
}

func TestJobDataCancelQueuedUsesStore(t *testing.T) {
	store := &fakeJobOperator{
		execution: jobstore.Execution{ID: 7, AppName: "core", JobName: "send-notification-email", Queue: "default"},
	}
	execution, err := NewJobData(store).CancelQueued(context.Background(), "7")
	if err != nil {
		t.Fatalf("CancelQueued() error = %v, want nil", err)
	}
	if store.cancelReference != "7" || store.cancelAt.IsZero() {
		t.Fatalf("CancelQueued store call = %q %v, want id 7 and a timestamp", store.cancelReference, store.cancelAt)
	}
	if execution.ID != 7 || execution.AppName != "core" || execution.JobName != "send-notification-email" || execution.Jobs == nil {
		t.Fatalf("SDK execution = %+v, want cancelled execution with Jobs service", execution)
	}
}

func TestJobDataRetryUsesStore(t *testing.T) {
	store := &fakeJobOperator{
		execution: jobstore.Execution{ID: 9, AppName: "crm", JobName: "send-email", Queue: "default"},
	}
	execution, err := NewJobData(store).Retry(context.Background(), "8", "manual-retry:8")
	if err != nil {
		t.Fatalf("Retry() error = %v, want nil", err)
	}
	if store.retryReference != "8" || store.retryKey != "manual-retry:8" {
		t.Fatalf("Retry store call = %q %q, want failed id and key", store.retryReference, store.retryKey)
	}
	if execution.ID != 9 || execution.JobName != "send-email" || execution.Jobs == nil {
		t.Fatalf("SDK execution = %+v, want queued retry with Jobs service", execution)
	}
}

func TestJobDataRetryReturnsStoreError(t *testing.T) {
	store := &fakeJobOperator{err: fmt.Errorf("job execution %q is queued; only failed executions can be retried", "8")}
	_, err := NewJobData(store).Retry(context.Background(), "8", "retry-key")
	if err == nil {
		t.Fatal("Retry() error = nil, want store error")
	}
	if err.Error() != store.err.Error() {
		t.Fatalf("Retry() error = %v, want store error", err)
	}
}

type fakeJobOperator struct {
	appName         string
	jobName         string
	payload         json.RawMessage
	options         jobstore.EnqueueOptions
	cancelReference string
	cancelAt        time.Time
	retryReference  string
	retryKey        string
	execution       jobstore.Execution
	err             error
}

func (s *fakeJobOperator) Enqueue(_ context.Context, appName string, jobName string, payload json.RawMessage, options jobstore.EnqueueOptions) (jobstore.Execution, error) {
	s.appName = appName
	s.jobName = jobName
	s.payload = payload
	s.options = options
	return s.execution, s.err
}

func (s *fakeJobOperator) CancelQueued(_ context.Context, reference string, now time.Time) (jobstore.Execution, error) {
	s.cancelReference = reference
	s.cancelAt = now
	return s.execution, s.err
}

func (s *fakeJobOperator) Retry(_ context.Context, reference string, idempotencyKey string) (jobstore.Execution, error) {
	s.retryReference = reference
	s.retryKey = idempotencyKey
	return s.execution, s.err
}
