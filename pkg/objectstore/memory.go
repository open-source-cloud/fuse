package objectstore

import (
	"context"
	"sync"
)

// MemoryObjectStore is an in-memory implementation of ObjectStore for dev/test.
// Not safe for multi-process use; all data is lost on process exit.
type MemoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

// NewMemoryObjectStore creates a new in-memory object store.
func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{
		objects: make(map[string][]byte),
	}
}

// Put stores data under the given key in memory.
func (m *MemoryObjectStore) Put(_ context.Context, key string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]byte, len(data))
	copy(cp, data)
	m.objects[key] = cp
	return nil
}

// Get retrieves data by key from memory.
func (m *MemoryObjectStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.objects[key]
	if !ok {
		return nil, ErrObjectNotFound
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

// Delete removes the object at the given key from memory.
func (m *MemoryObjectStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.objects, key)
	return nil
}

// Exists checks whether a key exists in memory.
func (m *MemoryObjectStore) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.objects[key]
	return ok, nil
}
