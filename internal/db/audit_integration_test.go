package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hapyco/dygo/internal/app/manifest"
	"github.com/hapyco/dygo/internal/entity/catalog"
	"github.com/hapyco/dygo/internal/entity/fieldtype"
	"github.com/hapyco/dygo/internal/entity/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Each integration test owns a database. No configured app database is modified.
func auditDatabase(t *testing.T) (*pgxpool.Pool, metadataCatalog) {
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
	name := fmt.Sprintf("dygo_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+quoteIdent(name)); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config := admin.Config().Copy()
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+quoteIdent(name)); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loadMetadataCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata.Apps = append(metadata.Apps, manifest.LoadedApp{Manifest: manifest.Manifest{Name: "audit", Label: "Audit", Version: "0.1.0"}})
	return pool, metadata
}

func syncAuditMetadata(t *testing.T, pool *pgxpool.Pool, metadata metadataCatalog) {
	t.Helper()
	ctx := context.Background()
	live, err := InspectLiveSchema(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildMetadataSchemaPlan(metadata.Entities, live)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := applyMetadataSchemaPlanAndRecords(ctx, pool, plan, metadata); err != nil {
		t.Fatal(err)
	}
}

func auditEntity(name string, fields ...schema.Field) catalog.LoadedEntity {
	return catalog.LoadedEntity{AppName: "audit", Entity: schema.Entity{Name: name, Label: name, Naming: schema.Naming{Strategy: schema.NamingStrategyManual}, Fields: fields}}
}

func TestPostgresCollectionReadScopeAndSystemInsertParity(t *testing.T) {
	pool, metadata := auditDatabase(t)
	parent := auditEntity("parent", schema.Field{Name: "contacts", Label: "Contacts", Type: "collection", Options: fieldtype.Options{Entity: "contact"}}, schema.Field{Name: "note", Label: "Note", Type: "text"})
	child := auditEntity("contact", schema.Field{Name: "email", Label: "Email", Type: "text", Required: true})
	child.Entity.IsCollection = true
	metadata.Entities = append(metadata.Entities, parent, child)
	syncAuditMetadata(t, pool, metadata)
	ctx := context.Background()
	store := NewRecordStoreWithHookPolicy(pool, RecordMutationHooksNone)
	for _, returning := range []bool{false, true} {
		input := recordInput(map[string]string{"name": fmt.Sprintf(`"parent-%t"`, returning), "contacts": `[{"name":"contact","email":"private@example.com"}]`})
		writer := store.SystemWriter()
		if returning {
			if _, err := writer.InsertReturningByIdentity(ctx, "audit", "parent", input, SystemMutationSilent); err != nil {
				t.Fatal(err)
			}
		} else {
			// Collection names are Entity-wide, even though rows belong to a parent.
			input["contacts"] = []byte(`[{"name":"first-contact","email":"private@example.com"}]`)
			if err := writer.InsertByIdentity(ctx, "audit", "parent", input, SystemMutationSilent); err != nil {
				t.Fatal(err)
			}
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_contact`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("system insert children: count=%d error=%v", count, err)
	}
	record, err := store.FindRecordByIdentity(ctx, "audit", "parent", recordInput(map[string]string{"name": `"parent-false"`}))
	if err != nil {
		t.Fatal(err)
	}
	id := record["id"].(int64)
	for _, allowed := range []bool{false, true} {
		scoped := store.WithScope(RecordScope{Where: "($1::boolean OR $2::boolean)", Args: []any{true, allowed}, FieldRead: map[string]string{"contacts": "$2::boolean"}})
		got, err := scoped.GetRecordByIdentity(ctx, "audit", "parent", id)
		if err != nil {
			t.Fatal(err)
		}
		_, exists := got["contacts"]
		if exists != allowed {
			t.Fatalf("collection visible=%t, policy=%t", exists, allowed)
		}
		got, err = scoped.UpdateRecordByIdentity(ctx, "audit", "parent", id, recordInput(map[string]string{"note": `"updated"`}))
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := got["contacts"]; exists != allowed {
			t.Fatalf("update collection visible=%t, policy=%t", exists, allowed)
		}
	}
	deniedWrite := store.WithScope(RecordScope{Where: "TRUE", FieldWrite: map[string]string{"note": "FALSE"}})
	if _, err := deniedWrite.CreateRecordByIdentity(ctx, "audit", "parent", recordInput(map[string]string{"name": `"denied"`, "note": `"private"`})); err == nil {
		t.Fatal("unconditional row access bypassed field-write denial on create")
	}
	// A denied collection is never queried, even if its physical table is unavailable.
	if _, err := pool.Exec(ctx, `DROP TABLE audit_contact`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.WithScope(RecordScope{Where: "TRUE", FieldRead: map[string]string{"contacts": "FALSE"}}).GetRecordByIdentity(ctx, "audit", "parent", id); err != nil {
		t.Fatalf("denied child storage was accessed: %v", err)
	}
}

func TestPostgresPruneRetiresMetadataAndPreservesRegistryIDs(t *testing.T) {
	pool, metadata := auditDatabase(t)
	entity := auditEntity("example", schema.Field{Name: "keep", Label: "Keep", Type: "text"}, schema.Field{Name: "remove", Label: "Remove", Type: "text"})
	entity.Entity.Indexes = []schema.Index{{Name: "by-remove", Fields: []string{"remove"}}}
	entity.Entity.Constraints = []schema.Constraint{{Name: "unique-remove", Type: "unique", Fields: []string{"remove"}}}
	metadata.Entities = append(metadata.Entities, entity)
	syncAuditMetadata(t, pool, metadata)
	ctx := context.Background()
	reader := NewMetadataReader(pool)
	before, err := reader.GetEntityMetaByIdentity(ctx, "audit", "example")
	if err != nil {
		t.Fatal(err)
	}
	entity.Entity.Fields = entity.Entity.Fields[:1]
	entity.Entity.Indexes = nil
	entity.Entity.Constraints = nil
	metadata.Entities[len(metadata.Entities)-1] = entity
	prune := func() {
		t.Helper()
		live, err := InspectLiveSchema(ctx, pool)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := BuildSchemaPrunePlan(metadata.Entities, live)
		if err != nil {
			t.Fatal(err)
		}
		plan.metadata = &metadata
		if _, err := ApplySchemaPrunePlan(ctx, pool, plan); err != nil {
			t.Fatal(err)
		}
	}
	prune()
	after, err := reader.GetEntityMetaByIdentity(ctx, "audit", "example")
	if err != nil {
		t.Fatal(err)
	}
	if after.ID != before.ID || len(after.Fields) != 1 || len(after.Indexes) != 0 || len(after.Constraints) != 0 {
		t.Fatalf("stale registry after prune: %+v", after)
	}
	store := NewRecordStoreWithHookPolicy(pool, RecordMutationHooksNone)
	if _, err := store.CreateRecordByIdentity(ctx, "audit", "example", recordInput(map[string]string{"name": `"survivor"`, "keep": `"value"`})); err != nil {
		t.Fatal(err)
	}
	metadata.Entities = metadata.Entities[:len(metadata.Entities)-1]
	prune()
	if _, err := reader.GetEntityMetaByIdentity(ctx, "audit", "example"); !IsMetadataNotFound(err) {
		t.Fatalf("removed Entity error=%v", err)
	}
	var retired bool
	if err := pool.QueryRow(ctx, `SELECT retired FROM entity WHERE id = $1`, before.ID).Scan(&retired); err != nil || !retired {
		t.Fatalf("registry identity lost: retired=%t error=%v", retired, err)
	}
	metadata.Entities = append(metadata.Entities, entity)
	syncAuditMetadata(t, pool, metadata)
	restored, err := reader.GetEntityMetaByIdentity(ctx, "audit", "example")
	if err != nil || restored.ID != before.ID {
		t.Fatalf("restore identity=%d expected=%d error=%v", restored.ID, before.ID, err)
	}
}

func TestPostgresCollectionFieldInvariantsRollbackParent(t *testing.T) {
	pool, metadata := auditDatabase(t)
	account := auditEntity("account", schema.Field{Name: "status", Label: "Status", Type: "text"})
	child := auditEntity("line",
		schema.Field{Name: "account", Label: "Account", Type: "link", Options: fieldtype.Options{Entity: "account", Filters: []fieldtype.LinkFilter{{Field: "status", Operator: "eq", Value: "Active"}}}},
		schema.Field{Name: "status", Label: "Status", Type: "text", Required: true, Fetch: &schema.Fetch{From: "account.status"}},
	)
	child.Entity.IsCollection = true
	parent := auditEntity("order", schema.Field{Name: "title", Label: "Title", Type: "text"}, schema.Field{Name: "lines", Label: "Lines", Type: "collection", Options: fieldtype.Options{Entity: "line"}})
	metadata.Entities = append(metadata.Entities, account, parent, child)
	syncAuditMetadata(t, pool, metadata)
	ctx := context.Background()
	store := NewRecordStoreWithHookPolicy(pool, RecordMutationHooksNone)
	for _, status := range []string{"Active", "Disabled"} {
		if _, err := store.CreateRecordByIdentity(ctx, "audit", "account", recordInput(map[string]string{"name": fmt.Sprintf(`%q`, status), "status": fmt.Sprintf(`%q`, status)})); err != nil {
			t.Fatal(err)
		}
	}
	record, err := store.CreateRecordByIdentity(ctx, "audit", "order", recordInput(map[string]string{"name": `"order"`, "title": `"before"`, "lines": `[{"account":"Active","status":null}]`}))
	if err != nil {
		t.Fatal(err)
	}
	rows := record["lines"].([]Record)
	if rows[0]["status"] != "Active" {
		t.Fatalf("fetched value was not authoritative: %+v", rows[0])
	}
	id := record["id"].(int64)
	rowID := rows[0]["id"].(int64)
	// Retain omitted link inputs from the existing child when applying fetch/filter rules.
	if _, err := store.UpdateRecordByIdentity(ctx, "audit", "order", id, recordInput(map[string]string{"lines": fmt.Sprintf(`[{"id":%d}]`, rowID)})); err != nil {
		t.Fatal(err)
	}
	_, err = store.UpdateRecordByIdentity(ctx, "audit", "order", id, recordInput(map[string]string{"title": `"must rollback"`, "lines": fmt.Sprintf(`[{"id":%d,"account":"Disabled"}]`, rowID)}))
	if err == nil {
		t.Fatal("filtered child link was accepted")
	}
	after, err := store.GetRecordByIdentity(ctx, "audit", "order", id)
	if err != nil {
		t.Fatal(err)
	}
	if after["title"] != "before" || after["lines"].([]Record)[0]["account"] != "Active" {
		t.Fatalf("partial mutation survived: %+v", after)
	}
	// A failed new child must also roll back the non-returning system create path.
	err = store.SystemWriter().InsertByIdentity(ctx, "audit", "order", recordInput(map[string]string{"name": `"invalid"`, "lines": `[{"account":"Disabled"}]`}), SystemMutationSilent)
	if err == nil {
		t.Fatal("system insert accepted invalid child")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_order WHERE name = 'invalid'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial system create: count=%d error=%v", count, err)
	}
}
