package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/entity/catalog"
	"github.com/hapyco/dygo/internal/entity/schema"
	"github.com/hapyco/dygo/internal/patches"
)

func systemPatch(t *testing.T, app, id, operations string) patches.LoadedPatch {
	t.Helper()
	patch, err := patches.Decode([]byte(fmt.Sprintf("kind: patch\nversion: 1\nid: %s\nphase: post-sync\ndescription: Test\noperations:\n%s", id, operations)))
	if err != nil {
		t.Fatal(err)
	}
	return patches.LoadedPatch{AppName: app, Path: "apps/" + app + "/patches/" + id + ".yml", Checksum: id, Patch: patch}
}

func TestSystemRecordPatchPlan(t *testing.T) {
	entity := testEntity("sales", "state", schema.Field{Name: "code", Type: "text", Unique: true})
	entity.Entity.IsSystem = true
	for _, tc := range []struct{ name, operation, want string }{
		{"create", "system-record-create\n    values: {code: private-secret}", ""},
		{"update", "system-record-update\n    id: 42\n    values: {code: private-secret}", ""},
		{"delete", "system-record-delete\n    id: 42", ""},
		{"upsert", "system-record-upsert\n    match: {code: private-secret}\n    values: {}", ""},
		{"bad match", "system-record-upsert\n    match: {missing: private-secret}\n    values: {}", "does not exist"},
		{"conflict", "system-record-upsert\n    match: {code: private-secret}\n    values: {code: different}", "conflict"},
		{"missing values", "system-record-create", "values"},
		{"invalid id", "system-record-delete\n    id: 0", "positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patch := systemPatch(t, "sales", "test", "  - type: "+tc.operation+"\n    entity: state\n    reason: Repair state\n")
			plan, err := BuildPatchOperationPlan([]patches.LoadedPatch{patch}, []catalog.LoadedEntity{entity}, LiveSchema{})
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(plan)
			if strings.Contains(string(encoded), "private-secret") {
				t.Fatal("plan exposed value")
			}
			if plan.Operations[0].record == nil || plan.Operations[0].SQL != "" {
				t.Fatal("missing structured operation")
			}
			patch.Patch.Phase = patches.PhasePreSync
			if _, err := BuildPatchOperationPlan([]patches.LoadedPatch{patch}, []catalog.LoadedEntity{entity}, LiveSchema{}); err == nil {
				t.Fatal("accepted pre-sync")
			}
			patch.Patch.Phase = patches.PhasePostSync
			patch.AppName = "foreign"
			if _, err := BuildPatchOperationPlan([]patches.LoadedPatch{patch}, []catalog.LoadedEntity{entity}, LiveSchema{}); err == nil {
				t.Fatal("accepted foreign ownership")
			}
		})
	}
}

func TestPostgresSystemRecordPatchAtomicityAndLedger(t *testing.T) {
	pool, metadata := auditDatabase(t)
	entity := auditEntity("patch-state", schema.Field{Name: "code", Label: "Code", Type: "text", Unique: true}, schema.Field{Name: "status", Label: "Status", Type: "text"})
	entity.AppName = "core"
	entity.Entity.IsSystem = true
	metadata.Entities = append(metadata.Entities, entity)
	syncAuditMetadata(t, pool, metadata)
	ctx := context.Background()
	live, err := InspectLiveSchema(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	patch := systemPatch(t, "core", "system-patch", `  - type: system-record-create
    entity: patch-state
    reason: Initialize state
    values: {name: primary, code: primary}
  - type: system-record-upsert
    entity: patch-state
    reason: Repair state
    match: {code: primary}
    values: {status: repaired}
`)
	plan, err := BuildPatchPlan([]patches.LoadedPatch{patch}, metadata.Entities, live, nil, patches.PhasePostSync)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM patch_state`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("dry run wrote data: %d %v", count, err)
	}
	result, err := ApplyPatchPlan(ctx, pool, plan, "", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != 1 {
		t.Fatal("missing applied patch")
	}
	rerun, err := BuildPatchPlan([]patches.LoadedPatch{patch}, metadata.Entities, live, result.Applied, patches.PhasePostSync)
	if err != nil || len(rerun.Pending) != 0 {
		t.Fatalf("repeat plan: %+v %v", rerun, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM patch_state WHERE status='repaired'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("upsert state: %d %v", count, err)
	}
	var activityBefore int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM activity`).Scan(&activityBefore); err != nil {
		t.Fatal(err)
	}
	failed := systemPatch(t, "core", "failed-system-patch", `  - type: system-record-create
    entity: patch-state
    reason: Must roll back
    values: {name: rollback, code: rollback}
  - type: sql
    name: fail
    reason: Exercise rollback
    statement: SELECT 1/0
`)
	failedPlan, err := BuildPatchPlan([]patches.LoadedPatch{failed}, metadata.Entities, live, result.Applied, patches.PhasePostSync)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPatchPlan(ctx, pool, failedPlan, "", "dev"); err == nil {
		t.Fatal("expected failure")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM patch_state WHERE code='rollback'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("record rollback: %d %v", count, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM activity`).Scan(&count); err != nil || count != activityBefore {
		t.Fatalf("Activity rollback: %d %v", count, err)
	}
	runs, err := NewPatchLedger(pool).ListPatchRuns(ctx)
	if err != nil || len(runs) != 1 {
		t.Fatalf("ledger rollback: %+v %v", runs, err)
	}
	var id int64
	if err := pool.QueryRow(ctx, `SELECT id FROM patch_state WHERE code='primary'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	cleanup := systemPatch(t, "core", "cleanup-system-patch", fmt.Sprintf(`  - type: system-record-update
    entity: patch-state
    reason: Finalize state
    id: %d
    values: {status: finalized}
  - type: system-record-delete
    entity: patch-state
    reason: Retire state
    id: %d
`, id, id))
	cleanupPlan, err := BuildPatchPlan([]patches.LoadedPatch{cleanup}, metadata.Entities, live, runs, patches.PhasePostSync)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPatchPlan(ctx, pool, cleanupPlan, "", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM patch_state`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("delete count=%d error=%v", count, err)
	}

}
