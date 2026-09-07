package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := syncStudioApp(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func syncStudioApp() error {
	sourceRoot := filepath.Join("..", "..", "apps", "studio")
	targetRoot := "bundled-app"
	if err := os.RemoveAll(targetRoot); err != nil {
		return fmt.Errorf("clear bundled Studio App: %w", err)
	}
	return filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		if relative != "." && !included(relative) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		targetPath := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("read Studio App source %s: %w", relative, err)
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("write bundled Studio App %s: %w", relative, err)
		}
		return nil
	})
}

func included(name string) bool {
	name = filepath.ToSlash(name)
	return name == "app.yml" ||
		name == "entities" ||
		strings.HasPrefix(name, "entities/") ||
		name == "access" ||
		strings.HasPrefix(name, "access/") ||
		name == "pages" ||
		strings.HasPrefix(name, "pages/")
}
