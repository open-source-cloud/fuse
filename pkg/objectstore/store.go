// Package objectstore defines the interface and implementations for pluggable
// object storage backends used to persist workflow data payloads.
package objectstore

import (
	"context"
	"errors"
)

var (
	// ErrObjectNotFound is returned when a requested object does not exist.
	ErrObjectNotFound = errors.New("object not found")
)

// ObjectStore is the interface for storing and retrieving binary data payloads
// keyed by a hierarchical string path (e.g., "workflows/{id}/journal/1/input.json").
//
// Implementations:
//   - MemoryObjectStore: in-memory map for dev/test
//   - FilesystemObjectStore: disk-based for shared volumes (NFS, EFS, K8s PVCs)
//   - S3ObjectStore: S3-compatible for production (AWS S3, MinIO, LocalStack)
type ObjectStore interface {
	// Put stores data under the given key, overwriting any existing value.
	Put(ctx context.Context, key string, data []byte) error

	// Get retrieves the data stored under the given key.
	// Returns ErrObjectNotFound if the key does not exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes the object at the given key.
	// Returns nil if the key does not exist (idempotent).
	Delete(ctx context.Context, key string) error

	// Exists checks whether an object exists at the given key.
	Exists(ctx context.Context, key string) (bool, error)
}
