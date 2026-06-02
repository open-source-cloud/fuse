package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionInfo_LeavesHandleNil(t *testing.T) {
	t.Parallel()

	input, err := NewFunctionInputWith(map[string]any{"k": "v"})
	require.NoError(t, err)

	execInfo := NewExecutionInfo("wf-1", NewExecID(1), input)

	assert.Equal(t, ID("wf-1"), execInfo.WorkflowID)
	assert.Equal(t, input, execInfo.Input)
	assert.Nil(t, execInfo.Handle, "Handle should default to nil; it is populated by the internal transport")
}

func TestExecutionInfo_HandleIsSettable(t *testing.T) {
	t.Parallel()

	execInfo := NewExecutionInfo("wf-1", NewExecID(1), nil)

	type fakeHandle struct{ name string }
	h := &fakeHandle{name: "worker"}
	execInfo.Handle = h

	got, ok := execInfo.Handle.(*fakeHandle)
	require.True(t, ok, "Handle should round-trip its concrete type via a type assertion")
	assert.Equal(t, "worker", got.name)
}
