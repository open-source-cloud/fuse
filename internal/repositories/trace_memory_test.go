package repositories

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTrace(workflowID, schemaID string, status workflow.State) *workflow.ExecutionTrace {
	now := time.Now()
	return &workflow.ExecutionTrace{
		WorkflowID:  workflowID,
		SchemaID:    schemaID,
		Status:      status,
		TriggeredAt: now,
		Steps:       []workflow.ExecutionStepTrace{},
	}
}

func TestMemoryTraceRepository_SaveAndFind(t *testing.T) {
	repo := NewMemoryTraceRepository()
	trace := newTestTrace("wf-1", "schema-1", workflow.StateFinished)

	err := repo.Save(trace)
	require.NoError(t, err)

	found, err := repo.FindByWorkflowID("wf-1")
	require.NoError(t, err)
	assert.Equal(t, "wf-1", found.WorkflowID)
	assert.Equal(t, "schema-1", found.SchemaID)
	assert.Equal(t, workflow.StateFinished, found.Status)
}

func TestMemoryTraceRepository_FindNotFound(t *testing.T) {
	repo := NewMemoryTraceRepository()

	_, err := repo.FindByWorkflowID("nonexistent")
	assert.ErrorIs(t, err, ErrTraceNotFound)
}

func TestMemoryTraceRepository_SaveOverwrite(t *testing.T) {
	repo := NewMemoryTraceRepository()

	trace1 := newTestTrace("wf-1", "schema-1", workflow.StateRunning)
	_ = repo.Save(trace1)

	trace2 := newTestTrace("wf-1", "schema-1", workflow.StateFinished)
	_ = repo.Save(trace2)

	found, _ := repo.FindByWorkflowID("wf-1")
	assert.Equal(t, workflow.StateFinished, found.Status)

	// Should not duplicate in schema index
	result, _ := repo.FindBySchemaID("schema-1", TraceQueryOpts{})
	assert.Equal(t, 1, result.Total)
}

func TestMemoryTraceRepository_Delete(t *testing.T) {
	repo := NewMemoryTraceRepository()

	trace := newTestTrace("wf-1", "schema-1", workflow.StateFinished)
	_ = repo.Save(trace)

	err := repo.Delete("wf-1")
	require.NoError(t, err)

	_, err = repo.FindByWorkflowID("wf-1")
	assert.ErrorIs(t, err, ErrTraceNotFound)

	result, _ := repo.FindBySchemaID("schema-1", TraceQueryOpts{})
	assert.Equal(t, 0, result.Total)
}

func TestMemoryTraceRepository_DeleteNonexistent(t *testing.T) {
	repo := NewMemoryTraceRepository()
	err := repo.Delete("nonexistent")
	assert.NoError(t, err)
}

func TestMemoryTraceRepository_FindBySchemaID(t *testing.T) {
	repo := NewMemoryTraceRepository()

	_ = repo.Save(newTestTrace("wf-1", "schema-1", workflow.StateFinished))
	_ = repo.Save(newTestTrace("wf-2", "schema-1", workflow.StateError))
	_ = repo.Save(newTestTrace("wf-3", "schema-2", workflow.StateFinished))

	result, err := repo.FindBySchemaID("schema-1", TraceQueryOpts{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Traces, 2)
}

func TestMemoryTraceRepository_FindBySchemaID_StatusFilter(t *testing.T) {
	repo := NewMemoryTraceRepository()

	_ = repo.Save(newTestTrace("wf-1", "schema-1", workflow.StateFinished))
	_ = repo.Save(newTestTrace("wf-2", "schema-1", workflow.StateError))

	status := "error"
	result, err := repo.FindBySchemaID("schema-1", TraceQueryOpts{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "wf-2", result.Traces[0].WorkflowID)
}

func TestMemoryTraceRepository_FindBySchemaID_Pagination(t *testing.T) {
	repo := NewMemoryTraceRepository()

	for i := range 5 {
		_ = repo.Save(newTestTrace(
			"wf-"+string(rune('A'+i)),
			"schema-1",
			workflow.StateFinished,
		))
	}

	result, err := repo.FindBySchemaID("schema-1", TraceQueryOpts{Limit: 2, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Traces, 2)
}

func TestMemoryTraceRepository_FindBySchemaID_Empty(t *testing.T) {
	repo := NewMemoryTraceRepository()

	result, err := repo.FindBySchemaID("nonexistent", TraceQueryOpts{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Traces)
}

func TestMemoryTraceRepository_FindBySchemaID_SinceFilter(t *testing.T) {
	repo := NewMemoryTraceRepository()

	old := &workflow.ExecutionTrace{
		WorkflowID:  "wf-old",
		SchemaID:    "schema-1",
		Status:      workflow.StateFinished,
		TriggeredAt: time.Now().Add(-48 * time.Hour),
		Steps:       []workflow.ExecutionStepTrace{},
	}
	recent := &workflow.ExecutionTrace{
		WorkflowID:  "wf-recent",
		SchemaID:    "schema-1",
		Status:      workflow.StateFinished,
		TriggeredAt: time.Now(),
		Steps:       []workflow.ExecutionStepTrace{},
	}
	_ = repo.Save(old)
	_ = repo.Save(recent)

	since := time.Now().Add(-24 * time.Hour)
	result, err := repo.FindBySchemaID("schema-1", TraceQueryOpts{Since: &since})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "wf-recent", result.Traces[0].WorkflowID)
}
