// Package frameworkapp installs framework-managed Apps into generated projects.
package frameworkapp

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	if fsys == nil {
		return false, nil
	}
	info, err := fs.Stat(fsys, "app.yml")
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
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
		if err := replaceCoreDir(source.FS, filepath.Join(root, filepath.FromSlash(CoreProjectPath))); err != nil {
			return "", fmt.Errorf("install %s: %w", source.Name, err)
		}
		return source.Name, nil
	}
	return "", fmt.Errorf("Core App assets are unavailable")
}

func replaceCoreDir(source fs.FS, target string) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create Core App cache parent: %w", err)
	}
	temp, err := os.MkdirTemp(parent, ".core-app-*")
	if err != nil {
		return fmt.Errorf("create temporary Core App cache: %w", err)
	}
	defer func() {
		if temp != "" {
			_ = os.RemoveAll(temp)
		}
	}()

	if err := copyFS(source, temp); err != nil {
		return err
	}
	if ok, err := HasManifest(os.DirFS(temp)); err != nil {
		return fmt.Errorf("verify temporary Core App cache: %w", err)
	} else if !ok {
		return fmt.Errorf("temporary Core App cache is missing app.yml")
	}

	backup := target + ".previous"
	_ = os.RemoveAll(backup)
	hadExisting := false
	if _, err := os.Stat(target); err == nil {
		hadExisting = true
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("move existing Core App cache aside: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check existing Core App cache: %w", err)
	}

	if err := os.Rename(temp, target); err != nil {
		if hadExisting {
			_ = os.Rename(backup, target)
		}
		return fmt.Errorf("replace Core App cache: %w", err)
	}
	temp = ""
	if hadExisting {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func copyFS(source fs.FS, target string) error {
	return fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		destination := filepath.Join(target, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return fmt.Errorf("read Core App asset %s: %w", name, err)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("create Core App asset directory %s: %w", name, err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			return fmt.Errorf("write Core App asset %s: %w", name, err)
		}
		return nil
	})
}
