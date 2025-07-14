// Package transport communication between core<->package-providers
package transport

import (
	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// FunctionTransport defines the interface for calling Functions
type FunctionTransport interface {
	Execute(actor.Handle, *workflow.ExecutionInfo) (workflow.FunctionResult, error)
}
