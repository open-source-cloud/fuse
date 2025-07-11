// Package logic provides logic nodes for workflows
package logic

import "github.com/open-source-cloud/fuse/pkg/workflow"

// PackageID is the id of the debug function package
const PackageID = "fuse/pkg/logic"

// New creates a new logic Package
func New() *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(IfFunctionID, IfFunctionMetadata(), IfFunction),
		workflow.NewFunction(RandFunctionID, RandFunctionMetadata(), RandFunction),
		workflow.NewFunction(TimerFunctionID, TimerFunctionMetadata(), TimerFunction),
		workflow.NewFunction(SumFunctionID, SumFunctionMetadata(), SumFunction),
	)
}
