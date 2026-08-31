// Package notifications implements the framework notification inbox and email delivery.
package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound means the notification does not belong to the current user.
var ErrNotFound = errors.New("notification not found")

// Queryer is the PostgreSQL behavior used by the notification inbox.
type Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Store reads and updates notification inbox records.
type Store struct {
	queryer Queryer
}

// Notification is one item in a user's inbox.
type Notification struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	DeepLink  string  `json:"deep-link"`
	CreatedAt string  `json:"created-at"`
	ReadAt    *string `json:"read-at"`
}

// NewStore creates a notification Store.
func NewStore(queryer Queryer) Store {
	return Store{queryer: queryer}
}

// ListUnread returns newest unread notifications for one user.
func (s Store) ListUnread(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	if s.queryer == nil {
		return nil, fmt.Errorf("notification store is unavailable")
	}
	rows, err := s.queryer.Query(ctx, `
SELECT id, name, title, message, deep_link, created_at
FROM notification
WHERE recipient_id = $1 AND read_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Notification{}
	for rows.Next() {
		var item Notification
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Name, &item.Title, &item.Message, &item.DeepLink, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.DeepLink = SafeDeepLink(item.DeepLink)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UnreadCount returns the number of unread notifications for one user.
func (s Store) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	if s.queryer == nil {
		return 0, fmt.Errorf("notification store is unavailable")
	}
	var count int64
	err := s.queryer.QueryRow(ctx, `SELECT COUNT(*) FROM notification WHERE recipient_id = $1 AND read_at IS NULL`, userID).Scan(&count)
	return count, err
}

// MarkRead marks one owned notification as read and returns it.
func (s Store) MarkRead(ctx context.Context, userID int64, notificationID int64, now time.Time) (Notification, error) {
	if s.queryer == nil {
		return Notification{}, fmt.Errorf("notification store is unavailable")
	}
	var item Notification
	var createdAt time.Time
	var readAt time.Time
	err := s.queryer.QueryRow(ctx, `
UPDATE notification
SET read_at = COALESCE(read_at, $3), updated_at = CASE WHEN read_at IS NULL THEN $3 ELSE updated_at END
WHERE id = $1 AND recipient_id = $2
RETURNING id, name, title, message, deep_link, created_at, read_at`, notificationID, userID, now.UTC()).Scan(
		&item.ID, &item.Name, &item.Title, &item.Message, &item.DeepLink, &createdAt, &readAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	if err != nil {
		return Notification{}, err
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	read := readAt.UTC().Format(time.RFC3339)
	item.ReadAt = &read
	item.DeepLink = SafeDeepLink(item.DeepLink)
	return item, nil
}

// DeepLink returns the safe local route for one owned notification.
func (s Store) DeepLink(ctx context.Context, userID int64, notificationID int64) (string, error) {
	if s.queryer == nil {
		return "", fmt.Errorf("notification store is unavailable")
	}
	var deepLink string
	err := s.queryer.QueryRow(ctx, `SELECT deep_link FROM notification WHERE id = $1 AND recipient_id = $2`, notificationID, userID).Scan(&deepLink)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return SafeDeepLink(deepLink), nil
}

// SafeDeepLink keeps notification navigation on the current Studio origin.
func SafeDeepLink(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return value
}
