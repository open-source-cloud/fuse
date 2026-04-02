package repositories

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/workflow"
	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryWorkflowRepository_SaveAndFindSubWorkflowRef(t *testing.T) {
	repo := NewMemoryWorkflowRepository()
	parentID := pkgworkflow.NewID()
	childID := pkgworkflow.NewID()

	ref := &workflow.SubWorkflowRef{
		ParentWorkflowID: parentID,
		ParentThreadID:   0,
		ParentExecID:     pkgworkflow.NewExecID(0),
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
}

func TestMemoryWorkflowRepository_FindSubWorkflowRef_NotFound(t *testing.T) {
	repo := NewMemoryWorkflowRepository()

	_, err := repo.FindSubWorkflowRef("nonexistent")

	assert.Error(t, err)
}

func TestMemoryWorkflowRepository_FindActiveSubWorkflows(t *testing.T) {
	repo := NewMemoryWorkflowRepository()
	parentID := pkgworkflow.NewID()
	child1 := pkgworkflow.NewID()
	child2 := pkgworkflow.NewID()
	otherParent := pkgworkflow.NewID()
	child3 := pkgworkflow.NewID()

	_ = repo.SaveSubWorkflowRef(&workflow.SubWorkflowRef{
		ParentWorkflowID: parentID,
		ChildWorkflowID:  child1,
		ChildSchemaID:    "schema-1",
	})
	_ = repo.SaveSubWorkflowRef(&workflow.SubWorkflowRef{
		ParentWorkflowID: parentID,
		ChildWorkflowID:  child2,
		ChildSchemaID:    "schema-2",
	})
	_ = repo.SaveSubWorkflowRef(&workflow.SubWorkflowRef{
		ParentWorkflowID: otherParent,
		ChildWorkflowID:  child3,
		ChildSchemaID:    "schema-3",
	})

	children, err := repo.FindActiveSubWorkflows(parentID.String())
	require.NoError(t, err)
	assert.Len(t, children, 2)
}

func TestMemoryWorkflowRepository_FindActiveSubWorkflows_Empty(t *testing.T) {
	repo := NewMemoryWorkflowRepository()

	children, err := repo.FindActiveSubWorkflows("no-parent")
	require.NoError(t, err)
	assert.Empty(t, children)
}
