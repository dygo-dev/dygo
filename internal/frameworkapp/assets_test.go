package frameworkapp

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestEmbeddedCoreMatchesFrameworkSource(t *testing.T) {
	embedded, err := EmbeddedCoreSource()
	if err != nil {
		t.Fatalf("EmbeddedCoreSource() error = %v, want nil", err)
	}
	want := fileContents(t, os.DirFS(FrameworkCorePath(filepath.Join("..", ".."))))
	got := fileContents(t, embedded.FS)
	for name, wantContent := range want {
		if gotContent, ok := got[name]; !ok || gotContent != wantContent {
			t.Fatalf("bundled Core App asset %q is stale; got %q, want %q; run go generate ./internal/frameworkapp", name, gotContent, wantContent)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Fatalf("bundled Core App has retired asset %q; run go generate ./internal/frameworkapp", name)
		}
	}
}

func TestEmbeddedCoreDoesNotShipAdministratorFixture(t *testing.T) {
	embedded, err := EmbeddedCoreSource()
	if err != nil {
		t.Fatalf("EmbeddedCoreSource() error = %v, want nil", err)
	}
	if _, err := fs.Stat(embedded.FS, "entities/user/fixtures.yml"); !os.IsNotExist(err) {
		t.Fatalf("Core user fixture stat error = %v, want no shipped Administrator fixture", err)
	}
}

func TestInstallCoreReplacesExistingCache(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, filepath.FromSlash(CoreProjectPath))
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", target, err)
	}
	if err := os.WriteFile(filepath.Join(target, "stale.yml"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile(stale.yml) error = %v", err)
	}

	name, err := InstallCore(root, Source{Name: "test Core App", FS: fstest.MapFS{
		"app.yml":                       {Data: []byte("name: core\n")},
		"entities/user/user.entity.yml": {Data: []byte("label: User\n")},
	}})
	if err != nil {
		t.Fatalf("InstallCore() error = %v, want nil", err)
	}
	if name != "test Core App" {
		t.Fatalf("InstallCore() source = %q, want test Core App", name)
	}
	if _, err := os.Stat(filepath.Join(target, "entities", "user", "user.entity.yml")); err != nil {
		t.Fatalf("Stat(installed user metadata) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "stale.yml")); !os.IsNotExist(err) {
		t.Fatalf("Stat(stale.yml) error = %v, want missing", err)
	}
}

func fileContents(t *testing.T, source fs.FS) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(name) == ".DS_Store" {
			return nil
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(name)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}
	return files
}
