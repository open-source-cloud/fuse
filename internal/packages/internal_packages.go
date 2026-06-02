package packages

import (
	"github.com/open-source-cloud/fuse/internal/packages/functions/ai"
	"github.com/open-source-cloud/fuse/internal/packages/functions/debug"
	"github.com/open-source-cloud/fuse/internal/packages/functions/http"
	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/internal/packages/functions/system"
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// InternalPackages defines the interface for an internal package
	InternalPackages interface {
		List() []*workflow.Package
	}
)

// NewInternal creates new InternalPackages service. The LLM provider registry is
// injected so the ai package can expose chat/agent functions, and the package
// registry backs the agent's tool catalog (synchronous functions become tools).
func NewInternal(providers llm.Registry, registry Registry) InternalPackages {
	return &DefaultInternalPackages{
		providers: providers,
		tools:     NewAgentToolRegistry(registry),
	}
}

// DefaultInternalPackages service for registering internal packages
type DefaultInternalPackages struct {
	providers llm.Registry
	tools     ai.ToolRegistry
}

// List returns the list of internal packages
func (p *DefaultInternalPackages) List() []*workflow.Package {
	return []*workflow.Package{
		debug.New(),
		logic.New(),
		http.New(),
		system.New(),
		ai.New(p.providers, p.tools),
	}
}
