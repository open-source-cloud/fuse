// Package packages provide a MemoryRegistry for the workflow node packages
package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/packages"
	"github.com/rs/zerolog/log"
)

type (
	// Registry defines the interface of a package Registry service
	Registry interface {
		Register(pkg *packages.Package)
		Get(pkgID string) (*LoadedPackage, error)
	}

	// MemoryRegistry is a MemoryRegistry for the packages
	MemoryRegistry struct {
		packages map[string]*LoadedPackage
	}
)

// NewPackageRegistry creates a new provider MemoryRegistry
func NewPackageRegistry() Registry {
	return &MemoryRegistry{
		packages: make(map[string]*LoadedPackage),
	}
}

// Register registers a provider by id
func (r *MemoryRegistry) Register(pkg *packages.Package) {
	log.Info().Str("packageID", pkg.ID).Msg("Package registered")
	r.packages[pkg.ID] = MapToRegistryPackage(pkg)
}

// Get returns a provider by id
func (r *MemoryRegistry) Get(pkgID string) (*LoadedPackage, error) {
	pkg, exists := r.packages[pkgID]
	if !exists {
		return nil, fmt.Errorf("package %s not found", pkgID)
	}
	return pkg, nil
}
