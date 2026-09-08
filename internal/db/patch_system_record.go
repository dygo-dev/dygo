package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hapyco/dygo/internal/patches"
	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

// Inputs stay private so serialized plans and diagnostics cannot expose secrets.
type patchSystemRecordOperation struct {
	entity string
	reason string
	id     int64
	values RecordInput
	match  RecordInput
}

func (p *patchOperationPlanner) planSystemRecord(r patchOperationReader) (PatchOperation, error) {
	if r.patch.Patch.Phase != patches.PhasePostSync {
		return PatchOperation{}, fmt.Errorf("%s requires post-sync phase", r.operation.Type)
	}
	if err := r.requireOperationFields(); err != nil {
		return PatchOperation{}, err
	}
	entity, err := r.requiredString("entity")
	if err != nil {
		return PatchOperation{}, err
	}
	target, err := p.entity(r.patch.AppName, entity)
	if err != nil {
		return PatchOperation{}, err
	}
	if strings.TrimSpace(r.patch.AppName) == "" || !target.Entity.IsSystem {
		return PatchOperation{}, fmt.Errorf("system Record operation requires an App-owned system Entity")
	}
	reason, err := r.requiredString("reason")
	if err != nil {
		return PatchOperation{}, err
	}
	record := &patchSystemRecordOperation{entity: entity, reason: reason}
	spec, _ := patches.OperationSpecFor(r.operation.Type)
	for _, field := range spec.Required {
		switch field {
		case "id":
			value, err := r.requiredString(field)
			if err != nil {
				return PatchOperation{}, err
			}
			record.id, err = strconv.ParseInt(value, 10, 64)
			if err != nil || record.id <= 0 {
				return PatchOperation{}, fmt.Errorf("id must be a positive Record ID")
			}
		case "values", "match":
			node, ok := r.operation.Fields[field]
			if !ok || node.Kind != yaml.MappingNode {
				return PatchOperation{}, fmt.Errorf("%s must be a mapping", field)
			}
			input := RecordInput{}
			for i := 0; i < len(node.Content); i += 2 {
				key := node.Content[i]
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
					return PatchOperation{}, fmt.Errorf("%s keys must be Field names", field)
				}
				var value any
				if err := node.Content[i+1].Decode(&value); err != nil {
					return PatchOperation{}, fmt.Errorf("invalid %s field %q", field, key.Value)
				}
				raw, err := json.Marshal(value)
				if err != nil {
					return PatchOperation{}, fmt.Errorf("invalid %s field %q", field, key.Value)
				}
				input[key.Value] = raw
			}
			if field == "match" {
				if len(input) == 0 {
					return PatchOperation{}, fmt.Errorf("match must not be empty")
				}
				record.match = input
			} else {
				record.values = input
			}
		}
	}
	if record.match != nil {
		if _, err := mergeSystemRecordMatch(record.match, record.values); err != nil {
			return PatchOperation{}, err
		}
		meta := MetadataEntityMeta{}
		meta.Name = target.Entity.Name
		for _, field := range target.Entity.Fields {
			meta.Fields = append(meta.Fields, MetadataField{Name: field.Name, Type: field.Type, Unique: field.Unique})
		}
		for _, constraint := range target.Entity.Constraints {
			fields, _ := json.Marshal(constraint.Fields)
			meta.Constraints = append(meta.Constraints, MetadataConstraint{Name: constraint.Name, Type: constraint.Type, Fields: fields})
		}
		if err := ValidateRecordMatch(meta, sortedRecordInputNames(record.match)); err != nil {
			return PatchOperation{}, err
		}
	}
	// Preview deliberately prints field names only; values can include nested secrets.
	description := fmt.Sprintf("%s %s/%s: %s", r.operation.Type, r.patch.AppName, entity, reason)
	if record.id != 0 {
		description += fmt.Sprintf(" (id %d)", record.id)
	}
	if record.values != nil {
		description += fmt.Sprintf("; values [%s] (redacted)", strings.Join(sortedRecordInputNames(record.values), ", "))
	}
	if record.match != nil {
		description += fmt.Sprintf("; match [%s] (redacted)", strings.Join(sortedRecordInputNames(record.match), ", "))
	}
	return r.base(PatchOperation{Description: description, record: record}), nil
}

func executePatchOperation(ctx context.Context, tx pgx.Tx, patch PlannedPatch, operation PatchOperation) error {
	if operation.record == nil {
		if patches.IsSystemRecordOperation(operation.Type) {
			return fmt.Errorf("system Record operation is missing its structured input")
		}
		_, err := tx.Exec(ctx, operation.SQL)
		return err
	}
	if patch.Phase != patches.PhasePostSync {
		return fmt.Errorf("system Record operations require post-sync phase")
	}
	record := operation.record
	writer := NewSystemRecordWriter(tx).Scoped(patch.AppName, record.reason)
	switch operation.Type {
	case patches.OperationSystemRecordCreate:
		_, err := writer.Create(ctx, record.entity, record.values)
		return err
	case patches.OperationSystemRecordUpdate:
		_, err := writer.Update(ctx, record.entity, record.id, record.values)
		return err
	case patches.OperationSystemRecordDelete:
		return writer.Delete(ctx, record.entity, record.id)
	case patches.OperationSystemRecordUpsert:
		_, err := writer.Upsert(ctx, record.entity, record.match, record.values)
		return err
	default:
		return fmt.Errorf("unsupported structured Record operation %q", operation.Type)
	}
}
