// Package system provides built-in system functions for workflow control flow (sleep, wait)
package system

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageID is the id of the system function package
const PackageID = "system"

// New creates a new system Package
func New() *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(SleepFunctionID, SleepFunctionMetadata(), SleepFunction),
		workflow.NewFunction(WaitFunctionID, WaitFunctionMetadata(), WaitFunction),
		workflow.NewFunction(SubWorkflowFunctionID, SubWorkflowFunctionMetadata(), SubWorkflowFunction),
		workflow.NewFunction(ForEachFunctionID, ForEachFunctionMetadata(), ForEachFunction),
	)
}

// SleepFullFunctionID is the full function ID for system/sleep
const SleepFullFunctionID = PackageID + "/" + SleepFunctionID

// WaitFullFunctionID is the full function ID for system/wait
const WaitFullFunctionID = PackageID + "/" + WaitFunctionID
