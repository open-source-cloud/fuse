package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensureGraphFn is an optional callback that saves the graph schema to the GraphRepository.
// For memory repos this is nil; for Postgres it saves the schema so Get() can reconstruct.
type ensureGraphFn func(t *testing.T, wf *internalworkflow.Workflow)

func contractTestWorkflowRepository(t *testing.T, newRepo func() repositories.WorkflowRepository, ensureGraph ensureGraphFn, reset func()) {
	t.Helper()

	saveWf := func(t *testing.T, repo repositories.WorkflowRepository, wf *internalworkflow.Workflow) {
		t.Helper()
		if ensureGraph != nil {
			ensureGraph(t, wf)
		}
		require.NoError(t, repo.Save(wf))
	}

	t.Run("Save and Get returns workflow with same ID and state", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		wf.SetState(internalworkflow.StateRunning)

		saveWf(t, repo, wf)
		found, err := repo.Get(wf.ID().String())

		require.NoError(t, err)
		assert.Equal(t, wf.ID(), found.ID())
		assert.Equal(t, internalworkflow.StateRunning, found.State())
	})

	t.Run("Exists returns true for saved workflow", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		saveWf(t, repo, wf)

		assert.True(t, repo.Exists(wf.ID().String()))
	})

	t.Run("Exists returns false for unknown workflow", func(t *testing.T) {
		reset()
		repo := newRepo()
		assert.False(t, repo.Exists("nonexistent-wf"))
	})

	t.Run("FindByState returns matching workflows", func(t *testing.T) {
		reset()
		repo := newRepo()

		wf1 := newTestWorkflow(t)
		wf1.SetState(internalworkflow.StateRunning)
		saveWf(t, repo, wf1)

		wf2 := newTestWorkflow(t)
		wf2.SetState(internalworkflow.StateSleeping)
		saveWf(t, repo, wf2)

		wf3 := newTestWorkflow(t)
		wf3.SetState(internalworkflow.StateFinished)
		saveWf(t, repo, wf3)

		ids, err := repo.FindByState(internalworkflow.StateRunning, internalworkflow.StateSleeping)
		require.NoError(t, err)
		assert.Contains(t, ids, wf1.ID().String())
		assert.Contains(t, ids, wf2.ID().String())
		assert.NotContains(t, ids, wf3.ID().String())
	})

	t.Run("FindByState returns empty for no matches", func(t *testing.T) {
		reset()
		repo := newRepo()
		ids, err := repo.FindByState(internalworkflow.StateCancelled)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("Save overwrites existing workflow state", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		wf.SetState(internalworkflow.StateRunning)
		saveWf(t, repo, wf)

		wf.SetState(internalworkflow.StateFinished)
		require.NoError(t, repo.Save(wf))

		found, err := repo.Get(wf.ID().String())
		require.NoError(t, err)
		assert.Equal(t, internalworkflow.StateFinished, found.State())
	})

	// --- Snapshot Ref ---

	t.Run("GetSnapshotRef returns empty for new workflow", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		saveWf(t, repo, wf)

		ref, err := repo.GetSnapshotRef(wf.ID().String())
		require.NoError(t, err)
		assert.Empty(t, ref)
	})

	t.Run("SetSnapshotRef and GetSnapshotRef round-trip", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		saveWf(t, repo, wf)

		require.NoError(t, repo.SetSnapshotRef(wf.ID().String(), "snapshots/wf-123/v1"))

		ref, err := repo.GetSnapshotRef(wf.ID().String())
		require.NoError(t, err)
		assert.Equal(t, "snapshots/wf-123/v1", ref)
	})

	t.Run("SetSnapshotRef overwrites previous ref", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		saveWf(t, repo, wf)

		require.NoError(t, repo.SetSnapshotRef(wf.ID().String(), "v1"))
		require.NoError(t, repo.SetSnapshotRef(wf.ID().String(), "v2"))

		ref, err := repo.GetSnapshotRef(wf.ID().String())
		require.NoError(t, err)
		assert.Equal(t, "v2", ref)
	})

	// --- FindExecutions ---

	t.Run("FindExecutions returns paginated results", func(t *testing.T) {
		reset()
		repo := newRepo()

		for i := range 3 {
			wf := newTestWorkflow(t)
			wf.SetState(internalworkflow.StateRunning)
			saveWf(t, repo, wf)
			_ = i
		}

		result, err := repo.FindExecutions(repositories.ExecutionListFilter{
			SchemaID: "test",
			Page:     1,
			Size:     2,
		})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 3, result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 2, result.Size)
		assert.Equal(t, 2, result.LastPage)
	})

	t.Run("FindExecutions filters by status", func(t *testing.T) {
		reset()
		repo := newRepo()

		wf1 := newTestWorkflow(t)
		wf1.SetState(internalworkflow.StateRunning)
		saveWf(t, repo, wf1)

		wf2 := newTestWorkflow(t)
		wf2.SetState(internalworkflow.StateFinished)
		saveWf(t, repo, wf2)

		result, err := repo.FindExecutions(repositories.ExecutionListFilter{
			SchemaID: "test",
			Status:   "running",
			Page:     1,
			Size:     10,
		})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, internalworkflow.StateRunning.String(), result.Items[0].State)
	})

	t.Run("FindExecutions returns empty for nonexistent schema", func(t *testing.T) {
		reset()
		repo := newRepo()

		result, err := repo.FindExecutions(repositories.ExecutionListFilter{
			SchemaID: "nonexistent-schema",
			Page:     1,
			Size:     10,
		})
		require.NoError(t, err)
		assert.Empty(t, result.Items)
		assert.Equal(t, 0, result.Total)
	})

	t.Run("FindExecutions defaults page and size", func(t *testing.T) {
		reset()
		repo := newRepo()
		wf := newTestWorkflow(t)
		saveWf(t, repo, wf)

		result, err := repo.FindExecutions(repositories.ExecutionListFilter{
			SchemaID: "test",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.Size)
	})
}

func contractTestWorkflowSubWorkflowRefs(t *testing.T, newRepo func() repositories.WorkflowRepository, ensureGraph ensureGraphFn) {
	t.Helper()

	saveWf := func(t *testing.T, repo repositories.WorkflowRepository, wf *internalworkflow.Workflow) {
		t.Helper()
		if ensureGraph != nil {
			ensureGraph(t, wf)
		}
		require.NoError(t, repo.Save(wf))
	}

	t.Run("SaveSubWorkflowRef and FindSubWorkflowRef round-trip", func(t *testing.T) {
		repo := newRepo()
		parentID := workflow.NewID()
		childID := workflow.NewID()

		parentWf := newTestWorkflowWithID(t, parentID)
		saveWf(t, repo, parentWf)

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
		saveWf(t, repo, parentWf)

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
	contractTestWorkflowRepository(t, repositories.NewMemoryWorkflowRepository, nil, func() {})
}

func TestMemoryWorkflowRepository_SubWorkflow_Contract(t *testing.T) {
	contractTestWorkflowSubWorkflowRefs(t, repositories.NewMemoryWorkflowRepository, nil)
}
