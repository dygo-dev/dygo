package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hapyco/dygo/internal/entity/fieldtype"
	"github.com/hapyco/dygo/internal/entity/schema"
	"github.com/hapyco/dygo/pkg/dygo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresTreeIntegrityAndTraversal(t *testing.T) {
	pool, metadata := auditDatabase(t)
	entity := auditEntity("node", schema.Field{Name: "parent", Label: "Parent", Type: "link", Index: true, Options: fieldtype.Options{Entity: "node"}}, schema.Field{Name: "title", Label: "Title", Type: "text"})
	entity.Entity.Tree = &schema.Tree{ParentField: "parent", LabelField: "title"}
	metadata.Entities = append(metadata.Entities, entity)
	syncAuditMetadata(t, pool, metadata)
	ctx := context.Background()
	store := NewRecordStore(pool)
	create := func(name, parent string) int64 {
		t.Helper()
		input := RecordInput{"name": []byte(fmt.Sprintf("%q", name)), "title": []byte(fmt.Sprintf("%q", name))}
		if parent != "" {
			input["parent"] = []byte(fmt.Sprintf("%q", parent))
		}
		record, err := store.CreateRecordByIdentity(ctx, "audit", "node", input)
		if err != nil {
			t.Fatal(err)
		}
		return record["id"].(int64)
	}
	root := create("root", "")
	child := create("child", "root")
	leaf := create("leaf", "child")
	other := create("other", "")
	meta, err := NewMetadataReader(pool).GetEntityMetaByIdentity(ctx, "audit", "node")
	if err != nil || meta.Tree == nil || meta.Tree.LabelField != "title" {
		t.Fatalf("tree roundtrip: %+v %v", meta.Tree, err)
	}
	names := func(result dygo.TreeResult) []string {
		out := []string{}
		for _, node := range result.Nodes {
			out = append(out, node.Record["name"].(string))
		}
		return out
	}
	t.Run("forest and path order", func(t *testing.T) {
		for _, tc := range []struct {
			operation string
			id        int64
			want      []string
		}{
			{"roots", 0, []string{"other", "root"}}, {"children", root, []string{"child"}}, {"descendants", root, []string{"child", "leaf"}}, {"ancestors", leaf, []string{"root", "child"}}, {"path", leaf, []string{"root", "child", "leaf"}},
		} {
			result, err := store.TreeRecords(ctx, "audit", "node", tc.operation, tc.id, RecordListParams{}, 0)
			if err != nil || !reflect.DeepEqual(names(result), tc.want) {
				t.Fatalf("%s: %v %v", tc.operation, names(result), err)
			}
		}
	})
	t.Run("search context sort and pre-pagination exclusion", func(t *testing.T) {
		result, err := store.TreeRecords(ctx, "audit", "node", "search", 0, RecordListParams{Limit: 1, Filters: []RecordFilter{{Field: "name", Operator: "eq", Value: "leaf"}}, Sort: []RecordSort{{Field: "name", Desc: true}}}, 0)
		if err != nil || len(result.Nodes) != 1 || len(result.Context) != 3 {
			t.Fatalf("search: %+v %v", result, err)
		}
		if result.Context[0]["name"] != "root" || result.Context[1]["name"] != "leaf" || result.Context[2]["name"] != "child" {
			t.Fatal("context does not preserve requested sort")
		}
		result, err = store.TreeRecords(ctx, "audit", "node", "search", 0, RecordListParams{Limit: 1}, root)
		if err != nil || !reflect.DeepEqual(names(result), []string{"other"}) {
			t.Fatalf("exclude: %+v %v", result, err)
		}
	})
	t.Run("row and parent field privacy", func(t *testing.T) {
		read := store.WithScope(RecordScope{Where: `"_dygo_record".name <> $1`, Args: []any{"child"}})
		result, err := read.TreeRecords(ctx, "audit", "node", "search", 0, RecordListParams{Filters: []RecordFilter{{Field: "name", Operator: "eq", Value: "leaf"}}}, 0)
		if err != nil || len(result.Nodes) != 1 || !result.Nodes[0].PathUnavailable || len(result.Context) != 0 || result.Nodes[0].Parent != "" {
			t.Fatalf("private path: %+v %v", result, err)
		}
		if _, ok := result.Nodes[0].Record["parent"]; ok {
			t.Fatal("hidden parent name exposed")
		}
		wire, err := json.Marshal(struct {
			Data    []dygo.TreeNode `json:"data"`
			Context []dygo.Record   `json:"context"`
		}{result.Nodes, result.Context})
		if err != nil || strings.Contains(string(wire), `"child"`) {
			t.Fatalf("hidden ancestor in response: %s %v", wire, err)
		}
		_, err = read.TreeRecords(ctx, "audit", "node", "path", leaf, RecordListParams{}, 0)
		assertRecordErrorCode(t, err, RecordErrorPermissionDenied)
		_, err = read.TreeRecords(ctx, "audit", "node", "search", 0, RecordListParams{}, child)
		assertRecordErrorCode(t, err, RecordErrorNotFound)
		roots, err := read.TreeRecords(ctx, "audit", "node", "roots", 0, RecordListParams{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range roots.Nodes {
			if n.Record["name"] == "root" && n.HasChildren {
				t.Fatal("hidden child indicator")
			}
		}
		fieldRead := store.WithScope(RecordScope{Where: "TRUE", FieldRead: map[string]string{"parent": "FALSE"}})
		_, err = fieldRead.TreeRecords(ctx, "audit", "node", "path", leaf, RecordListParams{}, 0)
		assertRecordErrorCode(t, err, RecordErrorPermissionDenied)
		mutate := store.WithScope(RecordScope{Where: "TRUE"}).WithTreeReadScope(RecordScope{Where: `"_dygo_record".name <> $1`, Args: []any{"child"}})
		_, err = mutate.UpdateRecordByIdentity(ctx, "audit", "node", other, RecordInput{"parent": []byte(`"child"`)})
		assertRecordErrorCode(t, err, RecordErrorPermissionDenied)
	})
	t.Run("cycles missing parents and restricted delete", func(t *testing.T) {
		for _, parent := range []string{"root", "leaf", "missing"} {
			_, err := store.UpdateRecordByIdentity(ctx, "audit", "node", root, RecordInput{"parent": []byte(fmt.Sprintf("%q", parent))})
			assertRecordErrorCode(t, err, RecordErrorValidation)
		}
		err := store.DeleteRecordByIdentity(ctx, "audit", "node", root)
		assertRecordErrorCode(t, err, RecordErrorConstraintViolation)
	})
	t.Run("final hook input and rollback", func(t *testing.T) {
		hooks := DefaultRecordHookRegistry()
		if err := hooks.RegisterEntity("audit", "node", RecordBeforeUpdate, "cycle", func(ctx context.Context, hook RecordHookContext) error {
			hook.Input["parent"] = json.RawMessage(`"leaf"`)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		_, err := NewRecordStoreWithHooks(pool, hooks).UpdateRecordByIdentity(ctx, "audit", "node", root, RecordInput{"title": []byte(`"changed"`)})
		assertRecordErrorCode(t, err, RecordErrorValidation)
		record, err := store.GetRecordByIdentity(ctx, "audit", "node", root)
		if err != nil || record["title"] != "root" {
			t.Fatal("failed Hook mutation persisted")
		}
	})
	t.Run("concurrent opposite moves", func(t *testing.T) {
		a := create("a", "")
		b := create("b", "")
		start := make(chan struct{})
		out := make(chan error, 2)
		for _, pair := range []struct {
			id     int64
			parent string
		}{{a, "b"}, {b, "a"}} {
			go func(id int64, parent string) {
				<-start
				_, err := store.UpdateRecordByIdentity(ctx, "audit", "node", id, RecordInput{"parent": []byte(fmt.Sprintf("%q", parent))})
				out <- err
			}(pair.id, pair.parent)
		}
		close(start)
		first, second := <-out, <-out
		if (first == nil) == (second == nil) {
			t.Fatalf("opposite moves: %v / %v", first, second)
		}
		if first != nil {
			assertRecordErrorCode(t, first, RecordErrorValidation)
		}
		if second != nil {
			assertRecordErrorCode(t, second, RecordErrorValidation)
		}
	})
	t.Run("branch query count does not grow with page size", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			create(fmt.Sprintf("wide-%02d", i), "other")
		}
		counter := &treeQueryCounter{}
		config := pool.Config().Copy()
		config.ConnConfig.Tracer = counter
		traced, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			t.Fatal(err)
		}
		defer traced.Close()
		tracedStore := NewRecordStore(traced)
		var counts []int64
		for _, limit := range []int{1, 20} {
			counter.count.Store(0)
			result, err := tracedStore.TreeRecords(ctx, "audit", "node", "children", other, RecordListParams{Limit: limit}, 0)
			if err != nil || len(result.Nodes) != limit {
				t.Fatalf("branch: %+v %v", result, err)
			}
			counts = append(counts, counter.count.Load())
		}
		if counts[0] != counts[1] {
			t.Fatalf("branch queries grow with rows: %v", counts)
		}
		t.Logf("branch reads: %d statements for either 1 or 20 children (includes metadata and transaction)", counts[0])
	})
	for _, mode := range []string{"create", "reparent"} {
		t.Run("concurrent child "+mode+" and delete", func(t *testing.T) {
			parentName := "race-parent-" + mode
			id := create(parentName, "")
			var movingID int64
			if mode == "reparent" {
				movingID = create("moving-child", "")
			}
			var wg sync.WaitGroup
			wg.Add(2)
			start := make(chan struct{})
			errs := make([]error, 2)
			go func() {
				defer wg.Done()
				<-start
				input := RecordInput{"parent": []byte(fmt.Sprintf("%q", parentName))}
				if mode == "create" {
					input["name"] = []byte(`"race-child"`)
					_, errs[0] = store.CreateRecordByIdentity(ctx, "audit", "node", input)
				} else {
					_, errs[0] = store.UpdateRecordByIdentity(ctx, "audit", "node", movingID, input)
				}
			}()
			go func() { defer wg.Done(); <-start; errs[1] = store.DeleteRecordByIdentity(ctx, "audit", "node", id) }()
			close(start)
			wg.Wait()
			if (errs[0] == nil) == (errs[1] == nil) {
				t.Fatalf("create/delete race: %v", errs)
			}
		})
	}
	t.Run("existing invalid hierarchy blocks activation", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE audit_node SET parent_id=$1 WHERE id=$2`, leaf, root); err != nil {
			t.Fatal(err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		err = validateTreeData(ctx, tx, metadata.Entities)
		if err == nil {
			t.Fatal("activated cyclic data")
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		var e RecordError
		_, err = store.TreeRecords(ctx, "audit", "node", "path", leaf, RecordListParams{}, 0)
		if !errors.As(err, &e) || e.Code != RecordErrorConstraintViolation {
			t.Fatalf("cycle read: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE entity SET tree=NULL WHERE name='audit.node'`); err != nil {
			t.Fatal(err)
		}
		live, err := InspectLiveSchema(ctx, pool)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := BuildMetadataSchemaPlan(metadata.Entities, live)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := applyMetadataSchemaPlanAndRecords(ctx, pool, plan, metadata); err == nil {
			t.Fatal("published invalid tree metadata")
		}
		var inactive bool
		if err := pool.QueryRow(ctx, `SELECT tree IS NULL FROM entity WHERE name='audit.node'`).Scan(&inactive); err != nil || !inactive {
			t.Fatalf("failed activation was not atomic: %t %v", inactive, err)
		}
	})
}

type treeQueryCounter struct{ count atomic.Int64 }

func (c *treeQueryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.count.Add(1)
	return ctx
}
func (*treeQueryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
