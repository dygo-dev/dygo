package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/app/manifest"
	"github.com/hapyco/dygo/internal/shape"
)

func TestCatalogDiscoversCanonicalPageBundle(t *testing.T) {
	root := t.TempDir()
	pageDir := filepath.Join(root, "pages", "home")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(pageDir, shape.PageMetadataFileName("home"))
	if err := os.WriteFile(path, []byte("label: Home\nroute:\n  path: /\nrenderer: entity-index\noptions: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := New([]manifest.LoadedApp{{Dir: root, Manifest: manifest.Manifest{Name: "studio"}}}).Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded %d pages, want 1", len(loaded))
	}
	page := loaded[0]
	if page.Page.Key != "home" || page.Page.Name != "studio.home" || page.Page.App != "studio" {
		t.Fatalf("loaded identity = %+v, want studio.home/home/studio", page.Page)
	}
	if page.RoutePath() != "/" {
		t.Fatalf("RoutePath() = %q, want /", page.RoutePath())
	}
}

func TestCatalogRejectsLegacyPageFilename(t *testing.T) {
	root := t.TempDir()
	pageDir := filepath.Join(root, "pages", "home")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pageDir, "page.yml"), []byte("label: Home\nroute:\n  path: /\nrenderer: entity-index\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := New([]manifest.LoadedApp{{Dir: root, Manifest: manifest.Manifest{Name: "studio"}}}).Discover()
	if err == nil || !strings.Contains(err.Error(), "home.page.yml") {
		t.Fatalf("Discover() error = %v, want canonical filename error", err)
	}
}

func TestDecodeValidatesPagePathAndRenderer(t *testing.T) {
	for name, input := range map[string]string{
		"nested path":      "label: Home\nroute:\n  path: /admin/home\nrenderer: entity-index\n",
		"unknown renderer": "label: Home\nroute:\n  path: /\nrenderer: custom\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(input)); err == nil {
				t.Fatal("Decode() error = nil, want validation error")
			}
		})
	}
}
