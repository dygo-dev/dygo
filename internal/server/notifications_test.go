package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hapyco/dygo/internal/notifications"
)

func TestNotificationRoutesListUnreadForCurrentUser(t *testing.T) {
	store := &fakeNotificationStore{items: []notifications.Notification{{
		ID: 9, Name: "notice-9", Title: "Leave approved", Message: "Your leave was approved.", DeepLink: "/hr-leave-request/HRL-9",
	}}}
	request := authenticatedRequest(http.MethodGet, "/api/v1/notifications?limit=10", "")
	recorder := httptest.NewRecorder()

	NewRouter(Options{Auth: validFakeAuthStore(), Notifications: store}).ServeHTTP(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.StatusCode, body)
	}
	if store.userID != 7 || store.limit != 10 {
		t.Fatalf("store call user = %d limit = %d", store.userID, store.limit)
	}
	if !strings.Contains(string(body), `"title":"Leave approved"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestNotificationRoutesMarkReadRejectsAnotherUsersNotification(t *testing.T) {
	store := &fakeNotificationStore{markErr: notifications.ErrNotFound}
	request := authenticatedRequest(http.MethodPost, "/api/v1/notifications/22/read", "")
	recorder := httptest.NewRecorder()

	NewRouter(Options{Auth: validFakeAuthStore(), Notifications: store}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

type fakeNotificationStore struct {
	items   []notifications.Notification
	userID  int64
	limit   int
	markErr error
}

func (f *fakeNotificationStore) ListUnread(_ context.Context, userID int64, limit int) ([]notifications.Notification, error) {
	f.userID, f.limit = userID, limit
	return f.items, nil
}

func (f *fakeNotificationStore) UnreadCount(context.Context, int64) (int64, error) {
	return int64(len(f.items)), nil
}

func (f *fakeNotificationStore) MarkRead(context.Context, int64, int64, time.Time) (notifications.Notification, error) {
	return notifications.Notification{}, f.markErr
}

func (f *fakeNotificationStore) DeepLink(context.Context, int64, int64) (string, error) {
	return "/", nil
}
