package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hapyco/dygo/internal/notifications"
)

const defaultNotificationLimit = 20
const maximumNotificationLimit = 50

type notificationHandler struct {
	store NotificationStore
	now   func() time.Time
}

func registerNotificationRoutes(router chi.Router, store NotificationStore) {
	handler := notificationHandler{store: store, now: time.Now}
	router.Get("/notifications", handler.listUnread)
	router.Get("/notifications/unread-count", handler.unreadCount)
	router.Post("/notifications/{notificationID}/read", handler.markRead)
	router.Get("/notifications/{notificationID}/deep-link", handler.deepLink)
}

func (h notificationHandler) listUnread(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeErrorEnvelope(w, http.StatusUnauthorized, "unauthenticated", "authentication required", nil)
		return
	}
	limit, err := notificationLimit(r)
	if err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	if h.store == nil {
		writeErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable", "notification service is unavailable", nil)
		return
	}
	items, err := h.store.ListUnread(r.Context(), user.ID, limit)
	if err != nil {
		writeErrorEnvelope(w, http.StatusInternalServerError, "internal_error", "notifications could not be loaded", nil)
		return
	}
	writeJSON(w, http.StatusOK, listEnvelope{Data: items, Meta: recordListMeta{Limit: limit, Count: len(items)}})
}

func (h notificationHandler) unreadCount(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeErrorEnvelope(w, http.StatusUnauthorized, "unauthenticated", "authentication required", nil)
		return
	}
	if h.store == nil {
		writeErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable", "notification service is unavailable", nil)
		return
	}
	count, err := h.store.UnreadCount(r.Context(), user.ID)
	if err != nil {
		writeErrorEnvelope(w, http.StatusInternalServerError, "internal_error", "notification count could not be loaded", nil)
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: map[string]int64{"count": count}})
}

func (h notificationHandler) markRead(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeErrorEnvelope(w, http.StatusUnauthorized, "unauthenticated", "authentication required", nil)
		return
	}
	id, err := notificationID(r)
	if err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	if h.store == nil {
		writeErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable", "notification service is unavailable", nil)
		return
	}
	item, err := h.store.MarkRead(r.Context(), user.ID, id, h.now())
	if errors.Is(err, notifications.ErrNotFound) {
		writeErrorEnvelope(w, http.StatusNotFound, "not_found", "notification not found", nil)
		return
	}
	if err != nil {
		writeErrorEnvelope(w, http.StatusInternalServerError, "internal_error", "notification could not be marked as read", nil)
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: item})
}

func (h notificationHandler) deepLink(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeErrorEnvelope(w, http.StatusUnauthorized, "unauthenticated", "authentication required", nil)
		return
	}
	id, err := notificationID(r)
	if err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	if h.store == nil {
		writeErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable", "notification service is unavailable", nil)
		return
	}
	link, err := h.store.DeepLink(r.Context(), user.ID, id)
	if errors.Is(err, notifications.ErrNotFound) {
		writeErrorEnvelope(w, http.StatusNotFound, "not_found", "notification not found", nil)
		return
	}
	if err != nil {
		writeErrorEnvelope(w, http.StatusInternalServerError, "internal_error", "notification link could not be loaded", nil)
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: map[string]string{"deep-link": link}})
}

func notificationLimit(r *http.Request) (int, error) {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return defaultNotificationLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maximumNotificationLimit {
		return 0, errors.New("limit must be an integer from 1 to 50")
	}
	return limit, nil
}

func notificationID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "notificationID"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("notification ID must be a positive integer")
	}
	return id, nil
}
