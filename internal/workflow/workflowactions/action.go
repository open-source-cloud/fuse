// Package workflowactions has all the potential actions workflows can take. This instructs the caller to know what to do
package workflowactions

import (
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const (
	// ActionNoop no operation action type
	ActionNoop ActionType = "noop"
	// ActionRunFunction run a workflow function action type
	ActionRunFunction ActionType = "function:run"
	// ActionRunParallelFunctions run several workflow functions in parallel action type
	ActionRunParallelFunctions ActionType = "functions:parallel-run"
	// ActionRetryFunction retry a failed function action type
	ActionRetryFunction ActionType = "function:retry"
)

type (
	// Action defines the basic interface of an Action
	Action interface {
		Type() ActionType
	}

	// NoopAction No Operation action
	NoopAction struct{}

	// RunFunctionAction run function action
	RunFunctionAction struct {
		ThreadID       uint16
		FunctionID     string
		FunctionExecID workflow.ExecID
		Args           map[string]any
	}

	// RunParallelFunctionsAction run several parallel functions action
	RunParallelFunctionsAction struct {
		Actions []*RunFunctionAction
	}

	// RetryFunctionAction retry a failed function with delay
	RetryFunctionAction struct {
		RunFunctionAction
		Delay   time.Duration
		Attempt int
	}
)

// Type returns the type for a NoopAction action
func (a *NoopAction) Type() ActionType {
	return ActionNoop
}

// Type returns the type for a RunFunctionAction action
func (a *RunFunctionAction) Type() ActionType {
	return ActionRunFunction
}

// Type returns the type for a RunParallelFunctionsAction action
func (a *RunParallelFunctionsAction) Type() ActionType {
	return ActionRunParallelFunctions
}

// Type returns the type for a RetryFunctionAction action
func (a *RetryFunctionAction) Type() ActionType {
	return ActionRetryFunction
}
