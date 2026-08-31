package db

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// RecordScope is compiled from one access policy AST for a Record operation.
type RecordScope struct {
	Where      string
	Args       []any
	FieldRead  map[string]string
	FieldWrite map[string]string
}

var placeholderPattern = regexp.MustCompile(`\$([0-9]+)`)

// WithScope returns a Record store constrained by scope.
func (s RecordStore) WithScope(scope RecordScope) RecordStore {
	s.scope = &scope
	return s
}

func (s RecordStore) scopedWhere(where string, args []any) (string, []any) {
	if s.scope == nil || s.scope.Where == "" || s.scope.Where == "TRUE" {
		return where, args
	}
	scopeWhere := shiftPlaceholders(s.scope.Where, len(args))
	if where == "" {
		where = scopeWhere
	} else {
		where = "(" + where + ") AND (" + scopeWhere + ")"
	}
	return where, append(args, s.scope.Args...)
}

func (s RecordStore) scopedWriteWhere(where string, args []any, input RecordInput) (string, []any) {
	where, args = s.scopedWhere(where, args)
	if s.scope == nil {
		return where, args
	}
	seen := map[string]bool{}
	for field := range input {
		predicate, restricted := s.scope.FieldWrite[field]
		if !restricted || seen[predicate] {
			continue
		}
		seen[predicate] = true
		where += " AND (" + shiftPlaceholders(predicate, len(args)-len(s.scope.Args)) + ")"
	}
	return where, args
}

func (s RecordStore) scopedReadWhere(where string, args []any, fields []string) (string, []any) {
	where, args = s.scopedWhere(where, args)
	if s.scope == nil {
		return where, args
	}
	offset := len(args) - len(s.scope.Args)
	seen := map[string]bool{}
	for _, path := range fields {
		field := strings.Split(path, ".")[0]
		predicate, restricted := s.scope.FieldRead[field]
		if !restricted || seen[predicate] {
			continue
		}
		seen[predicate] = true
		where += " AND (" + shiftPlaceholders(predicate, offset) + ")"
	}
	return where, args
}

// AuthorizeField checks the current Record scope for one field on one row.
// It is used by framework services that mutate a field outside the normal
// Record update path, such as file attachments.
func (s RecordStore) AuthorizeField(ctx context.Context, appName string, entity string, id int64, field string, write bool) error {
	if s.scope == nil {
		return nil
	}
	predicate, restricted := s.scope.FieldRead[field]
	if write {
		predicate, restricted = s.scope.FieldWrite[field]
	}
	if !restricted || predicate == "" || predicate == "TRUE" {
		return nil
	}
	layout, err := s.recordLayoutByIdentity(ctx, appName, entity)
	if err != nil {
		return err
	}
	var allowed bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s AS %s WHERE %s.id = $1 AND (%s))", quoteIdent(layout.Table), quoteIdent(recordSelectSourceAlias), quoteIdent(recordSelectSourceAlias), shiftPlaceholders(predicate, 1))
	if err := s.queryer.QueryRow(ctx, query, append([]any{id}, s.scope.Args...)...).Scan(&allowed); err != nil {
		return recordError(RecordErrorInternal, "evaluate field access failed", map[string]any{"entity": layout.Entity, "field": field}, err)
	}
	if !allowed {
		return recordError(RecordErrorPermissionDenied, "permission denied", map[string]any{"entity": layout.Entity, "field": field}, nil)
	}
	return nil
}

func (s RecordStore) validateProposedScope(ctx context.Context, layout recordLayout, mutation recordMutation, input RecordInput) error {
	if s.scope == nil || s.scope.Where == "" || s.scope.Where == "TRUE" {
		return nil
	}
	values := map[string]string{}
	for index, column := range mutation.Columns {
		values[column] = mutation.Placeholders[index]
	}
	columns := append([]string{systemColumnName, systemColumnOwnerID}, storedLayoutColumns(layout)...)
	selects := make([]string, 0, len(columns))
	for _, column := range columns {
		expression := values[column]
		if expression == "" {
			expression = "NULL"
		}
		selects = append(selects, expression+" AS "+quoteIdent(column))
	}
	predicates := []string{s.scope.Where}
	for field := range input {
		if predicate, restricted := s.scope.FieldWrite[field]; restricted {
			predicates = append(predicates, predicate)
		}
	}
	where := shiftPlaceholders("("+strings.Join(predicates, ") AND (")+")", len(mutation.Values))
	args := append(append([]any{}, mutation.Values...), s.scope.Args...)
	var allowed bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM (SELECT %s) AS %s WHERE %s)", strings.Join(selects, ", "), quoteIdent(recordSelectSourceAlias), where)
	if err := s.queryer.QueryRow(ctx, query, args...).Scan(&allowed); err != nil {
		return recordError(RecordErrorInternal, "evaluate proposed record access failed", map[string]any{"entity": layout.Entity}, err)
	}
	if !allowed {
		return recordError(RecordErrorPermissionDenied, "permission denied", map[string]any{"entity": layout.Entity}, nil)
	}
	return nil
}

func storedLayoutColumns(layout recordLayout) []string {
	columns := []string{}
	for _, field := range layout.Fields {
		if field.Storage && !field.SystemName {
			columns = append(columns, field.Column)
		}
	}
	return columns
}

func shiftPlaceholders(sql string, offset int) string {
	if offset == 0 {
		return sql
	}
	return placeholderPattern.ReplaceAllStringFunc(sql, func(match string) string {
		value, _ := strconv.Atoi(match[1:])
		return fmt.Sprintf("$%d", value+offset)
	})
}

func (s RecordStore) selectList(layout recordLayout, scopeOffset int) string {
	list := layout.selectList()
	if s.scope == nil || len(s.scope.FieldRead) == 0 {
		return list
	}
	for _, field := range sortedScopeFields(s.scope.FieldRead) {
		list += ", (" + shiftPlaceholders(s.scope.FieldRead[field], scopeOffset) + ") AS " + quoteIdent("__dygo_read_"+field)
	}
	return list
}

func (s RecordStore) proposedUpdatePredicate(layout recordLayout, mutation recordMutation, scopeOffset int) string {
	if s.scope == nil || s.scope.Where == "" || s.scope.Where == "TRUE" {
		return ""
	}
	changed := map[string]string{}
	for index, column := range mutation.Columns {
		changed[column] = mutation.Placeholders[index]
	}
	columns := append([]string{systemColumnName, systemColumnOwnerID}, storedLayoutColumns(layout)...)
	selects := make([]string, 0, len(columns))
	for _, column := range columns {
		expression := changed[column]
		if expression == "" {
			expression = quoteIdent(recordSelectSourceAlias) + "." + quoteIdent(column)
		}
		selects = append(selects, expression+" AS "+quoteIdent(column))
	}
	predicate := strings.ReplaceAll(s.scope.Where, quoteIdent(recordSelectSourceAlias), quoteIdent("_dygo_proposed"))
	predicate = shiftPlaceholders(predicate, scopeOffset)
	return fmt.Sprintf("EXISTS (SELECT 1 FROM (SELECT %s) AS %s WHERE %s)", strings.Join(selects, ", "), quoteIdent("_dygo_proposed"), predicate)
}

func (s RecordStore) scopedRecordFromValues(layout recordLayout, values []any) (Record, error) {
	baseCount := layout.recordValueCount()
	if len(values) < baseCount {
		return nil, recordError(RecordErrorInternal, "record column count did not match metadata", map[string]any{"entity": layout.Entity, "expected": baseCount, "actual": len(values)}, nil)
	}
	record, err := layout.recordFromValues(values[:baseCount])
	if err != nil {
		return nil, err
	}
	if s.scope == nil || len(values) == baseCount {
		return record, nil
	}
	fields := sortedScopeFields(s.scope.FieldRead)
	if len(values) != baseCount+len(fields) {
		return nil, recordError(RecordErrorInternal, "record access column count did not match scope", map[string]any{"entity": layout.Entity}, nil)
	}
	for index, field := range fields {
		allowed, ok := values[baseCount+index].(bool)
		if !ok {
			return nil, recordError(RecordErrorInternal, "record access value was invalid", map[string]any{"entity": layout.Entity, "field": field}, nil)
		}
		if !allowed {
			delete(record, field)
		}
	}
	return record, nil
}

func sortedScopeFields(values map[string]string) []string {
	fields := make([]string, 0, len(values))
	for field := range values {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}
