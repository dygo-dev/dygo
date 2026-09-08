package executionactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/actions"
	"github.com/hapyco/dygo/pkg/dygo"
)

func TestRegistrarRegistersCancelAndRetry(t *testing.T) {
	registry, err := actions.NewRegistry([]dygo.EntityActionRegistrar{Registrar()})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v, want nil", err)
	}
	definitions := registry.Definitions("core", "job-execution")
	if len(definitions) != 2 {
		t.Fatalf("Definitions() = %+v, want cancel and retry", definitions)
	}
	cancel := definitions[0]
	if cancel.Name != "cancel" || cancel.Label != "Cancel" || cancel.Selection != dygo.ActionSelectionRecord || !cancel.Danger || cancel.Confirm == "" {
		t.Fatalf("cancel definition = %+v, want record cancel with confirm", cancel)
	}
	retry := definitions[1]
	if retry.Name != "retry" || retry.Label != "Retry" || retry.Selection != dygo.ActionSelectionRecord || retry.Danger || retry.Confirm == "" {
		t.Fatalf("retry definition = %+v, want record retry with confirm", retry)
	}
}

func TestCancelQueuedRejectsNonQueuedExecution(t *testing.T) {
	jobs := &fakeJobData{err: fmt.Errorf(`job execution "12" is running; only queued executions can be cancelled`)}
	_, err := cancelQueued(context.Background(), dygo.EntityActionCall{RecordIDs: []int64{12}, Jobs: jobs})
	assertActionError(t, err, "invalid_request", "only queued executions can be cancelled")
	if jobs.cancelled != "12" {
		t.Fatalf("CancelQueued reference = %q, want 12", jobs.cancelled)
	}
}

func TestCancelQueuedReturnsCancelledExecution(t *testing.T) {
	jobs := &fakeJobData{execution: dygo.JobExecution{ID: 12, AppName: "core", JobName: "send-notification-email", Queue: "default"}}
	result, err := cancelQueued(context.Background(), dygo.EntityActionCall{RecordIDs: []int64{12}, Jobs: jobs})
	if err != nil {
		t.Fatalf("cancelQueued() error = %v, want nil", err)
	}
	got, _ := result.(map[string]any)
	if got["id"] != int64(12) || got["status"] != "cancelled" || got["job-name"] != "send-notification-email" {
		t.Fatalf("cancelQueued() = %#v, want cancelled execution", result)
	}
}

func TestRetryFailedGeneratesIdempotencyKey(t *testing.T) {
	jobs := &fakeJobData{execution: dygo.JobExecution{ID: 20, AppName: "crm", JobName: "send-email", Queue: "default"}}
	result, err := retryFailed(context.Background(), dygo.EntityActionCall{
		RecordIDs: []int64{19},
		Input:     json.RawMessage(`{}`),
		Jobs:      jobs,
	})
	if err != nil {
		t.Fatalf("retryFailed() error = %v, want nil", err)
	}
	if jobs.retried != "19" || jobs.retryKey != "manual-retry:19" {
		t.Fatalf("Retry call = %q %q, want generated key for 19", jobs.retried, jobs.retryKey)
	}
	got, _ := result.(map[string]any)
	if got["id"] != int64(20) || got["status"] != "queued" {
		t.Fatalf("retryFailed() = %#v, want queued retry", result)
	}
}

func TestRetryFailedUsesCallerIdempotencyKey(t *testing.T) {
	jobs := &fakeJobData{execution: dygo.JobExecution{ID: 21, AppName: "crm", JobName: "send-email"}}
	_, err := retryFailed(context.Background(), dygo.EntityActionCall{
		RecordIDs: []int64{19},
		Input:     json.RawMessage(`{"idempotency-key":"studio-retry:19"}`),
		Jobs:      jobs,
	})
	if err != nil {
		t.Fatalf("retryFailed() error = %v, want nil", err)
	}
	if jobs.retryKey != "studio-retry:19" {
		t.Fatalf("Retry key = %q, want caller key", jobs.retryKey)
	}
}

func TestRetryFailedRejectsNonFailedExecution(t *testing.T) {
	jobs := &fakeJobData{err: fmt.Errorf(`job execution "19" is succeeded; only failed executions can be retried`)}
	_, err := retryFailed(context.Background(), dygo.EntityActionCall{RecordIDs: []int64{19}, Jobs: jobs})
	assertActionError(t, err, "invalid_request", "only failed executions can be retried")
}

func TestRetryFailedMapsMissingExecution(t *testing.T) {
	jobs := &fakeJobData{err: fmt.Errorf(`job execution "19" was not found`)}
	_, err := retryFailed(context.Background(), dygo.EntityActionCall{RecordIDs: []int64{19}, Jobs: jobs})
	assertActionError(t, err, "not_found", "was not found")
}

func TestRetryFailedHidesUnexpectedStoreError(t *testing.T) {
	jobs := &fakeJobData{err: fmt.Errorf("load job execution: pq: permission denied for relation job_execution")}
	_, err := retryFailed(context.Background(), dygo.EntityActionCall{RecordIDs: []int64{19}, Jobs: jobs})
	assertActionError(t, err, "internal_error", "Job Execution action failed")
}

func assertActionError(t *testing.T, err error, code string, contains string) {
	t.Helper()
	var actionErr dygo.ActionError
	if err == nil {
		t.Fatalf("error = nil, want ActionError %s", code)
	}
	if !errors.As(err, &actionErr) {
		t.Fatalf("error = %v, want ActionError", err)
	}
	if actionErr.Code != code || !strings.Contains(actionErr.Message, contains) {
		t.Fatalf("ActionError = %+v, want code %s containing %q", actionErr, code, contains)
	}
}

type fakeJobData struct {
	cancelled string
	retried   string
	retryKey  string
	execution dygo.JobExecution
	err       error
}

func (d *fakeJobData) Enqueue(context.Context, string, string, json.RawMessage, dygo.EnqueueOptions) (dygo.JobExecution, error) {
	return dygo.JobExecution{}, fmt.Errorf("enqueue is not used")
}

func (d *fakeJobData) CancelQueued(_ context.Context, reference string) (dygo.JobExecution, error) {
	d.cancelled = reference
	return d.execution, d.err
}

func (d *fakeJobData) Retry(_ context.Context, reference string, idempotencyKey string) (dygo.JobExecution, error) {
	d.retried = reference
	d.retryKey = idempotencyKey
	return d.execution, d.err
}
