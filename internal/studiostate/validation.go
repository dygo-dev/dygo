package studiostate

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/dygodata"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/internal/shape"
	"github.com/hapyco/dygo/pkg/dygo"
)

func (s Store) validate(ctx context.Context, actor dygo.Actor, identity string, filters []Filter) error {
	if err := requireActor(actor); err != nil {
		return err
	}
	parts := strings.Split(identity, "/")
	if len(parts) != 2 {
		return invalid("entity must be app/entity")
	}
	for _, part := range parts {
		if err := shape.ValidateMetadataName("entity", part); err != nil {
			return invalid("invalid Entity identity")
		}
	}
	if filters == nil || len(filters) > 100 {
		return invalid("filters must be an array of at most 100 predicates")
	}
	app, entity := parts[0], parts[1]
	scope, err := permissions.NewChecker(s.queryer).RecordScope(ctx, permissions.Request{Actor: actor, Resource: dygo.Resource{Kind: dygo.ResourceEntity, App: app, Name: entity}, Action: permissions.ActionRead})
	if err != nil {
		return err
	}
	if scope.Where == "FALSE" {
		return permissions.Error{Code: permissions.ErrorDenied, Message: "Entity is not readable"}
	}
	meta, err := db.NewMetadataReader(s.queryer).GetEntityMetaByIdentity(ctx, app, entity)
	if err != nil {
		return err
	}
	if meta.IsPrivate {
		return db.RecordError{Code: db.RecordErrorPermissionDenied, Message: "private Entity cannot be filtered"}
	}
	params := db.RecordListParams{Limit: 1}
	for _, filter := range filters {
		// Studio predicates address one visible field. Relationship paths need
		// their own field-permission contract before Studio can persist them.
		if strings.Contains(filter.Field, ".") {
			return invalid("saved filters require direct fields")
		}
		params.Filters = append(params.Filters, db.RecordFilter{Field: filter.Field, Operator: filter.Operator, Value: filter.Value})
		if filter.Operator == "empty" || filter.Operator == "not-empty" {
			continue
		}
		for _, field := range meta.Fields {
			if field.Name != filter.Field || field.Type != "link" {
				continue
			}
			var options struct {
				App    string `json:"app"`
				Entity string `json:"entity"`
			}
			if err := json.Unmarshal(field.Options, &options); err != nil {
				return err
			}
			if options.App == "" {
				options.App = app
			}
			// Find uses the actor's target Record scope, including conditional
			// access. A guessed linked name must not validate through system access.
			_, err := dygodata.NewRecordData(s.queryer, nil).AsActor(actor).Find(ctx, options.App, options.Entity, input(map[string]any{"name": filter.Value}))
			if err != nil {
				var recordErr db.RecordError
				var permissionErr permissions.Error
				if (errors.As(err, &recordErr) && (recordErr.Code == db.RecordErrorNotFound || recordErr.Code == db.RecordErrorPermissionDenied)) ||
					(errors.As(err, &permissionErr) && permissionErr.Code == permissions.ErrorDenied) {
					return db.RecordError{Code: db.RecordErrorValidation, Message: "linked filter value is unavailable", Details: map[string]any{"field": filter.Field}}
				}
				return err
			}
		}
	}
	return db.NewRecordStore(s.queryer).WithScope(db.RecordScope{Where: scope.Where, Args: scope.Args, FieldRead: scope.FieldRead, FieldWrite: scope.FieldWrite}).ValidateListByIdentity(ctx, app, entity, params)
}
