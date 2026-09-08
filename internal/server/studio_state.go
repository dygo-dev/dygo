package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/internal/studiostate"
	"github.com/hapyco/dygo/pkg/dygo"
)

type StudioStateStore interface {
	Preferences(context.Context, dygo.Actor) (map[string]any, error)
	PutPreference(context.Context, dygo.Actor, string, json.RawMessage) error
	DeletePreference(context.Context, dygo.Actor, string) error
	SavedFilters(context.Context, dygo.Actor, string) ([]studiostate.SavedFilter, error)
	CreateSavedFilter(context.Context, dygo.Actor, string, string, []studiostate.Filter) (studiostate.SavedFilter, error)
	UpdateSavedFilter(context.Context, dygo.Actor, int64, *string, *[]studiostate.Filter) (studiostate.SavedFilter, error)
	DeleteSavedFilter(context.Context, dygo.Actor, int64) error
}

func registerStudioStateRoutes(router chi.Router, store StudioStateStore) {
	router.Route("/studio", func(router chi.Router) {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if store == nil {
					writeErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable", "Studio state is unavailable", nil)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		h := studioStateHandler{store}
		router.Get("/preferences", h.preferences)
		router.Put("/preferences/{key}", h.preference)
		router.Delete("/preferences/{key}", h.preference)
		router.Get("/saved-filters", h.savedFilters)
		router.Post("/saved-filters", h.savedFilters)
		router.Patch("/saved-filters/{id}", h.savedFilter)
		router.Delete("/saved-filters/{id}", h.savedFilter)
	})
}

type studioStateHandler struct{ store StudioStateStore }

func studioActor(r *http.Request) dygo.Actor {
	user, _ := CurrentUserFromContext(r.Context())
	return dygo.Actor{UserID: user.ID, Email: user.Email, Administrator: user.Administrator}
}

func decodeStudioInput(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = errors.New("extra JSON")
		}
	}
	if err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, "invalid_request", "invalid request body", nil)
		return false
	}
	return true
}

func writeStudioState(w http.ResponseWriter, status int, data any, err error) {
	if err != nil {
		var permissionErr permissions.Error
		if errors.As(err, &permissionErr) {
			writePermissionError(w, err)
		} else if db.IsMetadataNotFound(err) {
			writeErrorEnvelope(w, http.StatusNotFound, "not_found", "Entity not found", nil)
		} else {
			writeRecordError(w, err)
		}
		return
	}
	writeJSON(w, status, dataEnvelope{Data: data})
}

func (h studioStateHandler) preferences(w http.ResponseWriter, r *http.Request) {
	data, err := h.store.Preferences(r.Context(), studioActor(r))
	writeStudioState(w, http.StatusOK, data, err)
}

func (h studioStateHandler) preference(w http.ResponseWriter, r *http.Request) {
	key, err := url.PathUnescape(chi.URLParam(r, "key"))
	if err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, "invalid_request", "invalid preference key", nil)
		return
	}
	if r.Method == http.MethodDelete {
		err = h.store.DeletePreference(r.Context(), studioActor(r), key)
	} else {
		var body struct {
			Value json.RawMessage `json:"value"`
		}
		if !decodeStudioInput(w, r, &body) {
			return
		}
		err = h.store.PutPreference(r.Context(), studioActor(r), key, body.Value)
	}
	writeStudioState(w, http.StatusOK, map[string]bool{"saved": true}, err)
}

func (h studioStateHandler) savedFilters(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data, err := h.store.SavedFilters(r.Context(), studioActor(r), r.URL.Query().Get("entity"))
		writeStudioState(w, http.StatusOK, data, err)
		return
	}
	var body struct {
		Entity  string               `json:"entity"`
		Label   string               `json:"label"`
		Filters []studiostate.Filter `json:"filters"`
	}
	if !decodeStudioInput(w, r, &body) {
		return
	}
	data, err := h.store.CreateSavedFilter(r.Context(), studioActor(r), body.Entity, body.Label, body.Filters)
	writeStudioState(w, http.StatusCreated, data, err)
}

func (h studioStateHandler) savedFilter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeErrorEnvelope(w, http.StatusBadRequest, "invalid_request", "invalid saved filter ID", nil)
		return
	}
	if r.Method == http.MethodDelete {
		err = h.store.DeleteSavedFilter(r.Context(), studioActor(r), id)
		writeStudioState(w, http.StatusOK, map[string]bool{"deleted": true}, err)
		return
	}
	var body struct {
		Label   *string               `json:"label"`
		Filters *[]studiostate.Filter `json:"filters"`
	}
	if !decodeStudioInput(w, r, &body) {
		return
	}
	data, err := h.store.UpdateSavedFilter(r.Context(), studioActor(r), id, body.Label, body.Filters)
	writeStudioState(w, http.StatusOK, data, err)
}
