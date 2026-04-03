package workflowactions

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestNoopAction_Type(t *testing.T) {
	action := &NoopAction{}
	assert.Equal(t, ActionNoop, action.Type())
}

func TestRunFunctionAction_Type(t *testing.T) {
	action := &RunFunctionAction{
		ThreadID:       0,
		FunctionID:     "test/func",
		FunctionExecID: workflow.NewExecID(0),
		Args:           map[string]any{"key": "value"},
	}
	assert.Equal(t, ActionRunFunction, action.Type())
}

func TestRunParallelFunctionsAction_Type(t *testing.T) {
	action := &RunParallelFunctionsAction{
		Actions: []*RunFunctionAction{
			{ThreadID: 0, FunctionID: "test/a"},
			{ThreadID: 1, FunctionID: "test/b"},
		},
	}
	assert.Equal(t, ActionRunParallelFunctions, action.Type())
}

func TestRetryFunctionAction_Type(t *testing.T) {
	action := &RetryFunctionAction{
		RunFunctionAction: RunFunctionAction{FunctionID: "test/func"},
		Delay:             5 * time.Second,
		Attempt:           2,
	}
	assert.Equal(t, ActionRetryFunction, action.Type())
}

func TestSleepAction_Type(t *testing.T) {
	action := &SleepAction{
		ThreadID: 0,
		ExecID:   workflow.NewExecID(0),
		Duration: 10 * time.Second,
		Reason:   "rate limit",
	}
	assert.Equal(t, ActionSleep, action.Type())
}

func TestWaitForEventAction_Type(t *testing.T) {
	action := &WaitForEventAction{
		ThreadID:    0,
		ExecID:      workflow.NewExecID(0),
		AwakeableID: "awk-123",
		Timeout:     30 * time.Second,
		Filter:      "",
	}
	assert.Equal(t, ActionWaitForEvent, action.Type())
}

func TestRunSubWorkflowAction_Type(t *testing.T) {
	action := &RunSubWorkflowAction{
		ParentWorkflowID: workflow.NewID(),
		ParentThreadID:   0,
		ParentExecID:     workflow.NewExecID(0),
		SchemaID:         "child-schema",
		Input:            map[string]any{"data": "test"},
		Async:            false,
	}
	assert.Equal(t, ActionRunSubWorkflow, action.Type())
}
