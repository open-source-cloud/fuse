package repositories

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/workflow"
	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAwakeable(id string, workflowID pkgworkflow.ID) *workflow.Awakeable {
	return &workflow.Awakeable{
		ID:         id,
		WorkflowID: workflowID,
		ExecID:     pkgworkflow.NewExecID(0),
		ThreadID:   0,
		CreatedAt:  time.Now(),
		Timeout:    30 * time.Second,
		DeadlineAt: time.Now().Add(30 * time.Second),
		Status:     workflow.AwakeablePending,
	}
}

func TestMemoryAwakeableRepository_SaveAndFindByID(t *testing.T) {
	repo := NewMemoryAwakeableRepository()
	wfID := pkgworkflow.NewID()
	awakeable := newTestAwakeable("awk-1", wfID)

	err := repo.Save(awakeable)
	require.NoError(t, err)

	found, err := repo.FindByID("awk-1")
	require.NoError(t, err)
	assert.Equal(t, "awk-1", found.ID)
	assert.Equal(t, wfID, found.WorkflowID)
	assert.Equal(t, workflow.AwakeablePending, found.Status)
}

func TestMemoryAwakeableRepository_FindByID_NotFound(t *testing.T) {
	repo := NewMemoryAwakeableRepository()

	_, err := repo.FindByID("nonexistent")

	assert.ErrorIs(t, err, ErrAwakeableNotFound)
}

func TestMemoryAwakeableRepository_FindPending(t *testing.T) {
	repo := NewMemoryAwakeableRepository()
	wfID := pkgworkflow.NewID()
	otherWfID := pkgworkflow.NewID()

	_ = repo.Save(newTestAwakeable("awk-1", wfID))
	_ = repo.Save(newTestAwakeable("awk-2", wfID))
	_ = repo.Save(newTestAwakeable("awk-3", otherWfID))

	// Resolve one to make sure it's not returned
	_ = repo.Resolve("awk-2", map[string]any{"done": true})

	pending, err := repo.FindPending(wfID.String())
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "awk-1", pending[0].ID)
}

func TestMemoryAwakeableRepository_Resolve(t *testing.T) {
	repo := NewMemoryAwakeableRepository()
	wfID := pkgworkflow.NewID()
	_ = repo.Save(newTestAwakeable("awk-1", wfID))

	err := repo.Resolve("awk-1", map[string]any{"approved": true})
	require.NoError(t, err)

	found, err := repo.FindByID("awk-1")
	require.NoError(t, err)
	assert.Equal(t, workflow.AwakeableResolved, found.Status)
	assert.Equal(t, true, found.Result["approved"])
}

func TestMemoryAwakeableRepository_Resolve_NotFound(t *testing.T) {
	repo := NewMemoryAwakeableRepository()

	err := repo.Resolve("nonexistent", map[string]any{})

	assert.ErrorIs(t, err, ErrAwakeableNotFound)
}
