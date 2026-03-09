package storage

import (
	"context"
	"os"
	"path/filepath"
)

// LocalStorage persists images on the local filesystem under baseDir.
// The identifier stored in the database is the bare relative filename
// (e.g. "abc123.jpg" or "avatars/abc123.jpg"), matching the format that
// already exists in the database so that existing records continue to work.
type LocalStorage struct {
	baseDir string // e.g. "static/uploads"
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

func (s *LocalStorage) Upload(ctx context.Context, data []byte, filename string) (UploadResult, error) {
	path := filepath.Join(s.baseDir, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return UploadResult{}, err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return UploadResult{}, err
	}
	// Return bare filename as the identifier — frontend prepends /static/uploads/.
	return UploadResult{URL: filename}, nil
}

func (s *LocalStorage) Delete(ctx context.Context, identifier string) error {
	return os.Remove(filepath.Join(s.baseDir, identifier))
}

func (s *LocalStorage) FetchContent(ctx context.Context, identifier string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.baseDir, identifier))
}
