package db

import (
	"context"
	"strings"
	"testing"

	"github.com/hapyco/dygo/pkg/dygo"
)

func TestRecordStoreLockRecordsUsesDeterministicOrderAndForUpdate(t *testing.T) {
	queryer := newUserRecordQueryer()
	queryer.rows = append(queryer.rows, newFakeRows([][]any{
		{int64(7), "user-7", nil, nil, "a@example.com", "A User", true},
	}))

	result, err := NewRecordStore(queryer).LockRecordsByIdentity(context.Background(), "core", "user", RecordListParams{
		Sort: []RecordSort{{Field: "full-name", Desc: true}},
	})
	if err != nil {
		t.Fatalf("LockRecordsByIdentity() error = %v, want nil", err)
	}
	if result.Count != 1 || result.Records[0]["email"] != "a@example.com" {
		t.Fatalf("LockRecordsByIdentity() result = %+v, want one user", result)
	}
	query := queryer.queries[len(queryer.queries)-1]
	if !strings.Contains(query, `ORDER BY "full_name" DESC, "id" ASC`) || !strings.HasSuffix(query, "FOR UPDATE") {
		t.Fatalf("lock query = %q, want deterministic order and FOR UPDATE", query)
	}
}

func TestRecordStoreGroupByValidatesFieldsAndBuildsAggregateQuery(t *testing.T) {
	queryer := newUserRecordQueryer()
	queryer.rows = append(queryer.rows, newFakeRows([][]any{{"A User", int64(1)}}))

	groups, err := NewRecordStore(queryer).GroupRecordsByIdentity(context.Background(), "core", "user", dygo.GroupByParams{
		GroupBy:    []string{"full-name"},
		Aggregates: []dygo.AggregateSpec{{Function: dygo.AggregateCount}},
	})
	if err != nil {
		t.Fatalf("GroupRecordsByIdentity() error = %v, want nil", err)
	}
	if len(groups) != 1 || groups[0].Group["full-name"] != "A User" || groups[0].Aggregates["count"] != int64(1) {
		t.Fatalf("groups = %#v, want one grouped count", groups)
	}
	query := queryer.queries[len(queryer.queries)-1]
	if !strings.Contains(query, `GROUP BY "full_name"`) || !strings.Contains(query, `COUNT(*)`) {
		t.Fatalf("group query = %q, want validated group and count", query)
	}

	_, err = NewRecordStore(newUserRecordQueryer()).GroupRecordsByIdentity(context.Background(), "core", "user", dygo.GroupByParams{
		GroupBy:    []string{"does-not-exist"},
		Aggregates: []dygo.AggregateSpec{{Function: dygo.AggregateCount}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("invalid group error = %v, want unknown field", err)
	}
}

func TestRecordStoreAggregateSupportsCountAndSum(t *testing.T) {
	queryer := newUserRecordQueryer()
	queryer.rows = append(queryer.rows, newFakeRows([][]any{{int64(3), int64(42)}}))

	results, err := NewRecordStore(queryer).AggregateRecordsByIdentity(context.Background(), "core", "user", dygo.AggregateParams{
		Aggregates: []dygo.AggregateSpec{
			{Function: dygo.AggregateCount},
			{Function: dygo.AggregateSum, Field: "id"},
		},
	})
	if err != nil {
		t.Fatalf("AggregateRecordsByIdentity() error = %v, want nil", err)
	}
	if len(results) != 2 || results[0].Value != int64(3) || results[1].Value != int64(42) {
		t.Fatalf("aggregate results = %#v, want count 3 and sum 42", results)
	}
	query := queryer.queries[len(queryer.queries)-1]
	if !strings.Contains(query, `COUNT(*) AS "count"`) || !strings.Contains(query, `SUM("id")`) {
		t.Fatalf("aggregate query = %q, want count and sum", query)
	}
}
