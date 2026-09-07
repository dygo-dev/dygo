package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hapyco/dygo/internal/corevalues"
	"github.com/jackc/pgx/v5"
)

// ActivityReader reads scoped Record Activity from Core activity records.
type ActivityReader struct {
	queryer ActivityQueryer
}

// ActivityQueryer is the database behavior needed by the Activity reader.
type ActivityQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ActivityActor is the optional user that caused an Activity entry.
type ActivityActor struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full-name"`
}

// ActivityEntry is one append-only Record history entry.
type ActivityEntry struct {
	ID        int64          `json:"id"`
	CreatedAt string         `json:"created-at"`
	Entity    string         `json:"entity"`
	RecordID  int64          `json:"record-id"`
	Kind      string         `json:"kind"`
	Operation string         `json:"operation"`
	Status    string         `json:"status"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Actor     *ActivityActor `json:"actor"`
	Changes   any            `json:"changes"`
	Snapshot  any            `json:"snapshot"`
	Details   any            `json:"details"`
}

// TimelineEvent is the storage-facing shape used for an Activity append.
type TimelineEvent struct {
	Kind      string
	Operation string
	Status    string
	Title     string
	Message   string
	Changes   any
	Snapshot  any
	Details   any
}

// ActivityListResult is a page of Activity entries.
type ActivityListResult struct {
	Activities []ActivityEntry
	Limit      int
	Offset     int
	Count      int
}

// NewActivityReader returns an Activity reader backed by queryer.
func NewActivityReader(queryer ActivityQueryer) ActivityReader {
	return ActivityReader{queryer: queryer}
}

// AddComment appends a comment Activity entry for an existing Record.
func (r ActivityReader) AddComment(ctx context.Context, entity string, recordID int64, message string) error {
	if strings.TrimSpace(message) == "" {
		return recordError(RecordErrorInvalidRequest, "comment message is required", map[string]any{"entity": entity}, nil)
	}
	return r.AddEvent(ctx, entity, recordID, TimelineEvent{
		Kind:      corevalues.ActivityKindComment,
		Operation: corevalues.ActivityOperationComment,
		Status:    corevalues.ActivityStatusSuccess,
		Title:     "Comment",
		Message:   message,
	})
}

// AddCommentByIdentity appends a comment using an app/entity identity.
func (r ActivityReader) AddCommentByIdentity(ctx context.Context, appName string, entity string, recordID int64, message string) error {
	if strings.TrimSpace(message) == "" {
		return recordError(RecordErrorInvalidRequest, "comment message is required", map[string]any{"entity": entity}, nil)
	}
	return r.AddEventByIdentity(ctx, appName, entity, recordID, TimelineEvent{
		Kind:      corevalues.ActivityKindComment,
		Operation: corevalues.ActivityOperationComment,
		Status:    corevalues.ActivityStatusSuccess,
		Title:     "Comment",
		Message:   message,
	})
}

// AddEvent appends one Activity entry for an existing Record.
func (r ActivityReader) AddEvent(ctx context.Context, entity string, recordID int64, event TimelineEvent) error {
	if err := r.requireQueryer(); err != nil {
		return err
	}
	entityID, entityName, err := r.resolveEntity(ctx, entity)
	if err != nil {
		return err
	}
	return r.addEvent(ctx, entity, entityID, entityName, recordID, event)
}

// AddEventByIdentity appends an Activity event using an app/entity identity.
func (r ActivityReader) AddEventByIdentity(ctx context.Context, appName string, entity string, recordID int64, event TimelineEvent) error {
	if err := r.requireQueryer(); err != nil {
		return err
	}
	entityID, entityName, err := r.resolveEntityByIdentity(ctx, appName, entity)
	if err != nil {
		return err
	}
	return r.addEvent(ctx, appName+"/"+entity, entityID, entityName, recordID, event)
}

func (r ActivityReader) addEvent(ctx context.Context, entity string, entityID int64, entityName string, recordID int64, event TimelineEvent) error {
	if recordID <= 0 {
		return invalidRecordIDError(entity)
	}
	if strings.TrimSpace(event.Kind) == "" || strings.TrimSpace(event.Operation) == "" {
		return recordError(RecordErrorInvalidRequest, "activity kind and operation are required", map[string]any{"entity": entity}, nil)
	}
	queryer, ok := r.queryer.(RecordQueryer)
	if !ok {
		return recordError(RecordErrorInternal, "activity writer requires a record queryer", map[string]any{"entity": entity}, nil)
	}
	input := RecordInput{
		"kind":      systemRecordString(event.Kind),
		"operation": systemRecordString(event.Operation),
		"status":    systemRecordString(defaultActivityStatus(event.Status)),
		"entity":    systemRecordString(entityName),
		"record-id": systemRecordInt(recordID),
		"title":     systemRecordString(defaultActivityTitle(event.Title, event.Kind)),
	}
	if message := strings.TrimSpace(event.Message); message != "" {
		input["message"] = systemRecordString(message)
	}
	if actor, ok := ActivityActorNameFromContext(ctx); ok {
		input["actor"] = systemRecordString(actor)
	}
	for name, value := range map[string]any{"changes": event.Changes, "snapshot": event.Snapshot, "details": event.Details} {
		if value == nil {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return recordError(RecordErrorInvalidRequest, fmt.Sprintf("activity %s is invalid", name), map[string]any{"entity": entity}, err)
		}
		input[name] = raw
	}
	return NewSystemRecordWriter(queryer).InsertByIdentity(ctx, "core", "activity", input, SystemMutationSilent)
}

func defaultActivityStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return corevalues.ActivityStatusSuccess
	}
	return status
}

func defaultActivityTitle(title string, kind string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	if strings.TrimSpace(kind) == corevalues.ActivityKindComment {
		return "Comment"
	}
	return "Timeline event"
}

// ListRecordActivity returns Activity entries for one Entity Record ID.
func (r ActivityReader) ListRecordActivity(ctx context.Context, entity string, recordID int64, params RecordListParams) (ActivityListResult, error) {
	if err := r.requireQueryer(); err != nil {
		return ActivityListResult{}, err
	}
	if recordID <= 0 {
		return ActivityListResult{}, invalidRecordIDError(entity)
	}
	params, err := normalizeRecordListParams(params)
	if err != nil {
		return ActivityListResult{}, err
	}
	entityID, err := r.entityID(ctx, entity)
	if err != nil {
		return ActivityListResult{}, err
	}
	rows, err := r.queryer.Query(ctx, `
SELECT
	a.id,
	a.created_at,
	e.slug,
	a.record_id,
	a.kind,
	a.operation,
	a.status,
	a.title,
	COALESCE(a.message, ''),
	a.changes,
	a.snapshot,
	a.details,
	COALESCE(u.id, 0),
	COALESCE(u.email, ''),
	COALESCE(u.full_name, '')
FROM "activity" a
JOIN "entity" e ON e.id = a.entity_id
LEFT JOIN "user" u ON u.id = a.actor_id
WHERE a.entity_id = $1 AND a.record_id = $2
ORDER BY a.created_at DESC, a.id DESC
LIMIT $3 OFFSET $4`, entityID, recordID, params.Limit, params.Offset)
	if err != nil {
		return ActivityListResult{}, classifyRecordDBError(err, "activity")
	}
	defer rows.Close()

	activities := []ActivityEntry{}
	for rows.Next() {
		var entry ActivityEntry
		var changes []byte
		var snapshot []byte
		var details []byte
		var actorID int64
		var actorEmail string
		var actorFullName string
		var createdAt time.Time
		if err := rows.Scan(
			&entry.ID,
			&createdAt,
			&entry.Entity,
			&entry.RecordID,
			&entry.Kind,
			&entry.Operation,
			&entry.Status,
			&entry.Title,
			&entry.Message,
			&changes,
			&snapshot,
			&details,
			&actorID,
			&actorEmail,
			&actorFullName,
		); err != nil {
			return ActivityListResult{}, recordError(RecordErrorInternal, "scan activity row failed", map[string]any{"entity": entity, "id": recordID}, err)
		}
		entry.CreatedAt = normalizeDatetimeValue(createdAt)
		entry.Changes, err = decodeActivityJSON(changes)
		if err != nil {
			return ActivityListResult{}, err
		}
		entry.Snapshot, err = decodeActivityJSON(snapshot)
		if err != nil {
			return ActivityListResult{}, err
		}
		entry.Details, err = decodeActivityJSON(details)
		if err != nil {
			return ActivityListResult{}, err
		}
		if actorID > 0 {
			entry.Actor = &ActivityActor{ID: actorID, Email: actorEmail, FullName: actorFullName}
		}
		activities = append(activities, entry)
	}
	if err := rows.Err(); err != nil {
		return ActivityListResult{}, classifyRecordDBError(err, "activity")
	}
	return ActivityListResult{Activities: activities, Limit: params.Limit, Offset: params.Offset, Count: len(activities)}, nil
}

func (r ActivityReader) entityID(ctx context.Context, entity string) (int64, error) {
	var id int64
	err := r.queryer.QueryRow(ctx, `SELECT id FROM "entity" WHERE slug = $1`, entity).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, recordError(RecordErrorNotFound, "entity not found", map[string]any{"entity": entity}, err)
	}
	if err != nil {
		return 0, classifyRecordDBError(err, entity)
	}
	return id, nil
}

func (r ActivityReader) resolveEntity(ctx context.Context, entity string) (int64, string, error) {
	entityID, err := r.entityID(ctx, entity)
	if err != nil {
		return 0, "", err
	}
	var entityName string
	if err := r.queryer.QueryRow(ctx, `
SELECT a.name || '.' || e.key
FROM "entity" e
JOIN "app" a ON a.id = e.app_id
WHERE e.id = $1`, entityID).Scan(&entityName); err != nil {
		return 0, "", classifyRecordDBError(err, entity)
	}
	return entityID, entityName, nil
}

func (r ActivityReader) resolveEntityByIdentity(ctx context.Context, appName string, entity string) (int64, string, error) {
	var entityID int64
	var entityName string
	err := r.queryer.QueryRow(ctx, `
SELECT e.id, a.name || '.' || e.key
FROM "entity" e
JOIN "app" a ON a.id = e.app_id
WHERE a.name = $1 AND e.key = $2`, appName, entity).Scan(&entityID, &entityName)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", recordError(RecordErrorNotFound, "entity not found", map[string]any{"app": appName, "entity": entity}, err)
	}
	if err != nil {
		return 0, "", classifyRecordDBError(err, appName+"/"+entity)
	}
	return entityID, entityName, nil
}

func (r ActivityReader) requireQueryer() error {
	if r.queryer == nil {
		return recordError(RecordErrorInternal, "activity queryer is required", nil, nil)
	}
	return nil
}

func decodeActivityJSON(raw []byte) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, recordError(RecordErrorInternal, "decode activity JSON failed", nil, err)
	}
	return value, nil
}
