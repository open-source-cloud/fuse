package workflow

const (
	// ActionNoop no operation action type
	ActionNoop        ActionType = "noop"
	// ActionRunFunction run a workflow function action type
	ActionRunFunction          ActionType = "function:run"
	// ActionRunParallelFunctions run several workflow functions in parallel action type
	ActionRunParallelFunctions ActionType = "functions:parallel-run"
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
		FunctionExecID ExecID
		Args           map[string]any
	}

	// RunParallelFunctionsAction run several parallel functions action
	RunParallelFunctionsAction struct {
		Actions []*RunFunctionAction
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
