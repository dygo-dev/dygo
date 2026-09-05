package dygodata

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/pkg/dygo"
)

func TestNotificationDataSendPersistsAndEnqueuesEmailOnce(t *testing.T) {
	records := &fakeNotificationRecords{}
	jobs := &fakeNotificationJobs{}
	service := NewNotificationData(records, jobs)
	message := dygo.NotificationMessage{
		Recipient:      "person@example.com",
		Title:          "Leave approved",
		Message:        "Your leave was approved.",
		DeepLink:       "/hr-leave-request/HRL-1",
		Email:          true,
		IdempotencyKey: "hr:approve:HRL-1",
	}

	first, err := service.Send(context.Background(), message)
	if err != nil {
		t.Fatalf("first Send() error = %v", err)
	}
	second, err := service.Send(context.Background(), message)
	if err != nil {
		t.Fatalf("second Send() error = %v", err)
	}
	if !first.Created || second.Created || first.ID != second.ID {
		t.Fatalf("receipts = first %+v second %+v", first, second)
	}
	if records.creates != 1 || jobs.enqueues != 1 {
		t.Fatalf("creates = %d, enqueues = %d, want one each", records.creates, jobs.enqueues)
	}
	if jobs.app != "core" || jobs.job != "send-notification-email" || jobs.key != "notification-email:notice-1" {
		t.Fatalf("job = %s/%s key %q", jobs.app, jobs.job, jobs.key)
	}
}

func TestNotificationDataSendPropagatesLookupFailure(t *testing.T) {
	records := &fakeNotificationRecords{findErr: errors.New("database unavailable")}
	service := NewNotificationData(records, &fakeNotificationJobs{})
	_, err := service.Send(context.Background(), dygo.NotificationMessage{
		Recipient: "person@example.com", Title: "Notice", Message: "Message", IdempotencyKey: "notice-1",
	})
	if err == nil || err.Error() != "database unavailable" {
		t.Fatalf("Send() error = %v, want lookup failure", err)
	}
	if records.creates != 0 {
		t.Fatalf("creates = %d, want no create after lookup failure", records.creates)
	}
}

type fakeNotificationRecords struct {
	dygo.RecordData
	record  dygo.Record
	creates int
	findErr error
}

func (f *fakeNotificationRecords) Lock(context.Context, string, string, dygo.RecordListParams) (dygo.RecordListResult, error) {
	return dygo.RecordListResult{Records: []dygo.Record{{"id": int64(9), "name": "person@example.com"}}}, nil
}

func (f *fakeNotificationRecords) Find(context.Context, string, string, dygo.RecordInput) (dygo.Record, error) {
	if f.record == nil {
		if f.findErr != nil {
			return nil, f.findErr
		}
		return nil, db.RecordError{Code: db.RecordErrorNotFound, Message: "notification not found"}
	}
	return f.record, nil
}

func (f *fakeNotificationRecords) Create(_ context.Context, app string, entity string, input dygo.RecordInput) (dygo.Record, error) {
	if app != "core" || entity != "notification" {
		return nil, errors.New("wrong entity")
	}
	f.creates++
	f.record = dygo.Record{"id": int64(1), "name": "notice-1"}
	for field, raw := range input {
		var value any
		_ = json.Unmarshal(raw, &value)
		f.record[field] = value
	}
	return f.record, nil
}

type fakeNotificationJobs struct {
	app      string
	job      string
	key      string
	enqueues int
}

func (f *fakeNotificationJobs) Enqueue(_ context.Context, app string, job string, _ json.RawMessage, options dygo.EnqueueOptions) (dygo.JobExecution, error) {
	f.app, f.job, f.key = app, job, options.IdempotencyKey
	f.enqueues++
	return dygo.JobExecution{ID: 1}, nil
}
