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
	// ActionSleep pause workflow execution for a duration
	ActionSleep ActionType = "workflow:sleep"
	// ActionWaitForEvent pause workflow execution until an external event arrives
	ActionWaitForEvent ActionType = "workflow:wait-for-event"
	// ActionRunSubWorkflow run a sub-workflow
	ActionRunSubWorkflow ActionType = "workflow:subworkflow:run"
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

	// SleepAction pauses workflow execution for a duration
	SleepAction struct {
		ThreadID uint16
		ExecID   workflow.ExecID
		Duration time.Duration
		Reason   string
	}

	// WaitForEventAction pauses workflow execution until an external event arrives
	WaitForEventAction struct {
		ThreadID    uint16
		ExecID      workflow.ExecID
		AwakeableID string
		Timeout     time.Duration
		Filter      string
	}

	// RunSubWorkflowAction runs a child workflow
	RunSubWorkflowAction struct {
		ParentWorkflowID workflow.ID
		ParentThreadID   uint16
		ParentExecID     workflow.ExecID
		SchemaID         string
		Input            map[string]any
		Async            bool
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

// Type returns the type for a SleepAction action
func (a *SleepAction) Type() ActionType {
	return ActionSleep
}

// Type returns the type for a WaitForEventAction action
func (a *WaitForEventAction) Type() ActionType {
	return ActionWaitForEvent
}

// Type returns the type for a RunSubWorkflowAction action
func (a *RunSubWorkflowAction) Type() ActionType {
	return ActionRunSubWorkflow
}
