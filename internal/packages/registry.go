// Package packages provide a MemoryRegistry for the workflow node packages
package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

type (
	Registry interface {
		Register(packageID string, provider workflow.Package)
		Get(packageID string) (workflow.Package, error)
	}

	// MemoryRegistry is a MemoryRegistry for the packages
	MemoryRegistry struct {
		packages map[string]workflow.Package
	}
)

// NewPackageRegistry creates a new provider MemoryRegistry
func NewPackageRegistry() Registry {
	return &MemoryRegistry{
		packages: make(map[string]workflow.Package),
	}
}

// Register registers a provider by ID
func (r *MemoryRegistry) Register(packageID string, pkg workflow.Package) {
	log.Info().Str("packageID", pkg.ID()).Msg("Package registered")
	r.packages[packageID] = pkg
}

// Get returns a provider by ID
func (r *MemoryRegistry) Get(packageID string) (workflow.Package, error) {
	provider, ok := r.packages[packageID]
	if !ok {
		return nil, fmt.Errorf("package %s not found", packageID)
	}
	return provider, nil
}
