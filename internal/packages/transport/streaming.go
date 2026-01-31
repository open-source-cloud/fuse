package transport

import (
	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// StreamingFunctionTransport defines the interface for streaming function execution
type StreamingFunctionTransport interface {
	FunctionTransport
	// ExecuteStream executes a function with streaming support
	// The callback is called for each stream chunk
	ExecuteStream(handle actor.Handle, execInfo *workflow.ExecutionInfo, callback workflow.StreamCallback) error
}
