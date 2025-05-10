// Packages providers provides a registry for the workflow node providers
package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/packages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/logic"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

var listOfPackages = []workflow.Package{
	debug.New(),
	logic.New(),
}

// Registry is a registry for the providers
type Registry struct {
	providers map[string]workflow.Package
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	store := &Registry{
		providers: make(map[string]workflow.Package),
	}
	for _, provider := range listOfPackages {
		store.Register(provider.ID(), provider)
	}
	return store
}

// Register registers a provider by ID
func (r *Registry) Register(providerID string, provider workflow.Package) {
	r.providers[providerID] = provider
}

// Get returns a provider by ID
func (r *Registry) Get(providerID string) (workflow.Package, error) {
	provider, ok := r.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("package %s not found", providerID)
	}
	return provider, nil
}
