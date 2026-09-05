// Package files implements the framework-owned private file service.
package files

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hapyco/dygo/internal/corevalues"
	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/dygodata"
	jobstore "github.com/hapyco/dygo/internal/jobs/store"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/pkg/dygo"
)

const (
	coreApp       = "core"
	coreEntity    = "file"
	cleanupJob    = "delete-file-blob"
	maxFileSize   = 100 << 20
	storageKeyLen = 24
)

// BlobStore is the deliberately small storage seam used by the file service.
type BlobStore interface {
	Put(context.Context, string, io.Reader, int64) error
	Open(context.Context, string) (io.ReadCloser, error)
	Remove(context.Context, string) error
}

type blobPathProvider interface {
	Path(string) string
}

type jobEnqueuer interface {
	Enqueue(context.Context, string, string, json.RawMessage, dygo.EnqueueOptions) (dygo.JobExecution, error)
}

type scopeChecker interface {
	RecordScope(context.Context, permissions.Request) (permissions.Scope, error)
}

// Service is the public FileData implementation.
type Service struct {
	queryer      db.RecordQueryer
	store        db.RecordStore
	blobs        BlobStore
	jobs         jobEnqueuer
	checker      scopeChecker
	actor        *dygo.Actor
	rollbackKeys *[]string
	bindingErr   error
}

var _ dygo.FileData = Service{}

// NewService returns a private file service backed by metadata-driven Core records.
func NewService(queryer db.RecordQueryer, blobs BlobStore, jobs jobEnqueuer, checker scopeChecker) Service {
	return Service{queryer: queryer, store: db.NewRecordStoreWithHookPolicy(queryer, db.RecordMutationHooksNone), blobs: blobs, jobs: jobs, checker: checker}
}

// WithQueryer returns a file view backed by a caller-owned transaction.
func (s Service) WithQueryer(queryer db.RecordQueryer) dygo.FileData {
	bound, err := s.bindQueryer(queryer)
	if err != nil {
		bound.bindingErr = err
	}
	keys := []string{}
	bound.rollbackKeys = &keys
	return bound
}

func (s Service) bindQueryer(queryer db.RecordQueryer) (Service, error) {
	s.queryer = queryer
	s.store = db.NewRecordStoreWithHookPolicy(queryer, db.RecordMutationHooksNone)
	beginner, ok := queryer.(jobstore.Beginner)
	if !ok {
		return s, fmt.Errorf("file transaction does not support job enqueueing")
	}
	jobs, err := dygodata.NewJobDataFromBeginner(beginner)
	if err != nil {
		return s, err
	}
	s.jobs = jobs
	return s, nil
}

// AsActor returns a view that applies the actor's conditional Record scopes.
func (s Service) AsActor(actor dygo.Actor) dygo.FileData {
	s.actor = &actor
	return s
}

// Upload stores one private file and optionally attaches it to a Record.
func (s Service) Upload(ctx context.Context, upload dygo.FileUpload) (dygo.File, error) {
	ctx = s.actorContext(ctx)
	if err := s.require(); err != nil {
		return dygo.File{}, err
	}
	if err := validateUpload(upload); err != nil {
		return dygo.File{}, err
	}
	if err := validateTarget(upload.Target); err != nil {
		return dygo.File{}, err
	}
	if hasTarget(upload.Target) {
		if err := s.authorizeTarget(ctx, upload.Target, permissions.ActionUpdate, true); err != nil {
			return dygo.File{}, err
		}
	}
	key, err := randomKey()
	if err != nil {
		return dygo.File{}, fmt.Errorf("create file storage key: %w", err)
	}
	hasher := sha256.New()
	body := io.TeeReader(io.LimitReader(upload.Body, maxFileSize+1), hasher)
	if err := s.blobs.Put(ctx, key, body, upload.Size); err != nil {
		return dygo.File{}, fmt.Errorf("store file: %w", err)
	}
	if s.rollbackKeys != nil {
		*s.rollbackKeys = append(*s.rollbackKeys, key)
	}
	mimeType := strings.TrimSpace(upload.ContentType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	input := db.RecordInput{
		"filename":     jsonString(upload.Filename),
		"storage-key":  jsonString(key),
		"checksum":     jsonString(hex.EncodeToString(hasher.Sum(nil))),
		"content-type": jsonString(mimeType),
		"size":         jsonInt(upload.Size),
		"private":      jsonBool(true),
		"retired":      jsonBool(false),
	}
	if s.actor != nil {
		input["actor"] = jsonString(s.actor.Email)
	}
	if hasTarget(upload.Target) {
		addTarget(input, upload.Target)
	}
	record, err := s.store.SystemWriter().InsertReturningByIdentity(ctx, coreApp, coreEntity, input, db.SystemMutationSilent)
	if err != nil {
		_ = s.blobs.Remove(context.Background(), key)
		return dygo.File{}, err
	}
	file, err := fileFromRecord(record)
	if err != nil {
		_ = s.store.SystemWriter().DeleteByIdentity(ctx, coreApp, coreEntity, numberValue(record["id"]), db.SystemMutationSilent)
		_ = s.blobs.Remove(context.Background(), key)
		return dygo.File{}, err
	}
	if hasTarget(file.Target) {
		err = db.NewActivityReader(s.queryer).AddEventByIdentity(ctx, file.Target.App, file.Target.Entity, file.Target.RecordID, db.TimelineEvent{
			Kind: corevalues.ActivityKindAttachment, Operation: corevalues.ActivityOperationAttachmentAdded,
			Status: corevalues.ActivityStatusSuccess, Title: "Attached " + file.Filename,
			Details: map[string]any{"file-id": file.ID, "field": file.Target.Field},
		})
		if err != nil {
			_ = s.store.SystemWriter().DeleteByIdentity(ctx, coreApp, coreEntity, file.ID, db.SystemMutationSilent)
			_ = s.blobs.Remove(context.Background(), key)
			return dygo.File{}, err
		}
	}
	return file, nil
}

func (s Service) actorContext(ctx context.Context) context.Context {
	if s.actor == nil {
		return ctx
	}
	return db.WithActivityActor(ctx, s.actor.UserID, s.actor.Email, s.actor.Administrator)
}

// Rollback removes blobs written through a transaction-scoped view.
func (s Service) Rollback(ctx context.Context) {
	if s.rollbackKeys == nil {
		return
	}
	for _, key := range *s.rollbackKeys {
		_ = s.blobs.Remove(ctx, key)
	}
	*s.rollbackKeys = nil
}

// Attach attaches an existing unassigned file to a permitted Record field.
func (s Service) Attach(ctx context.Context, id int64, target dygo.FileTarget) (dygo.File, error) {
	if err := s.require(); err != nil {
		return dygo.File{}, err
	}
	if id <= 0 || !targetProvided(target) {
		return dygo.File{}, fileError("invalid_request", "file and target are required")
	}
	if err := validateTarget(target); err != nil {
		return dygo.File{}, err
	}
	if err := s.authorizeTarget(ctx, target, permissions.ActionUpdate, true); err != nil {
		return dygo.File{}, err
	}
	file, err := s.get(ctx, id)
	if err != nil {
		return dygo.File{}, err
	}
	if hasTarget(file.Target) {
		return dygo.File{}, fileError("conflict", "file is already attached")
	}
	record, err := s.store.SystemWriter().UpdateByIdentity(ctx, coreApp, coreEntity, id, targetInput(target), db.SystemMutationSilent)
	if err != nil {
		return dygo.File{}, err
	}
	return fileFromRecord(record)
}

// Open opens a permitted private file and returns its persisted descriptor.
func (s Service) Open(ctx context.Context, id int64) (dygo.File, io.ReadCloser, error) {
	if err := s.require(); err != nil {
		return dygo.File{}, nil, err
	}
	file, err := s.get(ctx, id)
	if err != nil {
		return dygo.File{}, nil, err
	}
	if !hasTarget(file.Target) {
		return dygo.File{}, nil, fileError("not_found", "file is not attached")
	}
	if err := s.authorizeTarget(ctx, file.Target, permissions.ActionRead, false); err != nil {
		return dygo.File{}, nil, err
	}
	reader, err := s.blobs.Open(ctx, file.StorageKey)
	if err != nil {
		return dygo.File{}, nil, fmt.Errorf("open file: %w", err)
	}
	return file, reader, nil
}

// Remove removes the Core File record, then queues durable blob cleanup.
func (s Service) Remove(ctx context.Context, id int64) error {
	if err := s.require(); err != nil {
		return err
	}
	if s.jobs == nil {
		return fmt.Errorf("file cleanup job service is unavailable")
	}
	beginner, ok := s.queryer.(jobstore.Beginner)
	if !ok {
		return fmt.Errorf("file removal transaction is unavailable")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin file removal: %w", err)
	}
	defer tx.Rollback(ctx)
	transactional, err := s.bindQueryer(tx)
	if err != nil {
		return err
	}
	if err := transactional.remove(ctx, id); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit file removal: %w", err)
	}
	return nil
}

func (s Service) remove(ctx context.Context, id int64) error {
	file, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if !hasTarget(file.Target) {
		return fileError("not_found", "file is not attached")
	}
	if err := s.authorizeTarget(ctx, file.Target, permissions.ActionUpdate, true); err != nil {
		return err
	}
	if err := s.store.SystemWriter().DeleteByIdentity(ctx, coreApp, coreEntity, id, db.SystemMutationSilent); err != nil {
		return err
	}
	payload := cleanupPayload{StorageKey: file.StorageKey}
	if paths, ok := s.blobs.(blobPathProvider); ok {
		payload.Path = paths.Path(file.StorageKey)
	}
	body, _ := json.Marshal(payload)
	_, err = s.jobs.Enqueue(ctx, coreApp, cleanupJob, body, dygo.EnqueueOptions{IdempotencyKey: "file-cleanup:" + fmt.Sprint(id)})
	return err
}

func (s Service) require() error {
	if s.bindingErr != nil {
		return s.bindingErr
	}
	if s.queryer == nil || s.blobs == nil {
		return fmt.Errorf("file service is unavailable")
	}
	return nil
}

func (s Service) get(ctx context.Context, id int64) (dygo.File, error) {
	if id <= 0 {
		return dygo.File{}, fileError("invalid_request", "file id must be positive")
	}
	record, err := s.store.GetRecordByIdentity(ctx, coreApp, coreEntity, id)
	if err != nil {
		return dygo.File{}, err
	}
	return fileFromRecord(record)
}

func (s Service) authorizeTarget(ctx context.Context, target dygo.FileTarget, action permissions.Action, write bool) error {
	if !hasTarget(target) {
		return fileError("invalid_request", "file target is required")
	}
	meta, err := db.NewMetadataReader(s.queryer).GetEntityMetaByIdentity(ctx, target.App, target.Entity)
	if err != nil {
		return err
	}
	found := false
	for _, field := range append(meta.SystemFields, meta.Fields...) {
		if field.Name == target.Field {
			found = true
			break
		}
	}
	if !found {
		return fileError("invalid_request", "file target field was not found")
	}
	if s.actor == nil || s.actor.Administrator {
		if _, err := s.store.GetRecordByIdentity(ctx, target.App, target.Entity, target.RecordID); err != nil {
			return err
		}
		return nil
	}
	if s.checker == nil {
		return fileError("internal_error", "permission scope is unavailable")
	}
	scope, err := s.checker.RecordScope(ctx, permissions.Request{Actor: permissions.Actor(*s.actor), Resource: permissions.Resource{Kind: permissions.ResourceEntity, App: target.App, Name: target.Entity}, Action: action})
	if err != nil {
		return err
	}
	scoped := s.store.WithScope(db.RecordScope{Where: scope.Where, Args: scope.Args, FieldRead: scope.FieldRead, FieldWrite: scope.FieldWrite})
	if _, err := scoped.GetRecordByIdentity(ctx, target.App, target.Entity, target.RecordID); err != nil {
		return err
	}
	return scoped.AuthorizeField(ctx, target.App, target.Entity, target.RecordID, target.Field, write)
}

func validateUpload(upload dygo.FileUpload) error {
	if strings.TrimSpace(upload.Filename) == "" || upload.Body == nil {
		return fileError("invalid_request", "filename and file body are required")
	}
	if upload.Size < -1 || upload.Size > maxFileSize {
		return fileError("invalid_request", "file size is invalid")
	}
	if !hasTarget(upload.Target) && upload.Size == 0 {
		return nil
	}
	return nil
}

func randomKey() (string, error) {
	buf := make([]byte, storageKeyLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func addTarget(input db.RecordInput, target dygo.FileTarget) {
	for key, value := range targetInput(target) {
		input[key] = value
	}
}

func targetInput(target dygo.FileTarget) db.RecordInput {
	return db.RecordInput{"app": jsonString(target.App), "entity": jsonString(target.Entity), "record-id": jsonInt(target.RecordID), "field": jsonString(target.Field)}
}

func hasTarget(target dygo.FileTarget) bool {
	return strings.TrimSpace(target.App) != "" && strings.TrimSpace(target.Entity) != "" && target.RecordID > 0 && strings.TrimSpace(target.Field) != ""
}

func validateTarget(target dygo.FileTarget) error {
	if !targetProvided(target) || hasTarget(target) {
		return nil
	}
	return fileError("invalid_request", "file target must include app, entity, record id, and field")
}

func targetProvided(target dygo.FileTarget) bool {
	return strings.TrimSpace(target.App) != "" || strings.TrimSpace(target.Entity) != "" || target.RecordID != 0 || strings.TrimSpace(target.Field) != ""
}

func fileFromRecord(record db.Record) (dygo.File, error) {
	id, ok := integer(record["id"])
	if !ok {
		return dygo.File{}, errors.New("file record id is invalid")
	}
	size, _ := integer(record["size"])
	private, _ := record["private"].(bool)
	retired, _ := record["retired"].(bool)
	file := dygo.File{ID: id, Name: stringValue(record["name"]), Filename: stringValue(record["filename"]), StorageKey: stringValue(record["storage-key"]), Checksum: stringValue(record["checksum"]), ContentType: stringValue(record["content-type"]), Size: size, Private: private, Actor: stringValue(record["actor"]), Retired: retired}
	file.Target = dygo.FileTarget{App: stringValue(record["app"]), Entity: stringValue(record["entity"]), RecordID: numberValue(record["record-id"]), Field: stringValue(record["field"])}
	return file, nil
}

type cleanupPayload struct {
	StorageKey string `json:"storage-key"`
	Path       string `json:"path,omitempty"`
}

func fileError(code string, message string) error {
	return dygo.ActionError{Code: code, Message: message}
}
func jsonString(value string) json.RawMessage { b, _ := json.Marshal(value); return b }
func jsonInt(value int64) json.RawMessage     { b, _ := json.Marshal(value); return b }
func jsonBool(value bool) json.RawMessage     { b, _ := json.Marshal(value); return b }

func stringValue(value any) string {
	s, _ := value.(string)
	return s
}

func integer(value any) (int64, bool) {
	switch number := value.(type) {
	case int64:
		return number, true
	case int:
		return int64(number), true
	case int32:
		return int64(number), true
	case float64:
		return int64(number), true
	default:
		return 0, false
	}
}

func numberValue(value any) int64 { number, _ := integer(value); return number }
