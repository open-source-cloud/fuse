// Package internalpackages hardcoded internal packages
package internalpackages

import (
	privpackages "github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/internalpackages/debug"
	"github.com/open-source-cloud/fuse/internal/packages/internalpackages/logic"
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/packages"
)

// New creates new InternalPackages service
func New(registry privpackages.Registry) *InternalPackages {
	return &InternalPackages{
		registry: registry,
	}
}

// InternalPackages service for registering internal packages
type InternalPackages struct {
	registry privpackages.Registry
}

// Register registers internal packages
func (p *InternalPackages) Register() {
	listOfInternalPackages := []*packages.Package{
		debug.New(),
		logic.New(),
	}
	for _, pkg := range listOfInternalPackages {
		for _, function := range pkg.Functions {
			function.Metadata.Transport = transport.Internal
		}
		p.registry.Register(pkg)
	}
}
