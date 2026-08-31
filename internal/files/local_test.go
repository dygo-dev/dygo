package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBlobStoreKeepsFilesPrivateAndRemovesThem(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalBlobStore(filepath.Join(root, "files"))
	if err != nil {
		t.Fatalf("NewLocalBlobStore() error = %v", err)
	}
	if err := store.Put(context.Background(), "secret", strings.NewReader("private"), int64(len("private"))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	info, err := os.Stat(store.Path("secret"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 600", info.Mode().Perm())
	}
	if _, err := store.Open(context.Background(), "../secret"); err == nil {
		t.Fatal("Open() accepted a path traversal key")
	}
	if err := store.Remove(context.Background(), "secret"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(store.Path("secret")); !os.IsNotExist(err) {
		t.Fatalf("removed blob stat error = %v, want not exists", err)
	}
}
