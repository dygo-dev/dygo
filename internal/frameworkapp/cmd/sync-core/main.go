package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	if err := syncCore(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func syncCore() error {
	sourceRoot := filepath.Join("..", "..", "apps", "core")
	targetRoot := filepath.Join("bundled", "core")
	if err := os.RemoveAll(targetRoot); err != nil {
		return fmt.Errorf("clear bundled Core App: %w", err)
	}
	return filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Name() == ".DS_Store" {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("read Core App source %s: %w", relative, err)
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("write bundled Core App %s: %w", relative, err)
		}
		return nil
	})
}
