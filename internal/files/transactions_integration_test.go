package files

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/dygodata"
	"github.com/hapyco/dygo/pkg/dygo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests own a disposable database and are skipped unless explicitly enabled.
func TestTransactionsRollbackMetadataWhenEnqueueFails(t *testing.T) {
	pool := transactionTestDatabase(t)
	ctx := context.Background()
	store := db.NewRecordStoreWithHookPolicy(pool, db.RecordMutationHooksNone)
	user, err := store.SystemWriter().InsertReturningByIdentity(ctx, "core", "user", db.RecordInput{
		"email":     json.RawMessage(`"transaction-test@example.com"`),
		"full-name": json.RawMessage(`"Transaction Test"`),
	}, db.SystemMutationBootstrap)
	if err != nil {
		t.Fatal(err)
	}
	userID := user["id"].(int64)
	file, err := store.SystemWriter().InsertReturningByIdentity(ctx, "core", "file", db.RecordInput{
		"filename":     json.RawMessage(`"test.txt"`),
		"storage-key":  json.RawMessage(`"transaction-test-key"`),
		"checksum":     json.RawMessage(`"checksum"`),
		"content-type": json.RawMessage(`"text/plain"`),
		"size":         json.RawMessage(`4`),
		"private":      json.RawMessage(`true`),
		"actor":        json.RawMessage(`"transaction-test@example.com"`),
		"retired":      json.RawMessage(`false`),
		"app":          json.RawMessage(`"core"`),
		"entity":       json.RawMessage(`"user"`),
		"record-id":    json.RawMessage(fmt.Sprintf(`%d`, userID)),
		"field":        json.RawMessage(`"full-name"`),
	}, db.SystemMutationBootstrap)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `UPDATE "job" j SET enabled = false FROM "app" a WHERE a.id = j.app_id AND a.name = 'core' AND j.key = 'delete-file-blob'`); err != nil {
		t.Fatal(err)
	}
	jobs, err := dygodata.NewJobDataFromBeginner(pool)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(pool, transactionBlobStore{}, jobs, nil)
	if err := service.Remove(ctx, file["id"].(int64)); err == nil {
		t.Fatal("Remove() error = nil, want enqueue failure")
	}
	if _, err := store.GetRecordByIdentity(ctx, "core", "file", file["id"].(int64)); err != nil {
		t.Fatalf("file metadata was not rolled back: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE "job" j SET enabled = true FROM "app" a WHERE a.id = j.app_id AND a.name = 'core' AND j.key = 'delete-file-blob'`); err != nil {
		t.Fatal(err)
	}
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bound := service.WithQueryer(transaction)
	if err := bound.Remove(ctx, file["id"].(int64)); err != nil {
		_ = transaction.Rollback(ctx)
		t.Fatalf("transaction-bound Remove() error = %v", err)
	}
	var executions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM "job_execution" WHERE idempotency_key = $1`, "file-cleanup:"+fmt.Sprint(file["id"])).Scan(&executions); err != nil {
		_ = transaction.Rollback(ctx)
		t.Fatal(err)
	}
	if executions != 0 {
		_ = transaction.Rollback(ctx)
		t.Fatalf("uncommitted cleanup executions visible outside transaction = %d", executions)
	}
	if err := transaction.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetRecordByIdentity(ctx, "core", "file", file["id"].(int64)); err != nil {
		t.Fatalf("outer action rollback lost file metadata: %v", err)
	}

	records := dygodata.NewRecordDataWithHookPolicy(pool, db.RecordMutationHooksNone)
	notifications := dygodata.NewNotificationData(records, jobs)
	if _, err := pool.Exec(ctx, `UPDATE "job" j SET enabled = false FROM "app" a WHERE a.id = j.app_id AND a.name = 'core' AND j.key = 'send-notification-email'`); err != nil {
		t.Fatal(err)
	}
	if _, err := notifications.Send(ctx, dygo.NotificationMessage{
		Recipient:      "transaction-test@example.com",
		Title:          "Transaction test",
		Message:        "This must roll back",
		IdempotencyKey: "transaction-test",
		Email:          true,
	}); err == nil {
		t.Fatal("Send() error = nil, want enqueue failure")
	}
	if _, err := store.FindRecordByIdentity(ctx, "core", "notification", db.RecordInput{
		"recipient":       json.RawMessage(`"transaction-test@example.com"`),
		"idempotency-key": json.RawMessage(`"transaction-test"`),
	}); err == nil {
		t.Fatal("notification metadata was not rolled back")
	}
	if _, err := pool.Exec(ctx, `UPDATE "job" j SET enabled = true FROM "app" a WHERE a.id = j.app_id AND a.name = 'core' AND j.key = 'send-notification-email'`); err != nil {
		t.Fatal(err)
	}
	created, err := notifications.Send(ctx, dygo.NotificationMessage{
		Recipient: "transaction-test@example.com", Title: "Transaction test", Message: "Retry succeeds",
		IdempotencyKey: "transaction-test", Email: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Created || created.Name == "" {
		t.Fatalf("retry receipt = %+v, want newly created notification", created)
	}
	duplicate, err := notifications.Send(ctx, dygo.NotificationMessage{
		Recipient: "transaction-test@example.com", Title: "Transaction test", Message: "Retry succeeds",
		IdempotencyKey: "transaction-test", Email: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Created || duplicate.Name != created.Name {
		t.Fatalf("duplicate receipt = %+v, want existing %q", duplicate, created.Name)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM "notification" WHERE idempotency_key = $1`, "transaction-test").Scan(&executions); err != nil {
		t.Fatal(err)
	}
	if executions != 1 {
		t.Fatalf("notifications = %d, want one", executions)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM "job_execution" WHERE idempotency_key = $1`, "notification-email:"+created.Name).Scan(&executions); err != nil {
		t.Fatal(err)
	}
	if executions != 1 {
		t.Fatalf("notification email executions = %d, want one", executions)
	}
}

func transactionTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DYGO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set DYGO_TEST_DATABASE_URL to run PostgreSQL regressions")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("dygo_tx_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config := admin.Config().Copy()
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SyncMetadataSchema(ctx, pool, root); err != nil {
		t.Fatal(err)
	}
	return pool
}

type transactionBlobStore struct{}

func (transactionBlobStore) Put(context.Context, string, io.Reader, int64) error { return nil }
func (transactionBlobStore) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}
func (transactionBlobStore) Remove(context.Context, string) error { return nil }
