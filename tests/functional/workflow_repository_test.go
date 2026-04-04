package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestWorkflowRepository(t *testing.T, newRepo func() repositories.WorkflowRepository) {
	t.Helper()

	t.Run("Save and Get returns workflow with same ID and state", func(t *testing.T) {
		repo := newRepo()
		wf := newTestWorkflow(t)
		wf.SetState(internalworkflow.StateRunning)

		require.NoError(t, repo.Save(wf))
		found, err := repo.Get(wf.ID().String())

		require.NoError(t, err)
		assert.Equal(t, wf.ID(), found.ID())
		assert.Equal(t, internalworkflow.StateRunning, found.State())
	})

	t.Run("Exists returns true for saved workflow", func(t *testing.T) {
		repo := newRepo()
		wf := newTestWorkflow(t)
		require.NoError(t, repo.Save(wf))

		assert.True(t, repo.Exists(wf.ID().String()))
	})

	t.Run("Exists returns false for unknown workflow", func(t *testing.T) {
		repo := newRepo()
		assert.False(t, repo.Exists("nonexistent-wf"))
	})

	t.Run("FindByState returns matching workflows", func(t *testing.T) {
		repo := newRepo()

		wf1 := newTestWorkflow(t)
		wf1.SetState(internalworkflow.StateRunning)
		require.NoError(t, repo.Save(wf1))

		wf2 := newTestWorkflow(t)
		wf2.SetState(internalworkflow.StateSleeping)
		require.NoError(t, repo.Save(wf2))

		wf3 := newTestWorkflow(t)
		wf3.SetState(internalworkflow.StateFinished)
		require.NoError(t, repo.Save(wf3))

		ids, err := repo.FindByState(internalworkflow.StateRunning, internalworkflow.StateSleeping)
		require.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, wf1.ID().String())
		assert.Contains(t, ids, wf2.ID().String())
	})

	t.Run("FindByState returns empty for no matches", func(t *testing.T) {
		repo := newRepo()
		ids, err := repo.FindByState(internalworkflow.StateCancelled)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("Save overwrites existing workflow state", func(t *testing.T) {
		repo := newRepo()
		wf := newTestWorkflow(t)
		wf.SetState(internalworkflow.StateRunning)
		require.NoError(t, repo.Save(wf))

		wf.SetState(internalworkflow.StateFinished)
		require.NoError(t, repo.Save(wf))

		found, err := repo.Get(wf.ID().String())
		require.NoError(t, err)
		assert.Equal(t, internalworkflow.StateFinished, found.State())
	})
}

func contractTestWorkflowSubWorkflowRefs(t *testing.T, newRepo func() repositories.WorkflowRepository) {
	t.Helper()

	t.Run("SaveSubWorkflowRef and FindSubWorkflowRef round-trip", func(t *testing.T) {
		repo := newRepo()
		parentID := workflow.NewID()
		childID := workflow.NewID()

		parentWf := newTestWorkflowWithID(t, parentID)
		require.NoError(t, repo.Save(parentWf))

		ref := &internalworkflow.SubWorkflowRef{
			ParentWorkflowID: parentID,
			ParentThreadID:   0,
			ParentExecID:     workflow.NewExecID(0),
			ChildWorkflowID:  childID,
			ChildSchemaID:    "child-schema",
			Async:            false,
		}

		err := repo.SaveSubWorkflowRef(ref)
		require.NoError(t, err)

		found, err := repo.FindSubWorkflowRef(childID.String())
		require.NoError(t, err)
		assert.Equal(t, parentID, found.ParentWorkflowID)
		assert.Equal(t, childID, found.ChildWorkflowID)
		assert.Equal(t, "child-schema", found.ChildSchemaID)
	})

	t.Run("FindSubWorkflowRef returns error for nonexistent child", func(t *testing.T) {
		repo := newRepo()
		s, err := repo.FindSubWorkflowRef("nonexistent")
		require.Nil(t, s)
		assert.Error(t, err)
	})

	t.Run("FindActiveSubWorkflows returns children of parent", func(t *testing.T) {
		repo := newRepo()
		parentID := workflow.NewID()
		parentWf := newTestWorkflowWithID(t, parentID)
		require.NoError(t, repo.Save(parentWf))

		child1 := workflow.NewID()
		child2 := workflow.NewID()
		err := repo.SaveSubWorkflowRef(&internalworkflow.SubWorkflowRef{
			ParentWorkflowID: parentID,
			ChildWorkflowID:  child1,
			ChildSchemaID:    "schema-1",
		})
		require.NoError(t, err)
		err = repo.SaveSubWorkflowRef(&internalworkflow.SubWorkflowRef{
			ParentWorkflowID: parentID,
			ChildWorkflowID:  child2,
			ChildSchemaID:    "schema-2",
		})
		require.NoError(t, err)

		children, err := repo.FindActiveSubWorkflows(parentID.String())
		require.NoError(t, err)
		assert.Len(t, children, 2)
	})

	t.Run("FindActiveSubWorkflows returns empty for unknown parent", func(t *testing.T) {
		repo := newRepo()
		children, err := repo.FindActiveSubWorkflows("no-parent")
		require.NoError(t, err)
		assert.Empty(t, children)
	})
}

func TestMemoryWorkflowRepository_Contract(t *testing.T) {
	contractTestWorkflowRepository(t, repositories.NewMemoryWorkflowRepository)
}

func TestMemoryWorkflowRepository_SubWorkflow_Contract(t *testing.T) {
	contractTestWorkflowSubWorkflowRefs(t, repositories.NewMemoryWorkflowRepository)
}
