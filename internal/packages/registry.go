// Package packages provide a MemoryRegistry for the workflow node packages
package packages

import (
	"fmt"
	"github.com/rs/zerolog/log"
)

type (
	Registry interface {
		Register(packageID string, provider Package)
		Get(packageID string) (Package, error)
	}

	// MemoryRegistry is a MemoryRegistry for the packages
	MemoryRegistry struct {
		packages map[string]Package
	}
)

// NewPackageRegistry creates a new provider MemoryRegistry
func NewPackageRegistry() Registry {
	return &MemoryRegistry{
		packages: make(map[string]Package),
	}
}

// Register registers a provider by id
func (r *MemoryRegistry) Register(packageID string, pkg Package) {
	log.Info().Str("packageID", pkg.ID()).Msg("Package registered")
	r.packages[packageID] = pkg
}

// Get returns a provider by id
func (r *MemoryRegistry) Get(packageID string) (Package, error) {
	provider, ok := r.packages[packageID]
	if !ok {
		return nil, fmt.Errorf("package %s not found", packageID)
	}
	return provider, nil
}
