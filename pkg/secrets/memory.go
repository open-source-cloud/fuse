package secrets

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// envSecretPrefix seeds the memory store from FUSE_SECRET_<NAME>=<value> env vars
// into the default environment.
const envSecretPrefix = "FUSE_SECRET_"

// MemorySecretStore is an in-memory ManagedSecretStore for dev and testing.
type MemorySecretStore struct {
	mu   sync.RWMutex
	data map[string]map[string]string // environment -> name -> value
}

// NewMemorySecretStore creates an empty store.
func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{data: make(map[string]map[string]string)}
}

// NewMemorySecretStoreFromEnv seeds from FUSE_SECRET_* env vars into defaultEnv.
func NewMemorySecretStoreFromEnv(defaultEnv string) *MemorySecretStore {
	s := NewMemorySecretStore()
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, envSecretPrefix) {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		name := strings.TrimPrefix(kv[:eq], envSecretPrefix)
		_ = s.Set(context.Background(), Scope{Environment: defaultEnv}, name, kv[eq+1:])
	}
	return s
}

// Resolve returns the secret for (environment, name) or ErrSecretNotFound.
func (s *MemorySecretStore) Resolve(_ context.Context, scope Scope, name string) (SecretValue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if env, ok := s.data[scope.Environment]; ok {
		if v, ok := env[name]; ok {
			return NewSecretValue(v), nil
		}
	}
	return SecretValue{}, fmt.Errorf("%w: %q (environment %q)", ErrSecretNotFound, name, scope.Environment)
}

// Set stores (or replaces) a secret value.
func (s *MemorySecretStore) Set(_ context.Context, scope Scope, name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[scope.Environment] == nil {
		s.data[scope.Environment] = make(map[string]string)
	}
	s.data[scope.Environment][name] = value
	return nil
}

// List returns the secret names in an environment, sorted.
func (s *MemorySecretStore) List(_ context.Context, environment string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.data[environment]))
	for k := range s.data[environment] {
		names = append(names, k)
	}
	sort.Strings(names)
	return names, nil
}

// Delete removes a secret.
func (s *MemorySecretStore) Delete(_ context.Context, scope Scope, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if env, ok := s.data[scope.Environment]; ok {
		delete(env, name)
	}
	return nil
}
