package functional_test

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestAwakeableRepository(t *testing.T, newRepo func() repositories.AwakeableRepository, wfRepo repositories.WorkflowRepository, graphRepo ...repositories.GraphRepository) {
	t.Helper()

	// seedWf ensures a workflow exists in the DB (for FK constraints).
	seedWf := func(t *testing.T, wfID workflow.ID) {
		t.Helper()
		if wfRepo != nil {
			wf := newTestWorkflowWithID(t, wfID)
			if len(graphRepo) > 0 && graphRepo[0] != nil {
				require.NoError(t, graphRepo[0].Save(wf.Graph()))
			}
			require.NoError(t, wfRepo.Save(wf))
		}
	}

	t.Run("Save and FindByID returns same awakeable", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID()
		seedWf(t, wfID)

		awk := &internalworkflow.Awakeable{
			ID:         "awk-contract-1",
			WorkflowID: wfID,
			ExecID:     workflow.NewExecID(0),
			ThreadID:   0,
			CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
			Timeout:    30 * time.Second,
			DeadlineAt: time.Now().Add(30 * time.Second).UTC().Truncate(time.Microsecond),
			Status:     internalworkflow.AwakeablePending,
		}

		require.NoError(t, repo.Save(awk))
		found, err := repo.FindByID("awk-contract-1")

		require.NoError(t, err)
		assert.Equal(t, "awk-contract-1", found.ID)
		assert.Equal(t, wfID, found.WorkflowID)
		assert.Equal(t, internalworkflow.AwakeablePending, found.Status)
	})

	t.Run("FindByID returns error for nonexistent awakeable", func(t *testing.T) {
		repo := newRepo()
		_, err := repo.FindByID("nonexistent-awk")
		assert.ErrorIs(t, err, repositories.ErrAwakeableNotFound)
	})

	t.Run("FindPending returns only pending awakeables for workflow", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID()
		otherWfID := workflow.NewID()
		seedWf(t, wfID)
		seedWf(t, otherWfID)

		err := repo.Save(&internalworkflow.Awakeable{
			ID: "awk-pending-1", WorkflowID: wfID, ExecID: workflow.NewExecID(0),
			Status: internalworkflow.AwakeablePending,
		})
		require.NoError(t, err)
		err = repo.Save(&internalworkflow.Awakeable{
			ID: "awk-pending-2", WorkflowID: wfID, ExecID: workflow.NewExecID(0),
			Status: internalworkflow.AwakeablePending,
		})
		require.NoError(t, err)
		err = repo.Save(&internalworkflow.Awakeable{
			ID: "awk-other", WorkflowID: otherWfID, ExecID: workflow.NewExecID(0),
			Status: internalworkflow.AwakeablePending,
		})
		require.NoError(t, err)

		err = repo.Resolve("awk-pending-2", map[string]any{"done": true})
		require.NoError(t, err)

		pending, err := repo.FindPending(wfID.String())
		require.NoError(t, err)
		assert.Len(t, pending, 1)
		assert.Equal(t, "awk-pending-1", pending[0].ID)
	})

	t.Run("Resolve changes status and stores result", func(t *testing.T) {
		repo := newRepo()
		wfID := workflow.NewID()
		seedWf(t, wfID)

		err := repo.Save(&internalworkflow.Awakeable{
			ID: "awk-resolve-1", WorkflowID: wfID, ExecID: workflow.NewExecID(0),
			Status: internalworkflow.AwakeablePending,
		})
		require.NoError(t, err)

		err = repo.Resolve("awk-resolve-1", map[string]any{"approved": true})
		require.NoError(t, err)

		found, err := repo.FindByID("awk-resolve-1")
		require.NoError(t, err)
		assert.Equal(t, internalworkflow.AwakeableResolved, found.Status)
		assert.Equal(t, true, found.Result["approved"])
	})

	t.Run("Resolve returns error for nonexistent awakeable", func(t *testing.T) {
		repo := newRepo()
		err := repo.Resolve("nonexistent-awk", map[string]any{})
		assert.ErrorIs(t, err, repositories.ErrAwakeableNotFound)
	})
}

func TestMemoryAwakeableRepository_Contract(t *testing.T) {
	contractTestAwakeableRepository(t, func() repositories.AwakeableRepository {
		return repositories.NewMemoryAwakeableRepository()
	}, nil)
}
