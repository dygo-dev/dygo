package dygo

import "context"

// NotificationMessage describes one in-app notification and optional email copy.
type NotificationMessage struct {
	Recipient      string
	Title          string
	Message        string
	DeepLink       string
	Email          bool
	IdempotencyKey string
}

// NotificationReceipt identifies one persisted notification.
type NotificationReceipt struct {
	ID      int64
	Name    string
	Created bool
}

// NotificationData sends durable user notifications from trusted App code.
type NotificationData interface {
	Send(context.Context, NotificationMessage) (NotificationReceipt, error)
}
