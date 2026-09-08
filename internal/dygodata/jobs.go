package dygodata

import (
	"context"
	"encoding/json"
	"time"

	jobstore "github.com/hapyco/dygo/internal/jobs/store"
	"github.com/hapyco/dygo/pkg/dygo"
)

type jobOperator interface {
	Enqueue(context.Context, string, string, json.RawMessage, jobstore.EnqueueOptions) (jobstore.Execution, error)
	CancelQueued(context.Context, string, time.Time) (jobstore.Execution, error)
	Retry(context.Context, string, string) (jobstore.Execution, error)
}

// JobData exposes durable Job operations through the public SDK.
type JobData struct {
	store jobOperator
}

var _ dygo.JobData = JobData{}

// NewJobData returns dygo JobData backed by a durable Job store.
func NewJobData(store jobOperator) JobData {
	return JobData{store: store}
}

// NewJobDataFromBeginner returns dygo JobData backed by the current transaction or pool.
func NewJobDataFromBeginner(beginner jobstore.Beginner) (JobData, error) {
	store, err := jobstore.New(beginner)
	if err != nil {
		return JobData{}, err
	}
	return NewJobData(store), nil
}

// Enqueue creates one durable Job Execution.
func (d JobData) Enqueue(ctx context.Context, appName string, jobName string, payload json.RawMessage, options dygo.EnqueueOptions) (dygo.JobExecution, error) {
	execution, err := d.store.Enqueue(ctx, appName, jobName, payload, jobstore.EnqueueOptions{
		IdempotencyKey: options.IdempotencyKey,
		Priority:       options.Priority,
		RunAfter:       options.RunAfter,
	})
	if err != nil {
		return dygo.JobExecution{}, err
	}
	return d.executionFromStore(execution), nil
}

// CancelQueued marks a queued Job Execution cancelled by id or name.
func (d JobData) CancelQueued(ctx context.Context, reference string) (dygo.JobExecution, error) {
	execution, err := d.store.CancelQueued(ctx, reference, time.Now().UTC())
	if err != nil {
		return dygo.JobExecution{}, err
	}
	return d.executionFromStore(execution), nil
}

// Retry queues a new Job Execution from a failed execution's payload.
func (d JobData) Retry(ctx context.Context, reference string, idempotencyKey string) (dygo.JobExecution, error) {
	execution, err := d.store.Retry(ctx, reference, idempotencyKey)
	if err != nil {
		return dygo.JobExecution{}, err
	}
	return d.executionFromStore(execution), nil
}

func (d JobData) executionFromStore(execution jobstore.Execution) dygo.JobExecution {
	return dygo.JobExecution{
		ID:      execution.ID,
		AppName: execution.AppName,
		JobName: execution.JobName,
		Queue:   execution.Queue,
		Attempt: execution.Attempts,
		Payload: execution.Payload,
		Jobs:    d,
	}
}
