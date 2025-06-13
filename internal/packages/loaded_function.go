package packages

import (
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// NewLoadedFunction creates a new LoadedFunction with transport.FunctionTransport based on metadata.Transport
func NewLoadedFunction(id string, metadata *FunctionMetadata) *LoadedFunction {
	return &LoadedFunction{
		ID:       id,
		Metadata: metadata,
	}
}

// NewLoadedInternalFunction creates a new LoadedFunction with transport.InternalFunctionTransport as transport.FunctionTransport
func NewLoadedInternalFunction(id string, metadata *FunctionMetadata, fn workflow.Function) *LoadedFunction {
	return &LoadedFunction{
		ID:        id,
		Metadata:  metadata,
		Transport: transport.NewInternalFunctionTransport(fn),
	}
}

// LoadedFunction represents an executable LoadedFunction and it's metadata
type LoadedFunction struct {
	ID        string                      `json:"id"`
	Metadata  *FunctionMetadata           `json:"metadata"`
	Transport transport.FunctionTransport `json:"transport"`
}
