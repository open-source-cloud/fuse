// Package logic provides logic nodes for workflows
package logic

import "github.com/open-source-cloud/fuse/pkg/packages"

// PackageID is the id of the debug function package
const PackageID = "fuse/pkg/logic"

// New creates a new logic Package
func New() *packages.Package {
	return packages.NewPackage(
		PackageID,
		packages.NewFunction(IfFunctionID, IfFunctionMetadata(), IfFunction),
		packages.NewFunction(RandFunctionID, RandFunctionMetadata(), RandFunction),
		packages.NewFunction(TimerFunctionID, TimerFunctionMetadata(), TimerFunction),
		packages.NewFunction(SumFunctionID, SumFunctionMetadata(), SumFunction),
	)
}
