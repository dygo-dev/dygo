package db

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/entity/schema"
)

func TestScopedSystemWriterRequiresScopeAndReason(t *testing.T) {
	for _, scope := range [][2]string{{"", "repair"}, {"core", " \t"}} {
		writer := NewSystemRecordWriter(nil).Scoped(scope[0], scope[1])
		_, err := writer.Create(context.Background(), "user", nil)
		if err == nil || !strings.Contains(err.Error(), "App scope and a non-empty reason") {
			t.Fatalf("Create scope %v: %v", scope, err)
		}
	}
}

func TestScopedSystemWriterRejectsUnownedAndNonSystemEntities(t *testing.T) {
	for _, tc := range []struct {
		name, app string
		queryer   *fakeRecordQueryer
	}{
		{"foreign", "crm", newSystemUserRecordQueryer()},
		{"non-system", "core", newUserRecordQueryer()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSystemRecordWriter(tc.queryer).Scoped(tc.app, "repair").Create(context.Background(), "user", nil)
			if err == nil || !strings.Contains(err.Error(), "owned system Entity") {
				t.Fatalf("Create: %v", err)
			}
		})
	}
}

func TestSystemRecordMatchMerge(t *testing.T) {
	input := RecordInput{"title": json.RawMessage(`"original"`)}
	merged, err := mergeSystemRecordMatch(RecordInput{"key": json.RawMessage(`"abc"`)}, input)
	if err != nil || string(merged["key"]) != `"abc"` {
		t.Fatalf("merge = %v, %v", merged, err)
	}
	if _, ok := input["key"]; ok {
		t.Fatal("merge mutated caller input")
	}
	for _, value := range []string{`1.0`, `1e0`} {
		if _, err := mergeSystemRecordMatch(RecordInput{"key": json.RawMessage(`1`)}, RecordInput{"key": json.RawMessage(value)}); err != nil {
			t.Fatalf("equivalent numeric match %s: %v", value, err)
		}
	}
	for _, tc := range []struct{ match, value string }{
		{`"abc"`, `"def"`}, {`null`, `null`}, {`bad`, `"abc"`},
		{`9007199254740992`, `9007199254740993`},
	} {
		_, err := mergeSystemRecordMatch(RecordInput{"key": json.RawMessage(tc.match)}, RecordInput{"key": json.RawMessage(tc.value)})
		if err == nil {
			t.Fatalf("accepted conflicting or invalid match %s, %s", tc.match, tc.value)
		}
	}
}

func TestSystemRecordMatchConflictRedactsValues(t *testing.T) {
	_, err := mergeSystemRecordMatch(RecordInput{"credential": json.RawMessage(`"sensitive-match"`)}, RecordInput{"credential": json.RawMessage(`"sensitive-input"`)})
	encoded, _ := json.Marshal(err)
	if err == nil || strings.Contains(err.Error(), "sensitive") || strings.Contains(string(encoded), "sensitive") {
		t.Fatalf("conflict error leaked a value: %v", err)
	}
}

func TestScopedSystemWriterRequiresUniqueMatch(t *testing.T) {
	_, err := NewSystemRecordWriter(newSystemUserRecordQueryer()).Scoped("core", "repair").Upsert(context.Background(), "user", RecordInput{"full-name": json.RawMessage(`"User"`)}, nil)
	if err == nil || !strings.Contains(err.Error(), "not backed by a unique") {
		t.Fatalf("Upsert = %v", err)
	}
}

func TestPostgresBusinessSystemWriter(t *testing.T) {
	pool, metadata := auditDatabase(t)
	entity := auditEntity("sync-state", schema.Field{Name: "code", Type: "text", Label: "Code", Required: true, Unique: true}, schema.Field{Name: "token", Type: "secret", Label: "Token"})
	entity.Entity.IsSystem = true
	metadata.Entities = append(metadata.Entities, entity)
	syncAuditMetadata(t, pool, metadata)
	ctx, _ := secretTestContext(t)
	writer := NewSystemRecordWriter(pool).Scoped("audit", "refresh integration state")
	input := RecordInput{"name": json.RawMessage(`"first"`), "token": json.RawMessage(`"private-token"`)}
	if _, err := writer.Create(ctx, "sync-state", input); err == nil {
		t.Fatal("required code accepted")
	}
	record, err := writer.Upsert(ctx, "sync-state", RecordInput{"code": json.RawMessage(`"external-1"`)}, input)
	if err != nil {
		t.Fatal(err)
	}
	id := record["id"].(int64)
	encoded, _ := json.Marshal(record)
	if strings.Contains(string(encoded), "private-token") {
		t.Fatal("returned plaintext secret")
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT token_encrypted FROM audit_sync_state WHERE id=$1`, id).Scan(&stored); err != nil || stored == "private-token" {
		t.Fatalf("secret storage error: %v", err)
	}
	for _, owner := range []string{"core", "foreign"} {
		if _, err := NewSystemRecordWriter(pool).Scoped(owner, "foreign").Create(ctx, "sync-state", input); err == nil {
			t.Fatal("foreign owner accepted")
		}
	}
	if _, err := writer.Create(ctx, "session", input); err == nil {
		t.Fatal("Business App wrote Core target")
	}
	store := NewRecordStore(pool)
	if _, err := store.GetRecordByIdentity(ctx, "audit", "sync-state", id); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateRecordByIdentity(ctx, "audit", "sync-state", id, RecordInput{"code": json.RawMessage(`"changed"`)}); err == nil {
		t.Fatal("ordinary update accepted")
	}
	if err := store.DeleteRecordByIdentity(ctx, "audit", "sync-state", id); err == nil {
		t.Fatal("ordinary delete accepted")
	}
	if _, err := writer.Update(ctx, "sync-state", id, RecordInput{"code": json.RawMessage(`"changed"`)}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Delete(ctx, "sync-state", id); err != nil {
		t.Fatal(err)
	}
}
