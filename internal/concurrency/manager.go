package concurrency

import (
	"fmt"
	"sync"
)

// Manager tracks and enforces concurrency limits across the system
type Manager struct {
	mu        sync.RWMutex
	functions map[string]*Semaphore // functionID -> semaphore
	workflows map[string]*Semaphore // schemaID -> semaphore
	keyed     map[string]*Semaphore // "scope:keyValue" -> semaphore
}

// NewManager creates a new concurrency manager
func NewManager() *Manager {
	return &Manager{
		functions: make(map[string]*Semaphore),
		workflows: make(map[string]*Semaphore),
		keyed:     make(map[string]*Semaphore),
	}
}

// AcquireFunction acquires a concurrency slot for a function execution.
// Blocks until a slot is available. Returns a release function.
func (m *Manager) AcquireFunction(functionID string, limit int) func() {
	sem := m.getOrCreate(&m.functions, functionID, limit)
	return sem.Acquire()
}

// TryAcquireWorkflow attempts to acquire a concurrency slot for a workflow execution.
// Returns a release function and true if a slot is immediately available.
// Returns nil, false if no slot is available (caller should retry later).
func (m *Manager) TryAcquireWorkflow(schemaID string, limit int) (func(), bool) {
	sem := m.getOrCreate(&m.workflows, schemaID, limit)
	return sem.TryAcquire()
}

// AcquireKeyed acquires a concurrency slot scoped by a key value.
// Blocks until a slot is available. Returns a release function.
func (m *Manager) AcquireKeyed(scope, keyValue string, limit int) func() {
	key := fmt.Sprintf("%s:%s", scope, keyValue)
	sem := m.getOrCreate(&m.keyed, key, limit)
	return sem.Acquire()
}

func (m *Manager) getOrCreate(store *map[string]*Semaphore, key string, limit int) *Semaphore {
	m.mu.Lock()
	defer m.mu.Unlock()
	sem, exists := (*store)[key]
	if !exists {
		sem = NewSemaphore(limit)
		(*store)[key] = sem
	}
	return sem
}
