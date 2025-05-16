package workflow

const (
	ActionNoop                 ActionType = "noop"
	ActionRunFunction          ActionType = "function:run"
	ActionRunParallelFunctions ActionType = "functions:parallel-run"
)

type (
	Action interface {
		Type() ActionType
	}

	NoopAction struct{}

	RunFunctionAction struct {
		Thread         int
		FunctionID     string
		FunctionExecID string
		Args           map[string]any
	}

	RunParallelFunctionsAction struct {
		Actions []*RunFunctionAction
	}
)

func (a *NoopAction) Type() ActionType {
	return ActionNoop
}

func (a *RunFunctionAction) Type() ActionType {
	return ActionRunFunction
}

func (a *RunParallelFunctionsAction) Type() ActionType {
	return ActionRunParallelFunctions
}
