package db

import (
	"context"
	"strings"
)

func privateRecordLayout(ctx context.Context, meta MetadataEntityMeta) (recordLayout, error) {
	if meta.IsPrivate {
		if _, system := ActivitySystemReasonFromContext(ctx); !system {
			return recordLayout{}, recordError(RecordErrorPermissionDenied, "private Entity requires its owner-scoped service", nil, nil)
		}
	}
	return newRecordLayout(meta)
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
