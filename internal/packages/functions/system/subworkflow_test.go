package system

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/stretchr/testify/assert"
)

func TestSubWorkflowFunctionMetadata_Transport(t *testing.T) {
	meta := SubWorkflowFunctionMetadata()
	assert.Equal(t, transport.Internal, meta.Transport)
}

func TestSubWorkflowFunctionMetadata_InputParameters(t *testing.T) {
	meta := SubWorkflowFunctionMetadata()

	assert.Len(t, meta.Input.Parameters, 3)

	schemaParam := meta.Input.Parameters[0]
	assert.Equal(t, "schemaId", schemaParam.Name)
	assert.Equal(t, "string", schemaParam.Type)
	assert.True(t, schemaParam.Required)

	inputParam := meta.Input.Parameters[1]
	assert.Equal(t, "input", inputParam.Name)
	assert.Equal(t, "map", inputParam.Type)
	assert.False(t, inputParam.Required)

	asyncParam := meta.Input.Parameters[2]
	assert.Equal(t, "async", asyncParam.Name)
	assert.Equal(t, "bool", asyncParam.Type)
	assert.False(t, asyncParam.Required)
}

func TestSubWorkflowFunctionMetadata_OutputParameters(t *testing.T) {
	meta := SubWorkflowFunctionMetadata()

	assert.Len(t, meta.Output.Parameters, 3)
	assert.Equal(t, "workflowId", meta.Output.Parameters[0].Name)
	assert.Equal(t, "status", meta.Output.Parameters[1].Name)
	assert.Equal(t, "output", meta.Output.Parameters[2].Name)
}

func TestSubWorkflowFunction_ReturnsSuccess(t *testing.T) {
	result, err := SubWorkflowFunction(nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSubWorkflowFullFunctionID(t *testing.T) {
	assert.Equal(t, "system/subworkflow", SubWorkflowFullFunctionID)
}
