package db

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"math/big"
	"strings"

	"github.com/hapyco/dygo/internal/recordsecret"
)

// ScopedSystemRecordWriter limits trusted mutations to one App's system Entities.
type ScopedSystemRecordWriter struct {
	writer SystemRecordWriter
	app    string
	reason string
}

// Scoped binds a trusted writer to an App and an auditable reason.
func (w SystemRecordWriter) Scoped(app, reason string) ScopedSystemRecordWriter {
	return ScopedSystemRecordWriter{writer: w, app: strings.TrimSpace(app), reason: strings.TrimSpace(reason)}
}

func (w ScopedSystemRecordWriter) prepare(ctx context.Context, entity string) (RecordStore, MetadataEntityMeta, error) {
	if w.app == "" || w.reason == "" {
		return RecordStore{}, MetadataEntityMeta{}, recordError(RecordErrorInvalidRequest, "trusted system Record writer requires App scope and a non-empty reason", nil, nil)
	}
	if err := w.writer.store.requireQueryer(); err != nil {
		return RecordStore{}, MetadataEntityMeta{}, err
	}
	meta, err := w.writer.store.metadata.GetEntityMetaByIdentity(ctx, w.app, entity)
	if err != nil {
		code := RecordErrorInternal
		if IsMetadataNotFound(err) {
			code = RecordErrorNotFound
		}
		return RecordStore{}, MetadataEntityMeta{}, recordError(code, "load owned system Entity metadata failed", map[string]any{"app": w.app, "entity": entity}, err)
	}
	if meta.App.Name != w.app || !meta.IsSystem {
		return RecordStore{}, MetadataEntityMeta{}, recordError(RecordErrorInvalidRequest, "trusted system Record writer requires an owned system Entity", map[string]any{"app": w.app, "entity": entity}, nil)
	}
	store, err := w.writer.mutationStore(w.app, entity, SystemMutationFramework)
	return store, meta, err
}

// Create creates an owned system Record using framework validation and Activity.
func (w ScopedSystemRecordWriter) Create(ctx context.Context, entity string, input RecordInput) (Record, error) {
	store, _, err := w.prepare(ctx, entity)
	if err != nil {
		return nil, err
	}
	return store.CreateRecordByIdentity(WithActivitySystemReason(ctx, w.reason), w.app, entity, input)
}

// Update updates an owned system Record.
func (w ScopedSystemRecordWriter) Update(ctx context.Context, entity string, id int64, input RecordInput) (Record, error) {
	store, _, err := w.prepare(ctx, entity)
	if err != nil {
		return nil, err
	}
	return store.UpdateRecordByIdentity(WithActivitySystemReason(ctx, w.reason), w.app, entity, id, input)
}

// Delete deletes an owned system Record.
func (w ScopedSystemRecordWriter) Delete(ctx context.Context, entity string, id int64) error {
	store, _, err := w.prepare(ctx, entity)
	if err != nil {
		return err
	}
	return store.SystemWriter().DeleteByIdentity(WithActivitySystemReason(ctx, w.reason), w.app, entity, id, SystemMutationFramework)
}

// Upsert matches a unique metadata key and creates or updates an owned system Record.
func (w ScopedSystemRecordWriter) Upsert(ctx context.Context, entity string, match, input RecordInput) (Record, error) {
	store, meta, err := w.prepare(ctx, entity)
	if err != nil {
		return nil, err
	}
	match = normalizeRecordInput(match)
	if err := ValidateRecordMatch(meta, sortedRecordInputNames(match)); err != nil {
		return nil, recordError(RecordErrorInvalidRequest, err.Error(), nil, nil)
	}
	merged, err := mergeSystemRecordMatch(match, input)
	if err != nil {
		return nil, err
	}
	ctx = recordsecret.WithOperation(WithActivitySystemReason(ctx, w.reason))
	return store.withRecordMutation(ctx, func(txStore RecordStore) (Record, error) {
		existing, err := txStore.FindRecordByIdentity(ctx, w.app, entity, match)
		if err != nil {
			if !isRecordNotFound(err) {
				return nil, err
			}
			return txStore.createRecordByIdentity(ctx, w.app, entity, merged)
		}
		id, err := activityRecordID(existing)
		if err != nil {
			return nil, err
		}
		return txStore.updateRecordByIdentity(ctx, w.app, entity, id, input)
	})
}

func mergeSystemRecordMatch(match, input RecordInput) (RecordInput, error) {
	merged := maps.Clone(normalizeRecordInput(input))
	for name, value := range match {
		var matchValue any
		if err := decodeSystemMatchValue(value, &matchValue); err != nil || matchValue == nil {
			return nil, recordError(RecordErrorInvalidRequest, "system Record match values must be valid and non-null", map[string]any{"field": name}, nil)
		}
		if provided, ok := merged[name]; ok {
			var inputValue any
			if err := decodeSystemMatchValue(provided, &inputValue); err != nil || !systemMatchValuesEqual(matchValue, inputValue) {
				return nil, recordError(RecordErrorInvalidRequest, "system Record values conflict with match", map[string]any{"field": name}, nil)
			}
		}
		merged[name] = value
	}
	return merged, nil
}

func systemMatchValuesEqual(left, right any) bool {
	if leftNumber, ok := left.(json.Number); ok {
		rightNumber, ok := right.(json.Number)
		if !ok {
			return false
		}
		leftValue, leftOK := new(big.Rat).SetString(string(leftNumber))
		rightValue, rightOK := new(big.Rat).SetString(string(rightNumber))
		return leftOK && rightOK && leftValue.Cmp(rightValue) == 0
	}
	return recordValuesEqual(left, right)
}

func decodeSystemMatchValue(raw json.RawMessage, target *any) error {
	if !json.Valid(raw) {
		return recordError(RecordErrorInvalidRequest, "invalid JSON match value", nil, nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(target)
}
