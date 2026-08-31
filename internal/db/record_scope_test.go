package db

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRecordScopeConstrainsListAndRedactsFields(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	queryer := newUserRecordQueryer()
	queryer.rows = append(queryer.rows, newFakeRows([][]any{{int64(1), "a@example.com", now, now, "a@example.com", "A User", true, false, int64(1)}}))
	store := NewRecordStore(queryer).WithScope(RecordScope{
		Where:     `"_dygo_record"."owner_id" = $1`,
		Args:      []any{int64(7)},
		FieldRead: map[string]string{"full-name": "FALSE"},
	})
	result, err := store.ListRecords(context.Background(), "user", RecordListParams{})
	if err != nil {
		t.Fatalf("ListRecords() error = %v, want nil", err)
	}
	if _, exists := result.Records[0]["full-name"]; exists {
		t.Fatalf("ListRecords() record = %+v, want denied field omitted", result.Records[0])
	}
	query := queryer.queries[len(queryer.queries)-1]
	if !strings.Contains(query, `"_dygo_record"."owner_id" = $1`) || !strings.Contains(query, `LIMIT $2 OFFSET $3`) {
		t.Fatalf("ListRecords() SQL = %q, want compiled scope and shifted pagination", query)
	}
}

func TestRecordScopeConstrainsDirectIDQuery(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	queryer := newUserRecordQueryer()
	queryer.rows = append(queryer.rows, newFakeRows([][]any{{int64(1), "a@example.com", now, now, "a@example.com", "A User", true}}))
	store := NewRecordStore(queryer).WithScope(RecordScope{Where: `"_dygo_record"."owner_id" = $1`, Args: []any{int64(7)}})
	if _, err := store.GetRecord(context.Background(), "user", 1); err != nil {
		t.Fatalf("GetRecord() error = %v, want nil", err)
	}
	query := queryer.queries[len(queryer.queries)-1]
	if !strings.Contains(query, `("id" = $1) AND ("_dygo_record"."owner_id" = $2)`) {
		t.Fatalf("GetRecord() SQL = %q, want ID and compiled scope in one query", query)
	}
}

func TestRecordScopeAddsFieldWritePredicateToMutation(t *testing.T) {
	store := NewRecordStore(nil).WithScope(RecordScope{
		Where:      `"_dygo_record"."owner_id" = $1`,
		Args:       []any{int64(7)},
		FieldWrite: map[string]string{"status": "FALSE"},
	})
	where, args := store.scopedWriteWhere(`"id" = $1`, []any{int64(10)}, RecordInput{"status": []byte(`"Approved"`)})
	if !strings.Contains(where, `owner_id" = $2`) || !strings.Contains(where, `AND (FALSE)`) {
		t.Fatalf("scopedWriteWhere() = %q, want row and field predicates", where)
	}
	if len(args) != 2 {
		t.Fatalf("scopedWriteWhere() args = %#v, want ID and actor", args)
	}
}

func TestRecordScopeCompilesSamePolicyForProposedUpdate(t *testing.T) {
	store := NewRecordStore(nil).WithScope(RecordScope{Where: `"_dygo_record"."employee_id" = $1`, Args: []any{int64(7)}})
	layout := recordLayout{Entity: "leave-request", Fields: []recordField{{Name: "employee", Type: "link", Column: "employee_id", Storage: true}}}
	mutation := recordMutation{Columns: []string{"employee_id"}, Placeholders: []string{"$1::bigint"}, Values: []any{int64(12)}}
	predicate := store.proposedUpdatePredicate(layout, mutation, 2)
	if !strings.Contains(predicate, `AS "employee_id"`) || !strings.Contains(predicate, `"_dygo_proposed"."employee_id" = $3`) {
		t.Fatalf("proposedUpdatePredicate() = %q, want proposed value and shifted actor predicate", predicate)
	}
}
