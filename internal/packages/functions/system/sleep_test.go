package system

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/stretchr/testify/assert"
)

func TestSleepFunctionMetadata_Transport(t *testing.T) {
	meta := SleepFunctionMetadata()
	assert.Equal(t, transport.Internal, meta.Transport)
}

func TestSleepFunctionMetadata_InputParameters(t *testing.T) {
	meta := SleepFunctionMetadata()

	assert.Len(t, meta.Input.Parameters, 2)

	durationParam := meta.Input.Parameters[0]
	assert.Equal(t, "duration", durationParam.Name)
	assert.Equal(t, "string", durationParam.Type)
	assert.True(t, durationParam.Required)

	reasonParam := meta.Input.Parameters[1]
	assert.Equal(t, "reason", reasonParam.Name)
	assert.Equal(t, "string", reasonParam.Type)
	assert.False(t, reasonParam.Required)
}

func TestSleepFunctionMetadata_OutputParameters(t *testing.T) {
	meta := SleepFunctionMetadata()

	assert.Len(t, meta.Output.Parameters, 1)
	assert.Equal(t, "sleptFor", meta.Output.Parameters[0].Name)
}

func TestSleepFunction_ReturnsSuccess(t *testing.T) {
	result, err := SleepFunction(nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}
