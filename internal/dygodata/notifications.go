package dygodata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hapyco/dygo/internal/db"
	jobstore "github.com/hapyco/dygo/internal/jobs/store"
	notificationstore "github.com/hapyco/dygo/internal/notifications"
	"github.com/hapyco/dygo/pkg/dygo"
)

const notificationEmailJob = "send-notification-email"

// NotificationData persists in-app notifications and their transactional email outbox jobs.
type NotificationData struct {
	records dygo.RecordData
	jobs    dygo.JobData
}

// NewNotificationData creates NotificationData from the transaction-scoped SDK services.
func NewNotificationData(records dygo.RecordData, jobs dygo.JobData) NotificationData {
	return NotificationData{records: records, jobs: jobs}
}

// Send creates one notification or returns the existing notification for the same recipient and key.
func (d NotificationData) Send(ctx context.Context, message dygo.NotificationMessage) (dygo.NotificationReceipt, error) {
	if records, ok := d.records.(RecordData); ok {
		if _, ok := records.queryer.(jobstore.Beginner); !ok {
			return dygo.NotificationReceipt{}, fmt.Errorf("notification transaction is unavailable")
		}
		var receipt dygo.NotificationReceipt
		err := records.Transaction(ctx, func(txCtx context.Context, txRecords dygo.RecordData) error {
			transactional, ok := txRecords.(RecordData)
			if !ok {
				return fmt.Errorf("notification transaction is unavailable")
			}
			beginner, ok := transactional.queryer.(jobstore.Beginner)
			if !ok {
				return fmt.Errorf("notification transaction job store is unavailable")
			}
			jobs, err := NewJobDataFromBeginner(beginner)
			if err != nil {
				return err
			}
			receipt, err = NotificationData{records: txRecords, jobs: jobs}.send(txCtx, message)
			return err
		})
		return receipt, err
	}
	return d.send(ctx, message)
}

func (d NotificationData) send(ctx context.Context, message dygo.NotificationMessage) (dygo.NotificationReceipt, error) {
	message.Recipient = strings.TrimSpace(message.Recipient)
	message.Title = strings.TrimSpace(message.Title)
	message.Message = strings.TrimSpace(message.Message)
	message.DeepLink = strings.TrimSpace(message.DeepLink)
	message.IdempotencyKey = strings.TrimSpace(message.IdempotencyKey)
	if message.Recipient == "" || message.Title == "" || message.Message == "" || message.IdempotencyKey == "" {
		return dygo.NotificationReceipt{}, fmt.Errorf("notification recipient, title, message, and idempotency key are required")
	}
	if message.DeepLink == "" {
		message.DeepLink = "/"
	}
	message.DeepLink = notificationstore.SafeDeepLink(message.DeepLink)
	locked, err := d.records.Lock(ctx, "core", "user", dygo.RecordListParams{
		Limit:   1,
		Filters: []dygo.RecordFilter{{Field: "name", Operator: "eq", Value: message.Recipient}},
	})
	if err != nil {
		return dygo.NotificationReceipt{}, err
	}
	if len(locked.Records) != 1 {
		return dygo.NotificationReceipt{}, fmt.Errorf("notification recipient was not found")
	}

	match := notificationMatch(message)
	if existing, err := d.records.Find(ctx, "core", "notification", match); err == nil {
		return notificationReceipt(existing, false), nil
	} else if !isRecordNotFound(err) {
		return dygo.NotificationReceipt{}, err
	}

	record, err := d.records.Create(ctx, "core", "notification", notificationInput(message))
	if err != nil {
		// A concurrent sender can win the unique recipient/key constraint.
		if existing, findErr := d.records.Find(ctx, "core", "notification", match); findErr == nil {
			return notificationReceipt(existing, false), nil
		} else if !isRecordNotFound(findErr) {
			return dygo.NotificationReceipt{}, findErr
		}
		return dygo.NotificationReceipt{}, err
	}
	receipt := notificationReceipt(record, true)
	if !message.Email {
		return receipt, nil
	}
	if d.jobs == nil {
		return dygo.NotificationReceipt{}, fmt.Errorf("notification email queue is unavailable")
	}
	payload, err := json.Marshal(map[string]any{"notification-id": receipt.ID})
	if err != nil {
		return dygo.NotificationReceipt{}, err
	}
	_, err = d.jobs.Enqueue(ctx, "core", notificationEmailJob, payload, dygo.EnqueueOptions{
		IdempotencyKey: "notification-email:" + receipt.Name,
	})
	if err != nil {
		return dygo.NotificationReceipt{}, err
	}
	return receipt, nil
}

func notificationMatch(message dygo.NotificationMessage) dygo.RecordInput {
	return recordInput(map[string]any{
		"recipient":       message.Recipient,
		"idempotency-key": message.IdempotencyKey,
	})
}

func notificationInput(message dygo.NotificationMessage) dygo.RecordInput {
	return recordInput(map[string]any{
		"recipient":       message.Recipient,
		"title":           message.Title,
		"message":         message.Message,
		"deep-link":       message.DeepLink,
		"send-email":      message.Email,
		"idempotency-key": message.IdempotencyKey,
	})
}

func notificationReceipt(record dygo.Record, created bool) dygo.NotificationReceipt {
	receipt := dygo.NotificationReceipt{Created: created}
	switch value := record["id"].(type) {
	case int64:
		receipt.ID = value
	case int:
		receipt.ID = int64(value)
	case float64:
		receipt.ID = int64(value)
	}
	receipt.Name, _ = record["name"].(string)
	return receipt
}

func isRecordNotFound(err error) bool {
	var recordErr db.RecordError
	return errors.As(err, &recordErr) && recordErr.Code == db.RecordErrorNotFound
}

func recordInput(values map[string]any) dygo.RecordInput {
	result := make(dygo.RecordInput, len(values))
	for field, value := range values {
		encoded, _ := json.Marshal(value)
		result[field] = encoded
	}
	return result
}
