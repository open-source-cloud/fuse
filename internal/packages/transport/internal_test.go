package transport

import (
	"testing"

	"ergo.services/ergo/gen"
	"ergo.services/ergo/lib"
	"ergo.services/ergo/testing/unit"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/require"
)

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
