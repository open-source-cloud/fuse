package packages

import (
	"github.com/open-source-cloud/fuse/internal/packages/functions/debug"
	"github.com/open-source-cloud/fuse/internal/packages/functions/http"
	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// InternalPackage defines the interface for an internal package
	InternalPackages interface {
		List() []*workflow.Package
	}
)

// NewInternal creates new InternalPackages service
func NewInternal() InternalPackages {
	return &DefaultInternalPackages{}
}

// InternalPackages service for registering internal packages
type DefaultInternalPackages struct{}

// List returns the list of internal packages
func (p *DefaultInternalPackages) List() []*workflow.Package {
	return []*workflow.Package{
		debug.New(),
		logic.New(),
		http.New(),
	}
}
