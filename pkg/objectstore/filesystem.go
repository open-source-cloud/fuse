package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FilesystemObjectStore stores objects as files on disk. The key becomes a
// relative file path under the configured base directory.
//
// Multiple FUSE instances can share this store by mounting the same volume
// (NFS, AWS EFS, GlusterFS, or a K8s PersistentVolumeClaim with ReadWriteMany).
//
// Writes are atomic: data is written to a temporary file in the same directory,
// then renamed to the target path. This prevents partial reads. File locking is
// not needed because the HA claim pattern guarantees a single writer per object.
type FilesystemObjectStore struct {
	basePath string
}

// NewFilesystemObjectStore creates a filesystem-backed object store rooted at basePath.
// The base directory is created if it does not exist.
func NewFilesystemObjectStore(basePath string) (*FilesystemObjectStore, error) {
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("objectstore/filesystem: create base path %q: %w", basePath, err)
	}
	return &FilesystemObjectStore{basePath: basePath}, nil
}

func (f *FilesystemObjectStore) fullPath(key string) string {
	return filepath.Join(f.basePath, filepath.FromSlash(key))
}

// Put stores data as a file on disk using atomic write (temp file + rename).
func (f *FilesystemObjectStore) Put(_ context.Context, key string, data []byte) error {
	target := f.fullPath(key)
	dir := filepath.Dir(target)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("objectstore/filesystem: mkdir %q: %w", dir, err)
	}

	// Atomic write: temp file + rename
	tmp, err := os.CreateTemp(dir, ".fuse-obj-*")
	if err != nil {
		return fmt.Errorf("objectstore/filesystem: create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("objectstore/filesystem: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("objectstore/filesystem: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("objectstore/filesystem: rename %q -> %q: %w", tmpName, target, err)
	}
	return nil
}

// Get reads a file from disk by key.
func (f *FilesystemObjectStore) Get(_ context.Context, key string) ([]byte, error) {
	data, err := os.ReadFile(f.fullPath(key))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("objectstore/filesystem: read %q: %w", key, err)
	}
	return data, nil
}

// Delete removes a file from disk by key. Idempotent if the file does not exist.
func (f *FilesystemObjectStore) Delete(_ context.Context, key string) error {
	err := os.Remove(f.fullPath(key))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("objectstore/filesystem: delete %q: %w", key, err)
	}
	return nil
}

// Exists checks whether a file exists on disk for the given key.
func (f *FilesystemObjectStore) Exists(_ context.Context, key string) (bool, error) {
	_, err := os.Stat(f.fullPath(key))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("objectstore/filesystem: stat %q: %w", key, err)
	}
	return true, nil
}
