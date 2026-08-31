package notifications

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hapyco/dygo/pkg/dygo"
)

func TestEmailJobSendsAndMarksNotification(t *testing.T) {
	records := &emailJobRecords{record: dygo.Record{
		"id": int64(7), "recipient": "person@example.com", "title": "Approved",
		"message": "Your leave was approved.", "send-email": true, "emailed-at": nil,
	}}
	mailer := &recordingMailer{}
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	payload := json.RawMessage(`{"notification-id":7}`)

	err := EmailJob(mailer, func() time.Time { return now })(context.Background(), dygo.JobExecution{Payload: payload, Records: records})
	if err != nil {
		t.Fatalf("EmailJob() error = %v", err)
	}
	if mailer.calls != 1 || mailer.recipient != "person@example.com" {
		t.Fatalf("mailer calls = %d recipient = %q", mailer.calls, mailer.recipient)
	}
	if records.emailedAt != now.Format(time.RFC3339) {
		t.Fatalf("emailed-at = %q, want %q", records.emailedAt, now.Format(time.RFC3339))
	}
}

func TestEmailJobSkipsAlreadySentNotification(t *testing.T) {
	records := &emailJobRecords{record: dygo.Record{
		"id": int64(7), "send-email": true, "emailed-at": "2026-08-31T12:00:00Z",
	}}
	mailer := &recordingMailer{}
	err := EmailJob(mailer, time.Now)(context.Background(), dygo.JobExecution{
		Payload: json.RawMessage(`{"notification-id":7}`), Records: records,
	})
	if err != nil {
		t.Fatalf("EmailJob() error = %v", err)
	}
	if mailer.calls != 0 {
		t.Fatalf("mailer calls = %d, want 0", mailer.calls)
	}
}

type recordingMailer struct {
	calls     int
	recipient string
}

func (m *recordingMailer) Send(_ context.Context, recipient string, _ string, _ string) error {
	m.calls++
	m.recipient = recipient
	return nil
}

type emailJobRecords struct {
	dygo.RecordData
	record    dygo.Record
	emailedAt string
}

func (r *emailJobRecords) Get(context.Context, string, string, int64) (dygo.Record, error) {
	return r.record, nil
}

func (r *emailJobRecords) Update(_ context.Context, _ string, _ string, _ int64, input dygo.RecordInput) (dygo.Record, error) {
	_ = json.Unmarshal(input["emailed-at"], &r.emailedAt)
	r.record["emailed-at"] = r.emailedAt
	return r.record, nil
}
