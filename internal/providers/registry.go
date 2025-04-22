package providers

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/providers/debug"
	"github.com/open-source-cloud/fuse/internal/providers/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

var listOfProviders = []workflow.NodeProvider{
	debug.NewNodeProvider(),
	logic.NewNodeProvider(),
}

// Registry is a registry for the providers
type Registry struct {
	providers map[string]workflow.NodeProvider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	store := &Registry{
		providers: make(map[string]workflow.NodeProvider),
	}
	for _, provider := range listOfProviders {
		store.Register(provider.ID(), provider)
	}
	return store
}

// Register registers a provider by ID
func (r *Registry) Register(providerID string, provider workflow.NodeProvider) {
	r.providers[providerID] = provider
}

// GetProvider returns a provider by ID
func (r *Registry) Get(providerID string) (workflow.NodeProvider, error) {
	provider, ok := r.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", providerID)
	}
	return provider, nil
}
