package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_CreatesPackageWithAllFunctions(t *testing.T) {
	pkg := New()

	require.NotNil(t, pkg)
	assert.Equal(t, PackageID, pkg.ID)
	assert.Len(t, pkg.Functions, 4)
	assert.Equal(t, SleepFunctionID, pkg.Functions[0].ID)
	assert.Equal(t, WaitFunctionID, pkg.Functions[1].ID)
	assert.Equal(t, SubWorkflowFunctionID, pkg.Functions[2].ID)
	assert.Equal(t, ForEachFunctionID, pkg.Functions[3].ID)
}

func TestSleepFunctionMetadata(t *testing.T) {
	meta := SleepFunctionMetadata()

	assert.Len(t, meta.Input.Parameters, 2)
	assert.Equal(t, "duration", meta.Input.Parameters[0].Name)
	assert.True(t, meta.Input.Parameters[0].Required)
	assert.Equal(t, "reason", meta.Input.Parameters[1].Name)
	assert.False(t, meta.Input.Parameters[1].Required)
}

func TestWaitFunctionMetadata(t *testing.T) {
	meta := WaitFunctionMetadata()

	assert.Len(t, meta.Input.Parameters, 2)
	assert.Equal(t, "timeout", meta.Input.Parameters[0].Name)
	assert.False(t, meta.Input.Parameters[0].Required)
	assert.Equal(t, "filter", meta.Input.Parameters[1].Name)
}

func TestFullFunctionIDs(t *testing.T) {
	assert.Equal(t, "system/sleep", SleepFullFunctionID)
	assert.Equal(t, "system/wait", WaitFullFunctionID)
	assert.Equal(t, "system/subworkflow", SubWorkflowFullFunctionID)
	assert.Equal(t, "system/foreach", ForEachFullFunctionID)
}
