package executionactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/hapyco/dygo/pkg/dygo"
)

const (
	coreApp      = "core"
	jobExecution = "job-execution"
	cancelAction = "cancel"
	retryAction  = "retry"
)

type retryInput struct {
	IdempotencyKey string `json:"idempotency-key"`
}

// Registrar registers Core Job Execution cancel and retry Entity actions.
func Registrar() dygo.EntityActionRegistrar {
	return func(registry dygo.EntityActionRegistry) error {
		if err := registry.RegisterEntity(coreApp, jobExecution, dygo.EntityActionDefinition{
			Name:      cancelAction,
			Label:     "Cancel",
			Selection: dygo.ActionSelectionRecord,
			Confirm:   "This queued Job Execution will not run.",
			Danger:    true,
		}, cancelQueued); err != nil {
			return err
		}
		return registry.RegisterEntity(coreApp, jobExecution, dygo.EntityActionDefinition{
			Name:      retryAction,
			Label:     "Retry",
			Selection: dygo.ActionSelectionRecord,
			Confirm:   "dygo will queue a new Job Execution with the same payload.",
		}, retryFailed)
	}
}

func cancelQueued(ctx context.Context, call dygo.EntityActionCall) (any, error) {
	if call.Jobs == nil {
		return nil, dygo.ActionError{Code: "internal_error", Message: "Job service is unavailable"}
	}
	execution, err := call.Jobs.CancelQueued(ctx, executionReference(call))
	if err != nil {
		return nil, mapJobStoreError(err)
	}
	return executionResult(execution, "cancelled"), nil
}

func retryFailed(ctx context.Context, call dygo.EntityActionCall) (any, error) {
	if call.Jobs == nil {
		return nil, dygo.ActionError{Code: "internal_error", Message: "Job service is unavailable"}
	}
	var input retryInput
	if len(call.Input) > 0 {
		if err := json.Unmarshal(call.Input, &input); err != nil {
			return nil, dygo.ActionError{Code: "invalid_request", Message: "retry input must be valid JSON"}
		}
	}
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = "manual-retry:" + executionReference(call)
	}
	execution, err := call.Jobs.Retry(ctx, executionReference(call), idempotencyKey)
	if err != nil {
		return nil, mapJobStoreError(err)
	}
	return executionResult(execution, "queued"), nil
}

func executionReference(call dygo.EntityActionCall) string {
	if len(call.RecordIDs) == 0 {
		return ""
	}
	return strconv.FormatInt(call.RecordIDs[0], 10)
}

func executionResult(execution dygo.JobExecution, status string) map[string]any {
	return map[string]any{
		"id":       execution.ID,
		"app-name": execution.AppName,
		"job-name": execution.JobName,
		"queue":    execution.Queue,
		"status":   status,
	}
}

func mapJobStoreError(err error) error {
	if err == nil {
		return nil
	}
	var actionErr dygo.ActionError
	if errors.As(err, &actionErr) {
		return actionErr
	}
	message := err.Error()
	if strings.Contains(message, "was not found") {
		return dygo.ActionError{Code: "not_found", Message: "Job Execution was not found"}
	}
	if strings.Contains(message, "only queued executions can be cancelled") ||
		strings.Contains(message, "only failed executions can be retried") ||
		strings.Contains(message, "id or name is required") ||
		strings.Contains(message, "requires an idempotency key") {
		return dygo.ActionError{Code: "invalid_request", Message: message}
	}
	return dygo.ActionError{Code: "internal_error", Message: "Job Execution action failed"}
}
