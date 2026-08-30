package db

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/entity/schema"
)

func TestGenerateRecordNameScopesSeriesByAppAndEntity(t *testing.T) {
	queryer := &fakeRecordQueryer{row: newFakeRow(int64(1))}
	store := NewRecordStore(queryer)
	layout := recordLayout{
		EntityID: 20,
		AppName:  "crm",
		Entity:   "invoice",
		Naming: schema.Naming{
			Strategy: schema.NamingStrategySeries,
			Pattern:  "INV-{####}",
		},
	}

	name, err := store.generateRecordName(context.Background(), layout, nil)
	if err != nil {
		t.Fatalf("generateRecordName() error = %v, want nil", err)
	}
	if name != "INV-0001" {
		t.Fatalf("generateRecordName() = %q, want INV-0001", name)
	}
	wantArgs := []any{"crm/invoice:INV-{####}", int64(20), "crm/invoice:INV-{####}", "INV-{####}", "invoice:INV-{####}"}
	if len(queryer.rowArgs) != 1 || !reflect.DeepEqual(queryer.rowArgs[0], wantArgs) {
		t.Fatalf("naming series args = %#v, want %#v", queryer.rowArgs, wantArgs)
	}
	if len(queryer.rowSQL) != 1 || !strings.Contains(queryer.rowSQL[0], `legacy."key" = $5`) {
		t.Fatalf("naming series queries = %q, want legacy counter adoption", queryer.rowSQL)
	}
}

func TestLegacySeriesKeyPreservesExistingCounterIdentity(t *testing.T) {
	if got := legacySeriesKey("crm/invoice:INV-2026-{####}", "crm", "invoice"); got != "invoice:INV-2026-{####}" {
		t.Fatalf("legacySeriesKey() = %q, want legacy Entity key", got)
	}
}
