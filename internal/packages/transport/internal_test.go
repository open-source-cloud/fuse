package transport

import (
	"testing"

	"ergo.services/ergo/gen"
	"ergo.services/ergo/lib"
	"ergo.services/ergo/testing/unit"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeHandle is a minimal actor.Handle for the async Execute path tests. Node()
// returns nil because these tests never reach the rebound Finish.
type fakeHandle struct{}

func (fakeHandle) Send(any, any) error { return nil }
func (fakeHandle) Node() gen.Node      { return nil }

func TestSendAsyncFunctionResult_UsesHandlerAtom(t *testing.T) {
	t.Parallel()

	events := lib.NewQueueMPSC()
	n := unit.NewTestNode(t, events, unit.TestOptions{NodeName: "fuse@localhost"})

	wfID := workflow.ID("550e8400-e29b-41d4-a716-446655440000")
	execID := workflow.NewExecID(1)
	out := workflow.FunctionOutput{Data: map[string]any{"k": "v"}}

	err := sendAsyncFunctionResult(n, wfID, execID, out)
	require.NoError(t, err)

	raw, popped := events.Pop()
	require.True(t, popped)
	ev, ok := raw.(unit.SendEvent)
	require.True(t, ok, "expected SendEvent, got %T", raw)
	wantAtom := gen.Atom(actornames.WorkflowHandlerName(wfID))
	require.Equal(t, wantAtom, ev.To)

	msg, ok := ev.Message.(messaging.Message)
	require.True(t, ok, "expected messaging.Message, got %T", ev.Message)
	require.Equal(t, messaging.AsyncFunctionResult, msg.Type)
}

func TestSendAsyncFunctionResult_NilNode(t *testing.T) {
	t.Parallel()

	err := sendAsyncFunctionResult(nil, "wf", workflow.NewExecID(0), workflow.FunctionOutput{})
	require.ErrorIs(t, err, errNilNode)
}

func TestExecuteSync_RunsFunctionWithoutHandle(t *testing.T) {
	t.Parallel()

	called := false
	fn := func(_ *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		called = true
		return workflow.NewFunctionResultSuccessWith(map[string]any{"ok": true}), nil
	}
	tr := NewInternalFunctionTransport(fn)

	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", nil)
	res, err := tr.ExecuteSync(execInfo)
	require.NoError(t, err)
	require.True(t, called, "the function should run")
	assert.False(t, res.Async, "a synchronous invocation returns its result inline")
	assert.Equal(t, workflow.FunctionSuccess, res.Output.Status)
	assert.Equal(t, true, res.Output.Data["ok"])
}

func TestExecuteSync_NilExecutionInfoReturnsError(t *testing.T) {
	t.Parallel()

	tr := NewInternalFunctionTransport(func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		return workflow.NewFunctionResultSuccess(), nil
	})
	_, err := tr.ExecuteSync(nil)
	require.ErrorIs(t, err, errNilExecutionInfo)
}

func TestExecute_NilExecutionInfoReturnsError(t *testing.T) {
	t.Parallel()

	tr := NewInternalFunctionTransport(func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		return workflow.NewFunctionResultSuccess(), nil
	})
	_, err := tr.Execute(fakeHandle{}, nil)
	require.ErrorIs(t, err, errNilExecutionInfo)
}

// A nil function pointer reaches the transport when an internal package is decoded from
// persistence (PackagedFunction.Function is json:"-") and registered as executable. Executing
// it must return an error, never call a nil func value (which panics the worker and leaves the
// workflow stuck "running").
func TestExecute_NilFunctionReturnsError(t *testing.T) {
	t.Parallel()

	tr := NewInternalFunctionTransport(nil)
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", nil)

	_, err := tr.Execute(fakeHandle{}, execInfo)
	require.ErrorIs(t, err, errNilFunction)
}

func TestExecuteSync_NilFunctionReturnsError(t *testing.T) {
	t.Parallel()

	tr := NewInternalFunctionTransport(nil)
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", nil)

	_, err := tr.ExecuteSync(execInfo)
	require.ErrorIs(t, err, errNilFunction)
}
