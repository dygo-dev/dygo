package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/entity/fieldtype"
	"github.com/hapyco/dygo/internal/entity/schema"
	"github.com/hapyco/dygo/internal/recordsecret"
	"github.com/hapyco/dygo/internal/secrets"
	"github.com/hapyco/dygo/pkg/dygo"
)

func secretTestContext(t *testing.T) (context.Context, secrets.Store) {
	t.Helper()
	store := secrets.NewStore(t.TempDir())
	if _, err := store.Init(); err != nil {
		t.Fatal(err)
	}
	if err := recordsecret.Init(store, secrets.EnvironmentDevelopment); err != nil {
		t.Fatal(err)
	}
	return recordsecret.WithStore(context.Background(), store, secrets.EnvironmentDevelopment), store
}
func TestSecretStorageValidation(t *testing.T) {
	ctx, _ := secretTestContext(t)
	layout := recordLayout{AppName: "test", Entity: "account"}
	field := recordField{Name: "token", Type: "secret"}
	for _, raw := range []string{`""`, `123`, `{"value":"sensitive"}`} {
		if _, err := secretStorageValue(ctx, layout, field, json.RawMessage(raw)); err == nil || strings.Contains(err.Error(), "sensitive") {
			t.Fatal("invalid input accepted or exposed")
		}
	}
	if v, err := secretStorageValue(ctx, layout, field, json.RawMessage(`null`)); err != nil || v != nil {
		t.Fatal("clear failed")
	}
	field.Required = true
	if _, err := secretStorageValue(ctx, layout, field, json.RawMessage(`null`)); err == nil {
		t.Fatal("required clear accepted")
	}
	if _, err := secretStorageValue(context.Background(), layout, field, json.RawMessage(`"value"`)); err == nil {
		t.Fatal("missing keys accepted")
	}
}
func TestSecretHookInputs(t *testing.T) {
	child := recordLayout{Fields: []recordField{{Name: "token", Type: "secret"}}, FieldByName: map[string]recordField{"token": {Name: "token", Type: "secret"}}}
	layout := child
	layout.Collections = map[string]recordCollection{"rows": {Layout: &child}}
	input := RecordInput{"token": json.RawMessage(`"parent-value"`), "note": json.RawMessage(`"old"`), "rows": json.RawMessage(`[{"id":1,"token":"child-value","note":"old"}]`)}
	clean, restore, err := hiddenHookInput(layout, input)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(clean)
	if strings.Contains(string(raw), "value") {
		t.Fatal("Hook received a secret")
	}
	clean["note"] = json.RawMessage(`"changed"`)
	clean["rows"] = json.RawMessage(`[{"id":1,"note":"changed"}]`)
	if err = restore(); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(input)
	if !strings.Contains(string(raw), "parent-value") || !strings.Contains(string(raw), "child-value") || !strings.Contains(string(raw), "changed") {
		t.Fatal("lost submitted values or Hook edits")
	}
	clean, restore, _ = hiddenHookInput(layout, input)
	clean["token"] = json.RawMessage(`"injected"`)
	if err = restore(); err == nil {
		t.Fatal("Hook injection accepted")
	}
	clean, restore, _ = hiddenHookInput(layout, input)
	clean["rows"] = json.RawMessage(`[{"id":1,"token":"injected"}]`)
	if err = restore(); err == nil {
		t.Fatal("nested Hook injection accepted")
	}
	empty := RecordInput{}
	clean, restore, _ = hiddenHookInput(layout, empty)
	clean["rows"] = json.RawMessage(`[{"token":"injected"}]`)
	if err = restore(); err == nil {
		t.Fatal("new collection Hook injection accepted")
	}

}
func TestPostgresSecretLifecycleAndRotation(t *testing.T) {
	pool, metadata := auditDatabase(t)
	parent := auditEntity("vault", schema.Field{Name: "token", Label: "Token", Type: "secret", Required: true}, schema.Field{Name: "optional", Label: "Optional", Type: "secret"}, schema.Field{Name: "note", Label: "Note", Type: "text"}, schema.Field{Name: "rows", Label: "Rows", Type: "collection", Options: fieldtype.Options{Entity: "vault-row"}})
	child := auditEntity("vault-row", schema.Field{Name: "token", Label: "Token", Type: "secret", Required: true})
	child.Entity.IsCollection = true
	single := auditEntity("vault-settings", schema.Field{Name: "token", Label: "Token", Type: "secret"})
	single.Entity.IsSingle = true
	single.Entity.Naming = schema.Naming{}
	metadata.Entities = append(metadata.Entities, parent, child, single)
	syncAuditMetadata(t, pool, metadata)
	ctx, keys := secretTestContext(t)
	hooks := NewRecordHookRegistry()
	if err := hooks.RegisterEntity("audit", "vault", RecordBeforeValidate, "check-hidden", func(_ context.Context, h RecordHookContext) error {
		raw, _ := json.Marshal(h.Input)
		if strings.Contains(string(raw), "private-value") {
			return errors.New("secret leaked to Hook")
		}
		h.Input["note"] = json.RawMessage(`"hooked"`)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	store := NewRecordStoreWithHooks(pool, hooks)
	record, err := store.CreateRecordByIdentity(ctx, "audit", "vault", recordInput(map[string]string{"name": `"one"`, "token": `"private-value"`, "optional": `"optional-value"`, "rows": `[{"token":"child-private-value"}]`}))
	if err != nil {
		t.Fatal(err)
	}
	id := record["id"].(int64)
	raw, _ := json.Marshal(record)
	if strings.Contains(string(raw), "private-value") || strings.Contains(string(raw), "encrypted") {
		t.Fatal("response leaked secret")
	}
	if record["note"] != "hooked" {
		t.Fatal("Hook edit lost")
	}
	var ciphertext string
	if err = pool.QueryRow(ctx, `SELECT token_encrypted FROM audit_vault WHERE id=$1`, id).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ciphertext, "private-value") {
		t.Fatal("plaintext stored")
	}
	value, err := store.DecryptSecret(ctx, "audit", "vault", id, "token")
	if err != nil || value != "private-value" {
		t.Fatalf("decrypt failed: %v", err)
	}
	status, err := store.SecretStatusByIdentity(ctx, "audit", "vault", id)
	if err != nil || !status.Fields["token"] || len(status.Collections["rows"]) != 1 {
		t.Fatalf("status: %+v %v", status, err)
	}
	for childID := range status.Collections["rows"] {
		if value, err = store.DecryptSecret(ctx, "audit", "vault-row", childID, "token"); err != nil || value != "child-private-value" {
			t.Fatal("child decrypt failed")
		}
		if _, err = store.UpdateRecordByIdentity(ctx, "audit", "vault", id, recordInput(map[string]string{"rows": fmt.Sprintf(`[{"id":%d}]`, childID)})); err != nil {
			t.Fatal(err)
		}
	}
	scoped := store.WithScope(RecordScope{Where: "TRUE", FieldRead: map[string]string{"token": "FALSE", "rows": "FALSE"}})
	status, err = scoped.SecretStatusByIdentity(ctx, "audit", "vault", id)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := status.Fields["token"]; ok || len(status.Collections) != 0 {
		t.Fatal("presence bypassed field access")
	}
	if _, err = store.WithScope(RecordScope{Where: "FALSE"}).SecretStatusByIdentity(ctx, "audit", "vault", id); err == nil {
		t.Fatal("presence bypassed row access")
	}
	if _, err = store.UpdateRecordByIdentity(ctx, "audit", "vault", id, recordInput(map[string]string{"token": `null`})); err == nil {
		t.Fatal("required clear accepted")
	}
	if _, err = store.UpdateRecordByIdentity(ctx, "audit", "vault", id, recordInput(map[string]string{"optional": `null`})); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DecryptSecret(ctx, "audit", "vault", id, "optional"); !errors.Is(err, dygo.ErrSecretUnset) {
		t.Fatal("unset error missing")
	}
	// A transaction rollback restores the old ciphertext.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewRecordStoreWithHookPolicy(tx, RecordMutationHooksNone).UpdateRecordByIdentity(ctx, "audit", "vault", id, recordInput(map[string]string{"token": `"rollback-value"`})); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(ctx)
	if value, err = store.DecryptSecret(ctx, "audit", "vault", id, "token"); err != nil || value != "private-value" {
		t.Fatal("rollback lost secret")
	}
	if _, err = store.UpdateSingleRecord(ctx, "vault-settings", recordInput(map[string]string{"token": `"single-private-value"`})); err != nil {
		t.Fatal(err)
	}
	ring, err := recordsecret.BeginRotation(keys, secrets.EnvironmentDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpdateRecordByIdentity(ctx, "audit", "vault", id, recordInput(map[string]string{"token": `"blocked"`})); err == nil {
		t.Fatal("write during rotation accepted")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	changed, err := RotateRecordSecrets(ctx, conn.Conn(), ring)
	if err != nil || changed != 3 {
		t.Fatalf("rotate changed=%d error=%v", changed, err)
	}
	changed, err = RotateRecordSecrets(ctx, conn.Conn(), ring)
	if err != nil || changed != 0 {
		t.Fatalf("resume changed=%d error=%v", changed, err)
	}
	if err = recordsecret.FinishRotation(keys, secrets.EnvironmentDevelopment, ring); err != nil {
		t.Fatal(err)
	}
	if value, err = store.DecryptSecret(ctx, "audit", "vault", id, "token"); err != nil || value != "private-value" {
		t.Fatal("rotation lost secret")
	}
	if value, err = ring.Decrypt("audit/vault/token", ciphertext); err != nil || value != "private-value" {
		t.Fatal("backup key lost")
	}
}

func TestPostgresSecretRotationResumesCommittedBatches(t *testing.T) {
	pool, metadata := auditDatabase(t)
	metadata.Entities = append(metadata.Entities, auditEntity("rotate", schema.Field{Name: "token", Label: "Token", Type: "secret"}))
	syncAuditMetadata(t, pool, metadata)
	ctx, keys := secretTestContext(t)
	ring, err := recordsecret.Load(keys, secrets.EnvironmentDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	old, err := ring.Encrypt("audit/rotate/token", "private-value")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO audit_rotate(name,token_encrypted) SELECT 'row-'||i, $1 FROM generate_series(1,101) i`, old); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE audit_rotate SET token_encrypted='invalid' WHERE id=101`); err != nil {
		t.Fatal(err)
	}
	ring, err = recordsecret.BeginRotation(keys, secrets.EnvironmentDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, RecordKeyLock).Scan(&locked); err != nil || !locked {
		t.Fatal("lock failed")
	}
	second, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Release()
	if err = second.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, RecordKeyLock).Scan(&locked); err != nil || locked {
		t.Fatal("concurrent rotation lock accepted")
	}
	changed, err := RotateRecordSecrets(ctx, conn.Conn(), ring)
	if err == nil || changed != 100 {
		t.Fatalf("expected committed first batch and corrupt second batch: %d %v", changed, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE audit_rotate SET token_encrypted=$1 WHERE id=101`, old); err != nil {
		t.Fatal(err)
	}
	resumed, err := recordsecret.BeginRotation(keys, secrets.EnvironmentDevelopment)
	if err != nil || resumed.Active != ring.Active {
		t.Fatal("resume changed key")
	}
	changed, err = RotateRecordSecrets(ctx, conn.Conn(), resumed)
	if err != nil || changed != 1 {
		t.Fatalf("resume failed: %d %v", changed, err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_rotate WHERE token_encrypted::jsonb->>'key'=$1`, ring.Active).Scan(&count); err != nil || count != 101 {
		t.Fatal("not all ciphertexts rotated")
	}
}
