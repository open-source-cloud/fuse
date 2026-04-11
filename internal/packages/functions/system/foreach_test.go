package system

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachFunctionMetadata_Transport(t *testing.T) {
	meta := ForEachFunctionMetadata()
	assert.Equal(t, transport.Internal, meta.Transport)
}

func TestForEachFunctionMetadata_InputParameters(t *testing.T) {
	meta := ForEachFunctionMetadata()

	require.Len(t, meta.Input.Parameters, 3)

	itemsParam := meta.Input.Parameters[0]
	assert.Equal(t, "items", itemsParam.Name)
	assert.Equal(t, "[]any", itemsParam.Type)
	assert.True(t, itemsParam.Required)

	batchSizeParam := meta.Input.Parameters[1]
	assert.Equal(t, "batchSize", batchSizeParam.Name)
	assert.Equal(t, "int", batchSizeParam.Type)
	assert.False(t, batchSizeParam.Required)
	assert.Equal(t, 1, batchSizeParam.Default)

	concurrencyParam := meta.Input.Parameters[2]
	assert.Equal(t, "concurrency", concurrencyParam.Name)
	assert.Equal(t, "int", concurrencyParam.Type)
	assert.False(t, concurrencyParam.Required)
	assert.Equal(t, 1, concurrencyParam.Default)
}

func TestForEachFunctionMetadata_OutputParameters(t *testing.T) {
	meta := ForEachFunctionMetadata()

	require.Len(t, meta.Output.Parameters, 6)

	names := make([]string, 0, 6)
	for _, p := range meta.Output.Parameters {
		names = append(names, p.Name)
	}
	assert.Contains(t, names, "item")
	assert.Contains(t, names, "batch")
	assert.Contains(t, names, "index")
	assert.Contains(t, names, "total")
	assert.Contains(t, names, "isLast")
	assert.Contains(t, names, "results")
}

func TestForEachFunctionMetadata_ConditionalOutput(t *testing.T) {
	meta := ForEachFunctionMetadata()

	assert.True(t, meta.Output.ConditionalOutput)
	assert.Equal(t, "_foreach_phase", meta.Output.ConditionalOutputField)
}

func TestForEachFunctionMetadata_OutputEdges(t *testing.T) {
	meta := ForEachFunctionMetadata()

	require.Len(t, meta.Output.Edges, 2)

	eachEdge := meta.Output.Edges[0]
	assert.Equal(t, "each", eachEdge.Name)
	assert.Equal(t, "each", eachEdge.ConditionalEdge.Value)

	doneEdge := meta.Output.Edges[1]
	assert.Equal(t, "done", doneEdge.Name)
	assert.Equal(t, "done", doneEdge.ConditionalEdge.Value)
}

func TestForEachFunction_ReturnsSuccess(t *testing.T) {
	result, err := ForEachFunction(nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestForEachFullFunctionID(t *testing.T) {
	assert.Equal(t, "system/foreach", ForEachFullFunctionID)
}
