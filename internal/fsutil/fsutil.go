// Package fsutil contains filesystem operations shared by framework installers.
package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ReplaceDir prepares a directory and replaces target with it atomically.
func ReplaceDir(target, pattern, label string, prepare func(string) error) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create %s cache parent: %w", label, err)
	}
	temp, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return fmt.Errorf("create temporary %s cache: %w", label, err)
	}
	defer func() {
		if temp != "" {
			_ = os.RemoveAll(temp)
		}
	}()

	if err := prepare(temp); err != nil {
		return err
	}

	backup := target + ".previous"
	_ = os.RemoveAll(backup)
	hadExisting := false
	if _, err := os.Stat(target); err == nil {
		hadExisting = true
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("move existing %s cache aside: %w", label, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check existing %s cache: %w", label, err)
	}
	if err := os.Rename(temp, target); err != nil {
		if hadExisting {
			_ = os.Rename(backup, target)
		}
		return fmt.Errorf("replace %s cache: %w", label, err)
	}
	temp = ""
	if hadExisting {
		_ = os.RemoveAll(backup)
	}
	return nil
}

// CopyFS copies all files and directories in source below target.
func CopyFS(source fs.FS, target, label string) error {
	return fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		destination := filepath.Join(target, filepath.FromSlash(name))
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return fmt.Errorf("create %s directory %s: %w", label, name, err)
			}
			return nil
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return fmt.Errorf("read %s %s: %w", label, name, err)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("create %s parent %s: %w", label, name, err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			return fmt.Errorf("write %s %s: %w", label, name, err)
		}
		return nil
	})
}

// HasFile reports whether fsys contains the named regular file.
func HasFile(fsys fs.FS, name string) (bool, error) {
	if fsys == nil {
		return false, nil
	}
	info, err := fs.Stat(fsys, name)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}
