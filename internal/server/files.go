package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hapyco/dygo/internal/auth"
	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/pkg/dygo"
)

type fileHandler struct{ files dygo.FileData }

func registerFileRoutes(router chi.Router, files dygo.FileData) {
	handler := fileHandler{files: files}
	router.Post("/files", handler.upload)
	router.Get("/files/{id}", handler.open)
	router.Delete("/files/{id}", handler.remove)
}

func (h fileHandler) upload(w http.ResponseWriter, r *http.Request) {
	if h.require(w) {
		return
	}
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeAuthError(w, auth.Error{Code: auth.ErrorUnauthenticated, Message: "authentication required"})
		return
	}
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeFileError(w, dygo.ActionError{Code: "invalid_request", Message: "multipart file body is required"})
		return
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		writeFileError(w, dygo.ActionError{Code: "invalid_request", Message: "multipart file field is required"})
		return
	}
	defer part.Close()
	recordID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("record-id")), 10, 64)
	if err != nil {
		recordID = 0
	}
	file, err := h.files.AsActor(dygo.Actor{UserID: user.ID, Email: user.Email, Administrator: user.Administrator}).Upload(r.Context(), dygo.FileUpload{
		Filename: header.Filename, ContentType: header.Header.Get("Content-Type"), Size: header.Size, Body: part,
		Target: dygo.FileTarget{App: r.FormValue("app"), Entity: r.FormValue("entity"), RecordID: recordID, Field: r.FormValue("field")},
	})
	if err != nil {
		writeFileError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dataEnvelope{Data: file})
}

func (h fileHandler) open(w http.ResponseWriter, r *http.Request) {
	if h.require(w) {
		return
	}
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeAuthError(w, auth.Error{Code: auth.ErrorUnauthenticated, Message: "authentication required"})
		return
	}
	id, err := fileID(r)
	if err != nil {
		writeFileError(w, err)
		return
	}
	file, reader, err := h.files.AsActor(dygo.Actor{UserID: user.ID, Email: user.Email, Administrator: user.Administrator}).Open(r.Context(), id)
	if err != nil {
		writeFileError(w, err)
		return
	}
	defer reader.Close()
	contentType := file.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Filename))
	if file.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	}
	if _, err := io.Copy(w, reader); err != nil {
		return
	}
}

func (h fileHandler) remove(w http.ResponseWriter, r *http.Request) {
	if h.require(w) {
		return
	}
	user, ok := CurrentUserFromContext(r.Context())
	if !ok {
		writeAuthError(w, auth.Error{Code: auth.ErrorUnauthenticated, Message: "authentication required"})
		return
	}
	id, err := fileID(r)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if err := h.files.AsActor(dygo.Actor{UserID: user.ID, Email: user.Email, Administrator: user.Administrator}).Remove(r.Context(), id); err != nil {
		writeFileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h fileHandler) require(w http.ResponseWriter) bool {
	if h.files != nil {
		return false
	}
	writeJSON(w, http.StatusServiceUnavailable, errorEnvelope{Error: apiError{Code: "service_unavailable", Message: "file service is unavailable"}})
	return true
}

func fileID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, dygo.ActionError{Code: "invalid_request", Message: "file id must be a positive integer"}
	}
	return id, nil
}

func writeFileError(w http.ResponseWriter, err error) {
	var actionErr dygo.ActionError
	if errors.As(err, &actionErr) {
		status := http.StatusInternalServerError
		switch actionErr.Code {
		case "invalid_request":
			status = http.StatusBadRequest
		case "not_found":
			status = http.StatusNotFound
		case "permission_denied":
			status = http.StatusForbidden
		case "conflict":
			status = http.StatusConflict
		}
		writeErrorEnvelope(w, status, actionErr.Code, actionErr.Message, actionErr.Details)
		return
	}
	var permissionErr permissions.Error
	if errors.As(err, &permissionErr) {
		writePermissionError(w, err)
		return
	}
	var recordErr db.RecordError
	if errors.As(err, &recordErr) {
		writeRecordError(w, err)
		return
	}
	writeErrorEnvelope(w, http.StatusInternalServerError, "internal_error", "file operation failed", nil)
}
