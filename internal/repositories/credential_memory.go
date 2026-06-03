package repositories

import (
	"sort"
	"sync"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// MemoryCredentialRepository is an in-memory CredentialRepository for dev and testing.
type MemoryCredentialRepository struct {
	mu          sync.RWMutex
	credentials map[string]*workflow.Credential
}

// NewMemoryCredentialRepository creates an empty memory credential repository.
func NewMemoryCredentialRepository() *MemoryCredentialRepository {
	return &MemoryCredentialRepository{credentials: make(map[string]*workflow.Credential)}
}

// FindByID finds a credential by id.
func (r *MemoryCredentialRepository) FindByID(id string) (*workflow.Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cred, ok := r.credentials[id]
	if !ok {
		return nil, ErrCredentialNotFound
	}
	return cred, nil
}

// FindAll returns all credentials sorted by id.
func (r *MemoryCredentialRepository) FindAll() ([]*workflow.Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	creds := make([]*workflow.Credential, 0, len(r.credentials))
	for _, cred := range r.credentials {
		creds = append(creds, cred)
	}
	sort.Slice(creds, func(i, j int) bool { return creds[i].ID < creds[j].ID })
	return creds, nil
}

// Save upserts a credential.
func (r *MemoryCredentialRepository) Save(cred *workflow.Credential) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.credentials[cred.ID] = cred
	return nil
}

// Delete removes a credential by id.
func (r *MemoryCredentialRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.credentials, id)
	return nil
}
