// Package packages provide a MemoryRegistry for the workflow node packages
package packages

import (
	"sync"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

type (
	// Registry defines the interface of a package Registry service
	Registry interface {
		Register(pkg *workflow.Package)
		Get(pkgID string) (*LoadedPackage, error)
		Has(pkgID string) bool
		List() ([]*LoadedPackage, error)
	}

	// MemoryRegistry is a MemoryRegistry for the packages
	MemoryRegistry struct {
		packages map[string]*LoadedPackage
		mu       sync.RWMutex
	}
)

// pkgRegistry is a global package registry
var pkgRegistry Registry

// NewPackageRegistry creates a new provider MemoryRegistry
func NewPackageRegistry() Registry {
	if pkgRegistry == nil {
		pkgRegistry = &MemoryRegistry{
			packages: make(map[string]*LoadedPackage),
		}
	}
	return pkgRegistry
}

// Register registers a provider by id.
//
// A registration never downgrades an executable (code-backed) function to a metadata-only one.
// Data-only registrations — API round-trips (PUT /v1/packages), persistence reloads, and
// cluster replication — carry no function pointer (PackagedFunction.Function is json:"-"), so for
// those functions we keep the executable transport already registered from code. A code-backed
// registration (non-nil transport) still upgrades/replaces as usual.
func (r *MemoryRegistry) Register(pkg *workflow.Package) {
	r.mu.Lock()
	defer r.mu.Unlock()

	incoming := MapToRegistryPackage(pkg)
	if existing, ok := r.packages[pkg.ID]; ok {
		for id, fn := range incoming.Functions {
			if fn.Transport != nil {
				continue
			}
			if prev, had := existing.Functions[id]; had && prev.Transport != nil {
				incoming.Functions[id] = prev
			}
		}
	}

	log.Info().Str("packageID", pkg.ID).Msg("Package registered")
	r.packages[pkg.ID] = incoming
}

// Get returns a provider by id
func (r *MemoryRegistry) Get(pkgID string) (*LoadedPackage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, exists := r.packages[pkgID]
	if !exists {
		return nil, ErrLoadedPackageNotFound
	}
	return pkg, nil
}

// Has returns true if the package is registered
func (r *MemoryRegistry) Has(pkgID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.packages[pkgID]
	return exists
}

// List returns all the packages
func (r *MemoryRegistry) List() ([]*LoadedPackage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	packages := make([]*LoadedPackage, 0, len(r.packages))
	for _, pkg := range r.packages {
		packages = append(packages, pkg)
	}
	return packages, nil
}
