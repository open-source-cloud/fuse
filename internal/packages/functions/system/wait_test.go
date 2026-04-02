package system

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/stretchr/testify/assert"
)

func TestWaitFunctionMetadata_Transport(t *testing.T) {
	meta := WaitFunctionMetadata()
	assert.Equal(t, transport.Internal, meta.Transport)
}

func TestWaitFunctionMetadata_InputParameters(t *testing.T) {
	meta := WaitFunctionMetadata()

	assert.Len(t, meta.Input.Parameters, 2)

	timeoutParam := meta.Input.Parameters[0]
	assert.Equal(t, "timeout", timeoutParam.Name)
	assert.Equal(t, "string", timeoutParam.Type)
	assert.False(t, timeoutParam.Required)

	filterParam := meta.Input.Parameters[1]
	assert.Equal(t, "filter", filterParam.Name)
	assert.Equal(t, "string", filterParam.Type)
	assert.False(t, filterParam.Required)
}

func TestWaitFunctionMetadata_OutputParameters(t *testing.T) {
	meta := WaitFunctionMetadata()

	assert.Len(t, meta.Output.Parameters, 2)
	assert.Equal(t, "data", meta.Output.Parameters[0].Name)
	assert.Equal(t, "timedOut", meta.Output.Parameters[1].Name)
}

func TestWaitFunction_ReturnsSuccess(t *testing.T) {
	result, err := WaitFunction(nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}
