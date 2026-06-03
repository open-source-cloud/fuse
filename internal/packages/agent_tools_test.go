package packages

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/functions/ai"
	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/internal/packages/functions/system"
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRegistry builds an isolated registry (not the NewPackageRegistry global)
// populated with the real system + logic packages and a fake ai package.
func newTestRegistry(t *testing.T) Registry {
	t.Helper()
	reg := &MemoryRegistry{packages: make(map[string]*LoadedPackage)}
	reg.Register(system.New())
	reg.Register(logic.New())

	// A stand-in ai package so we can assert the whole ai package is excluded
	// without constructing the real one (which needs a provider + tool registry).
	fakeAi := workflow.NewPackage(ai.PackageID,
		workflow.NewFunction("chat", workflow.FunctionMetadata{
			Transport: transport.Internal,
			Input:     workflow.InputMetadata{CustomParameters: false},
		}, func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResultSuccess(), nil
		}),
	)
	reg.Register(fakeAi)
	return reg
}

func toolIDSet(tools []ai.ToolDescriptor) map[string]ai.ToolDescriptor {
	out := make(map[string]ai.ToolDescriptor, len(tools))
	for _, tl := range tools {
		out[tl.FunctionID] = tl
	}
	return out
}

func TestListTools_IncludesSyncSchemaFunctionsOnly(t *testing.T) {
	t.Parallel()

	adapter := NewAgentToolRegistry(newTestRegistry(t))
	byID := toolIDSet(adapter.ListTools())

	// Included: synchronous, declared-parameter functions.
	assert.Contains(t, byID, "fuse/pkg/logic/sum")
	assert.Contains(t, byID, "fuse/pkg/logic/rand")

	// Excluded: intercepted system functions.
	assert.NotContains(t, byID, system.SleepFullFunctionID)
	assert.NotContains(t, byID, system.WaitFullFunctionID)
	assert.NotContains(t, byID, system.SubWorkflowFullFunctionID)
	assert.NotContains(t, byID, system.ForEachFullFunctionID)
	// Excluded: async (timer) and schemaless (if), and the whole ai package.
	assert.NotContains(t, byID, "fuse/pkg/logic/timer")
	assert.NotContains(t, byID, "fuse/pkg/logic/if")
	assert.NotContains(t, byID, "fuse/pkg/ai/chat")
}

func TestListTools_DescriptorShape(t *testing.T) {
	t.Parallel()

	adapter := NewAgentToolRegistry(newTestRegistry(t))
	sum, ok := toolIDSet(adapter.ListTools())["fuse/pkg/logic/sum"]
	require.True(t, ok)

	assert.Equal(t, "fuse__pkg__logic__sum", sum.MangledName)
	assert.Equal(t, "object", sum.Parameters["type"])
	props, ok := sum.Parameters["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "values")
}

func TestInvokeTool_RunsSyncFunctionInline(t *testing.T) {
	t.Parallel()

	adapter := NewAgentToolRegistry(newTestRegistry(t))
	input, err := workflow.NewFunctionInputWith(map[string]any{"values": []float64{2, 3}})
	require.NoError(t, err)
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", input)

	res, err := adapter.InvokeTool("fuse/pkg/logic/sum", execInfo)
	require.NoError(t, err)
	assert.False(t, res.Async, "a synchronous tool returns its result inline")
	assert.Equal(t, workflow.FunctionSuccess, res.Output.Status)
	assert.InDelta(t, 5.0, res.Output.Data["sum"], 0.0001)
}

func TestInvokeTool_UnknownFunctionReturnsError(t *testing.T) {
	t.Parallel()

	adapter := NewAgentToolRegistry(newTestRegistry(t))
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", nil)

	_, err := adapter.InvokeTool("does/not/exist", execInfo)
	require.Error(t, err)
}

func TestIsExposableTool_Predicate(t *testing.T) {
	t.Parallel()

	intl := &LoadedFunction{
		ID:        "fuse/pkg/logic/sum",
		Metadata:  &FunctionMetadata{Transport: transport.Internal},
		Transport: transport.NewInternalFunctionTransport(func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) { return workflow.NewFunctionResultSuccess(), nil }),
	}
	assert.True(t, isExposableTool("fuse/pkg/logic/sum", intl))

	// schemaless
	custom := &LoadedFunction{
		ID:        "fuse/pkg/logic/if",
		Metadata:  &FunctionMetadata{Transport: transport.Internal, Input: FunctionInputMetadata{CustomParameters: true}},
		Transport: transport.NewInternalFunctionTransport(func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) { return workflow.NewFunctionResultSuccess(), nil }),
	}
	assert.False(t, isExposableTool("fuse/pkg/logic/if", custom))

	// intercepted/async denylist
	assert.False(t, isExposableTool(system.SleepFullFunctionID, intl))

	// non-invocable (nil transport)
	noTransport := &LoadedFunction{ID: "x", Metadata: &FunctionMetadata{Transport: transport.Internal}}
	assert.False(t, isExposableTool("x", noTransport))
}
