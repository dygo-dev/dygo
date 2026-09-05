package db

import (
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
)

func TestSystemRecordWriterRejectsInvalidPolicy(t *testing.T) {
	err := NewSystemRecordWriter(&fakeRecordQueryer{}).InsertByIdentity(context.Background(), "core", "user", RecordInput{}, SystemMutationPolicy("nope"))
	if err == nil || !strings.Contains(err.Error(), "system mutation policy is invalid") {
		t.Fatalf("InsertByIdentity(invalid policy) error = %v, want invalid policy", err)
	}
}

func fakeSystemInsertRows(sql string) pgx.Rows {
	columns := 0
	if strings.HasPrefix(sql, `INSERT INTO "activity"`) {
		columns = 4 + len(activityFieldRows())
	} else if strings.HasPrefix(sql, `INSERT INTO "patch_run"`) {
		columns = 11
	} else {
		return nil
	}
	values := make([]any, columns)
	values[0], values[1] = int64(1), "system-record"
	return newFakeRows([][]any{values})
}
