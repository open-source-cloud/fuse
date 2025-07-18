package packages

import (
	"github.com/open-source-cloud/fuse/internal/packages/functions/debug"
	"github.com/open-source-cloud/fuse/internal/packages/functions/http"
	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// NewInternal creates new InternalPackages service
func NewInternal(registry Registry) *InternalPackages {
	return &InternalPackages{
		registry: registry,
	}
}

// InternalPackages service for registering internal packages
type InternalPackages struct {
	registry Registry
}

// Register registers internal packages
func (p *InternalPackages) Register() {
	for _, pkg := range p.List() {
		for _, function := range pkg.Functions {
			function.Metadata.Transport = transport.Internal
		}
		p.registry.Register(pkg)
	}
}

func (p *InternalPackages) List() []*workflow.Package {
	return []*workflow.Package{
		debug.New(),
		logic.New(),
		http.New(),
	}
}
