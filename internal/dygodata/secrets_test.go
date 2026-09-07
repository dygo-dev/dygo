package dygodata

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/pkg/dygo"
)

type secretAudit struct {
	entries []dygo.LogEntry
	err     error
}

func (s *secretAudit) WriteLog(_ context.Context, entry dygo.LogEntry) error {
	s.entries = append(s.entries, entry)
	return s.err
}
func TestSecretDecryptRequiresExplicitSystem(t *testing.T) {
	for _, view := range []dygo.RecordData{NewRecordData(nil, nil), NewRecordData(nil, nil).AsActor(dygo.Actor{UserID: 1, Administrator: true}), NewRecordData(nil, nil).AsSystem(" "), NewRecordData(nil, nil).AsSystem("job").AsActor(dygo.Actor{Administrator: true})} {
		audit := &secretAudit{}
		ctx := dygo.WithLogWriter(context.Background(), audit)
		value, err := view.DecryptSecret(ctx, "app", "account", 1, "token")
		var denied db.RecordError
		if value != "" || !errors.As(err, &denied) || denied.Code != db.RecordErrorPermissionDenied {
			t.Fatalf("access was not denied: %v", err)
		}
		if len(audit.entries) != 1 || audit.entries[0].Metadata["outcome"] != "denied-or-failed" {
			t.Fatal("denial not audited")
		}
		raw, _ := json.Marshal(audit.entries)
		if strings.Contains(string(raw), "ciphertext") {
			t.Fatal("audit exposed ciphertext")
		}
	}
}
