package files

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hapyco/dygo/pkg/dygo"
)

func TestCleanupJobRemovesPrivateBlob(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-file")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	registry := &testJobRegistry{}
	if err := CleanupJobRegistrar()(registry); err != nil {
		t.Fatalf("register cleanup job: %v", err)
	}
	payload, _ := json.Marshal(cleanupPayload{Path: path})
	if err := registry.fn(context.Background(), dygo.JobExecution{Payload: payload}); err != nil {
		t.Fatalf("cleanup job: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("blob stat error = %v, want not exists", err)
	}
}

type testJobRegistry struct{ fn dygo.JobFunc }

func (r *testJobRegistry) RegisterJob(_ string, _ string, fn dygo.JobFunc) error {
	r.fn = fn
	return nil
}
