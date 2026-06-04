package packages

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// internalFnMetadata is a minimal Internal-transport function metadata.
func internalFnMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			Parameters: make([]workflow.ParameterSchema, 0),
			Edges:      workflow.InputEdgeMetadata{Parameters: make([]workflow.ParameterSchema, 0)},
		},
		Output: workflow.OutputMetadata{
			Parameters: make([]workflow.ParameterSchema, 0),
			Edges:      make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// A package decoded from persistence loses its code-backed function (PackagedFunction.Function
// is json:"-") while keeping Transport=Internal in metadata. Executing such a function must
// return an error rather than panic the worker — this is what previously left workflows stuck
// "running" on an HA node that lost the internal-package Save race. See ADR-0018 / package_service.
func TestExecuteFunction_PersistenceDecodedInternalFunc_DoesNotPanic(t *testing.T) {
	t.Parallel()

	fnID := "noop"
	pkg := workflow.NewPackage("fuse/pkg/test",
		workflow.NewFunction(fnID, internalFnMetadata(), func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResultSuccess(), nil
		}),
	)

	// Round-trip through persistence: the function pointer is dropped, metadata is kept.
	data, err := pkg.Encode()
	require.NoError(t, err)
	decoded, err := workflow.DecodePackage(data)
	require.NoError(t, err)
	require.Nil(t, decoded.Functions[0].Function, "func pointer must be lost after decode")

	loaded := MapToRegistryPackage(decoded)
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", nil)

	require.NotPanics(t, func() {
		_, execErr := loaded.ExecuteFunction(nil, "fuse/pkg/test/"+fnID, execInfo)
		assert.Error(t, execErr, "executing a func-less internal function must error, not panic")
	})
}

// The same package registered directly from code (function pointer intact) executes normally.
func TestExecuteFunction_CodeBackedInternalFunc_Runs(t *testing.T) {
	t.Parallel()

	fnID := "noop"
	pkg := workflow.NewPackage("fuse/pkg/test",
		workflow.NewFunction(fnID, internalFnMetadata(), func(*workflow.ExecutionInfo) (workflow.FunctionResult, error) {
			return workflow.NewFunctionResultSuccess(), nil
		}),
	)

	loaded := MapToRegistryPackage(pkg)
	execInfo := workflow.NewExecutionInfo("wf-1", workflow.NewExecID(1), "", nil)

	res, err := loaded.ExecuteFunction(nil, "fuse/pkg/test/"+fnID, execInfo)
	require.NoError(t, err)
	assert.Equal(t, workflow.FunctionSuccess, res.Output.Status)
}
