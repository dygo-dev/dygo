package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hapyco/dygo/internal/entity/catalog"
	"github.com/jackc/pgx/v5"
)

// WithTreeReadScope supplies the read scope used to authorize a move destination.
// The mutation scope remains responsible for the source and parent field write.
func (s RecordStore) WithTreeReadScope(scope RecordScope) RecordStore {
	s.treeReadScope = &scope
	return s
}

// WithTreeReadScopeResolver defers read-policy compilation until a parent is
// actually assigned, using the mutation transaction rather than another pool read.
func (s RecordStore) WithTreeReadScopeResolver(resolve func(context.Context, RecordQueryer) (RecordScope, error)) RecordStore {
	s.treeReadScopeResolver = resolve
	return s
}

func (s RecordStore) lockTree(ctx context.Context, layout recordLayout) error {
	if layout.Tree == nil {
		return nil
	}
	if _, ok := s.queryer.(pgx.Tx); !ok {
		return recordError(RecordErrorInternal, "tree mutation requires a transaction", nil, nil)
	}
	// Lock before reading structural state, including reads performed by Hooks.
	_, err := s.queryer.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "dygo.tree:"+layout.Table)
	return err
}

func (s RecordStore) validateTreeParent(ctx context.Context, layout recordLayout, id int64, input RecordInput) error {
	if layout.Tree == nil {
		return nil
	}
	raw, changed := input[layout.Tree.ParentField]
	if !changed || rawIsNull(raw) {
		return nil
	}
	field := layout.FieldByName[layout.Tree.ParentField]
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return linkNameRequiredError(layout, field)
	}
	parent, err := s.linkValueCodec().idByName(ctx, layout, field, name)
	if err != nil {
		return err
	}
	read := s
	read.scope = s.treeReadScope
	if s.treeReadScopeResolver != nil {
		scope, err := s.treeReadScopeResolver(ctx, s.queryer)
		if err != nil {
			return err
		}
		read.scope = &scope
	}
	// A scoped caller must explicitly supply a destination read scope.
	if s.scope != nil && read.scope == nil {
		return treeAccessError()
	}
	if _, err := read.getRecordWithLayout(ctx, layout, parent); err != nil {
		if isTreeAccessError(err) {
			return treeAccessError()
		}
		return err
	}
	var invalid bool
	query := fmt.Sprintf(`WITH RECURSIVE ancestors AS (
 SELECT id, %s AS parent_id FROM %s WHERE id = $1
 UNION SELECT p.id, p.%s FROM %s p JOIN ancestors a ON p.id = a.parent_id
) SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)`, quoteIdent(field.Column), quoteIdent(layout.Table), quoteIdent(field.Column), quoteIdent(layout.Table))
	if err := s.queryer.QueryRow(ctx, query, parent, id).Scan(&invalid); err != nil {
		return err
	}
	if invalid {
		return recordError(RecordErrorValidation, "tree parent would create a cycle", nil, nil)
	}
	return nil
}

func (s RecordStore) restrictTreeDelete(ctx context.Context, layout recordLayout, id int64) error {
	if layout.Tree == nil {
		return nil
	}
	var children bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE %s = $1)", quoteIdent(layout.Table), quoteIdent(layout.FieldByName[layout.Tree.ParentField].Column))
	if err := s.queryer.QueryRow(ctx, query, id).Scan(&children); err != nil {
		return err
	}
	if children {
		return recordError(RecordErrorConstraintViolation, "tree node has children", nil, nil)
	}
	return nil
}

// Validate under a table lock as well as the runtime structural lock: existing
// non-tree writers do not yet know that the Entity is becoming a tree.
func validateTreeData(ctx context.Context, tx pgx.Tx, entities []catalog.LoadedEntity) error {
	for _, loaded := range entities {
		if loaded.Entity.Tree == nil {
			continue
		}
		table := entityTableName(loaded.AppName, loaded.Entity.Name)
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "dygo.tree:"+table); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "LOCK TABLE "+quoteIdent(table)+" IN SHARE ROW EXCLUSIVE MODE"); err != nil {
			return err
		}
		column := quoteIdent(strings.ReplaceAll(loaded.Entity.Tree.ParentField, "-", "_") + "_id")
		query := fmt.Sprintf(`WITH RECURSIVE reachable AS (
 SELECT id FROM %[1]s WHERE %[2]s IS NULL
 UNION SELECT n.id FROM %[1]s n JOIN reachable r ON n.%[2]s = r.id
) SELECT EXISTS (SELECT 1 FROM %[1]s WHERE id NOT IN (SELECT id FROM reachable))`, quoteIdent(table), column)
		var invalid bool
		if err := tx.QueryRow(ctx, query).Scan(&invalid); err != nil {
			return err
		}
		if invalid {
			return fmt.Errorf("cannot activate tree %s/%s: existing hierarchy contains a cycle or missing parent", loaded.AppName, loaded.Entity.Name)
		}
	}
	return nil
}
