package db

import (
	"context"
	"fmt"
	"strings"
)

type privateOwnerAccessContextKey struct{}

// WithPrivateOwnerAccess marks a RecordData operation that has already bound
// the authenticated actor to the private Entity owner scope.
func WithPrivateOwnerAccess(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, privateOwnerAccessContextKey{}, true)
}

func privateOwnerAccessFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	allowed, _ := ctx.Value(privateOwnerAccessContextKey{}).(bool)
	return allowed
}

func privateRecordLayout(ctx context.Context, meta MetadataEntityMeta) (recordLayout, error) {
	if meta.IsPrivate {
		if !privateOwnerAccessFromContext(ctx) {
			return recordLayout{}, recordError(RecordErrorPermissionDenied, "private Entity requires its owner-scoped service", nil, nil)
		}
	}
	return newRecordLayout(meta)
}

// WithPrivateOwnerScope returns a RecordStore constrained to one owner's
// private Entity Records. The metadata contract supplies the owner Link field;
// callers do not need to duplicate table or column naming rules.
func (s RecordStore) WithPrivateOwnerScope(ctx context.Context, appName string, entity string, ownerID int64) (RecordStore, error) {
	layout, err := s.recordLayoutByIdentity(ctx, appName, entity)
	if err != nil {
		return RecordStore{}, err
	}
	if !layout.IsPrivate || strings.TrimSpace(layout.PrivateOwnerField) == "" {
		return RecordStore{}, recordError(RecordErrorPermissionDenied, "Entity is not configured for private owner access", map[string]any{"entity": entity}, nil)
	}
	ownerField, ok := layout.FieldByName[layout.PrivateOwnerField]
	if !ok || ownerField.Type != "link" || ownerField.Column == "" {
		return RecordStore{}, recordError(RecordErrorInternal, "private owner field metadata is invalid", map[string]any{"entity": entity, "field": layout.PrivateOwnerField}, nil)
	}
	return s.WithScope(RecordScope{
		Where: fmt.Sprintf("%s = $1", quoteIdent(recordSelectSourceAlias)+"."+quoteIdent(ownerField.Column)),
		Args:  []any{ownerID},
	}), nil
}

// ValidateListByIdentity compiles a query without reading target Records. A
// scoped caller cannot persist filters on fields denied by every grant. Row
// conditions still apply when the saved query is executed through Record APIs.
func (s RecordStore) ValidateListByIdentity(ctx context.Context, app, entity string, params RecordListParams) error {
	params, err := normalizeRecordListParams(params)
	if err != nil {
		return err
	}
	layout, err := s.recordLayoutByIdentity(ctx, app, entity)
	if err != nil {
		return err
	}
	if layout.IsSingle || layout.IsCollection {
		return recordError(RecordErrorInvalidRequest, "Entity does not support Record lists", nil, nil)
	}
	if s.scope != nil {
		for _, filter := range params.Filters {
			if s.scope.FieldRead[strings.SplitN(filter.Field, ".", 2)[0]] == "FALSE" {
				return recordError(RecordErrorPermissionDenied, "filter field is not readable", map[string]any{"field": filter.Field}, nil)
			}
		}
	}
	_, err = s.listQuery(ctx, layout, params)
	return err
}
