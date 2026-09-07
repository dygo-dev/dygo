// Package frameworkapp installs framework-managed Apps into generated projects.
package frameworkapp

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hapyco/dygo/internal/fsutil"
	"github.com/hapyco/dygo/internal/shape"
)

const CoreProjectPath = shape.LocalCoreAppDir

//go:embed all:bundled/core
var bundled embed.FS

// Source describes one framework App asset source.
type Source struct {
	Name string
	FS   fs.FS
}

// FrameworkCorePath returns the Core App path inside a framework checkout.
func FrameworkCorePath(root string) string {
	return filepath.Join(root, "apps", "core")
}

// SourceFromDir returns a source when dir contains a Core App manifest.
func SourceFromDir(name string, dir string) (Source, bool, error) {
	if strings.TrimSpace(dir) == "" {
		return Source{}, false, nil
	}
	source := Source{Name: name, FS: os.DirFS(dir)}
	ok, err := HasManifest(source.FS)
	if err != nil {
		return Source{}, false, fmt.Errorf("check %s: %w", name, err)
	}
	return source, ok, nil
}

// EmbeddedCoreSource returns the Core App bundled into the dygo binary.
func EmbeddedCoreSource() (Source, error) {
	fsys, err := fs.Sub(bundled, "bundled/core")
	if err != nil {
		return Source{}, fmt.Errorf("open bundled Core App: %w", err)
	}
	if ok, err := HasManifest(fsys); err != nil {
		return Source{}, fmt.Errorf("check bundled Core App: %w", err)
	} else if !ok {
		return Source{}, fmt.Errorf("bundled Core App is missing app.yml")
	}
	return Source{Name: "bundled Core App", FS: fsys}, nil
}

// HasManifest reports whether fsys contains an App manifest.
func HasManifest(fsys fs.FS) (bool, error) {
	return fsutil.HasFile(fsys, "app.yml")
}

// InstallCore installs the first valid Core source into a generated project.
func InstallCore(root string, sources ...Source) (string, error) {
	for _, source := range sources {
		if source.FS == nil {
			continue
		}
		ok, err := HasManifest(source.FS)
		if err != nil {
			return "", fmt.Errorf("check %s: %w", source.Name, err)
		}
		if !ok {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(CoreProjectPath))
		if err := fsutil.ReplaceDir(target, ".core-app-*", "Core App", func(temp string) error {
			if err := fsutil.CopyFS(source.FS, temp, "Core App asset"); err != nil {
				return err
			}
			if ok, err := HasManifest(os.DirFS(temp)); err != nil {
				return fmt.Errorf("verify temporary Core App cache: %w", err)
			} else if !ok {
				return fmt.Errorf("temporary Core App cache is missing app.yml")
			}
			return nil
		}); err != nil {
			return "", fmt.Errorf("install %s: %w", source.Name, err)
		}
		return source.Name, nil
	}
	return "", fmt.Errorf("Core App assets are unavailable")
}
