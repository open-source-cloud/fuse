package repositories

import (
	"sort"
	"sync"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// MemoryEnvironmentRepository is an in-memory EnvironmentRepository seeded with the default
// environment so the registry is never empty (matches the postgres migration seed).
type MemoryEnvironmentRepository struct {
	mu           sync.RWMutex
	environments map[string]*workflow.Environment
}

// NewMemoryEnvironmentRepository creates a memory environment repository seeded with the
// default environment.
func NewMemoryEnvironmentRepository() *MemoryEnvironmentRepository {
	return &MemoryEnvironmentRepository{
		environments: map[string]*workflow.Environment{
			workflow.DefaultEnvironmentName: workflow.NewEnvironment(workflow.DefaultEnvironmentName, "Default environment"),
		},
	}
}

// FindByID finds an environment by name.
func (r *MemoryEnvironmentRepository) FindByID(name string) (*workflow.Environment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	env, ok := r.environments[name]
	if !ok {
		return nil, ErrEnvironmentNotFound
	}
	return env, nil
}

// FindAll returns all environments sorted by name.
func (r *MemoryEnvironmentRepository) FindAll() ([]*workflow.Environment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	envs := make([]*workflow.Environment, 0, len(r.environments))
	for _, env := range r.environments {
		envs = append(envs, env)
	}
	sort.Slice(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })
	return envs, nil
}

// Save upserts an environment.
func (r *MemoryEnvironmentRepository) Save(env *workflow.Environment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.environments[env.Name] = env
	return nil
}

// Delete removes an environment by name.
func (r *MemoryEnvironmentRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.environments, name)
	return nil
}
