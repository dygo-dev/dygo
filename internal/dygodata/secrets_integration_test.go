package dygodata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/recordsecret"
	"github.com/hapyco/dygo/internal/secrets"
	"github.com/hapyco/dygo/pkg/dygo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSDKSecretAccessAndTransaction(t *testing.T) {
	url := os.Getenv("DYGO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set DYGO_TEST_DATABASE_URL to run PostgreSQL regressions")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("dygo_secret_sdk_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	cfg := admin.Config().Copy()
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Error(err)
		}
	}()
	root, _ := filepath.Abs("../..")
	if _, err = db.SyncMetadataSchema(ctx, pool, root); err != nil {
		t.Fatal(err)
	}
	// Test-only metadata extends Core User in an isolated database. Production
	// callers use authored Entity metadata and additive schema sync.
	if _, err = pool.Exec(ctx, `ALTER TABLE "user" ADD COLUMN token_encrypted text; INSERT INTO field(name,entity_id,field_name,label,type,position) SELECT name||'.token',id,'token','Token','secret',100 FROM entity WHERE name='core.user'`); err != nil {
		t.Fatal(err)
	}
	keys := secrets.NewStore(t.TempDir())
	if _, err = keys.Init(); err != nil {
		t.Fatal(err)
	}
	if err = recordsecret.Init(keys, secrets.EnvironmentDevelopment); err != nil {
		t.Fatal(err)
	}
	ctx = recordsecret.WithStore(ctx, keys, secrets.EnvironmentDevelopment)
	record, err := db.NewRecordStore(pool).SystemWriter().InsertReturningByIdentity(ctx, "core", "user", db.RecordInput{"email": json.RawMessage(`"secret-test@example.com"`), "full-name": json.RawMessage(`"Secret Test"`), "token": json.RawMessage(`"private-sdk-value"`)}, db.SystemMutationBootstrap)
	if err != nil {
		t.Fatal(err)
	}
	id := record["id"].(int64)
	sdk := NewRecordData(pool, nil).AsSystem("integration-use")
	value, err := sdk.DecryptSecret(ctx, "core", "user", id, "token")
	if err != nil || value != "private-sdk-value" {
		t.Fatalf("SDK decrypt failed: %v", err)
	}
	var audit string
	if err = pool.QueryRow(ctx, `SELECT metadata::text FROM log WHERE title='Record secret decryption' ORDER BY id DESC LIMIT 1`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit, "integration-use") || strings.Contains(audit, "private-sdk-value") {
		t.Fatal("incorrect audit")
	}
	sentinel := errors.New("rollback")
	err = sdk.Transaction(ctx, func(ctx context.Context, records dygo.RecordData) error {
		if _, err := records.DecryptSecret(ctx, "core", "user", id, "token"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("transaction access failed: %v", err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM log WHERE title='Record secret decryption'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("transaction context was not preserved")
	}
	broken := &secretAudit{err: errors.New("audit failure")}
	if value, err = sdk.DecryptSecret(dygo.WithLogWriter(ctx, broken), "core", "user", id, "token"); err == nil || value != "" {
		t.Fatal("secret returned without audit")
	}
}
