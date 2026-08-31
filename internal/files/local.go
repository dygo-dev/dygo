package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalBlobStore stores private files below one project-owned directory.
type LocalBlobStore struct{ root string }

// NewLocalBlobStore creates a private local filesystem driver.
func NewLocalBlobStore(root string) (LocalBlobStore, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return LocalBlobStore{}, fmt.Errorf("file storage root is required")
	}
	root = filepath.Join(root, "private")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return LocalBlobStore{}, fmt.Errorf("create private file storage: %w", err)
	}
	return LocalBlobStore{root: root}, nil
}

func (s LocalBlobStore) Path(key string) string {
	if !safeKey(key) {
		return ""
	}
	return filepath.Join(s.root, key)
}

func (s LocalBlobStore) Put(_ context.Context, key string, reader io.Reader, size int64) error {
	path := s.Path(key)
	if path == "" {
		return fmt.Errorf("invalid file storage key")
	}
	if size > maxFileSize {
		return fmt.Errorf("file is too large")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := io.Copy(file, reader)
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	if written > maxFileSize || (size >= 0 && written != size) {
		_ = os.Remove(path)
		return fmt.Errorf("file size does not match upload")
	}
	return file.Sync()
}

func (s LocalBlobStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path := s.Path(key)
	if path == "" {
		return nil, fmt.Errorf("invalid file storage key")
	}
	return os.Open(path)
}

func (s LocalBlobStore) Remove(_ context.Context, key string) error {
	path := s.Path(key)
	if path == "" {
		return fmt.Errorf("invalid file storage key")
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func safeKey(key string) bool {
	return key != "" && filepath.Base(key) == key && key != "." && key != ".." && !strings.ContainsAny(key, `/\\`)
}
