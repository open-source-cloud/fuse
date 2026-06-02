// Package transport communication between core<->package-providers
package transport

import (
	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// FunctionTransport defines the interface for calling Functions
type FunctionTransport interface {
	// Execute runs the function with a worker handle, enabling asynchronous
	// completion (the handle is used to route the result back via Node().Send).
	Execute(actor.Handle, *workflow.ExecutionInfo) (workflow.FunctionResult, error)
	// ExecuteSync runs the function with no worker handle, for synchronous,
	// in-process invocation (e.g. an ai/agent tool call). It must only be used
	// for functions that complete synchronously; the result is returned inline.
	ExecuteSync(*workflow.ExecutionInfo) (workflow.FunctionResult, error)
}
