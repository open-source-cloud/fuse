package repositories_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWorkflow(t *testing.T) *internalworkflow.Workflow {
	t.Helper()
	schema := mocks.SmallTestGraphSchema()
	graph, err := internalworkflow.NewGraph(schema)
	require.NoError(t, err)
	return internalworkflow.New(workflow.NewID(), graph)
}

func TestMemoryWorkflowRepository_FindByState(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, repo repositories.WorkflowRepository)
		states   []internalworkflow.State
		expected int
	}{
		{
			name: "empty repository returns no results",
			setup: func(_ *testing.T, _ repositories.WorkflowRepository) {
				// No setup: assert FindByState on an empty repository.
			},
			states:   []internalworkflow.State{internalworkflow.StateRunning},
			expected: 0,
		},
		{
			name: "finds running workflows",
			setup: func(t *testing.T, repo repositories.WorkflowRepository) {
				wf := newTestWorkflow(t)
				wf.SetState(internalworkflow.StateRunning)
				require.NoError(t, repo.Save(wf))
			},
			states:   []internalworkflow.State{internalworkflow.StateRunning},
			expected: 1,
		},
		{
			name: "does not return workflows in other states",
			setup: func(t *testing.T, repo repositories.WorkflowRepository) {
				wf := newTestWorkflow(t)
				wf.SetState(internalworkflow.StateFinished)
				require.NoError(t, repo.Save(wf))
			},
			states:   []internalworkflow.State{internalworkflow.StateRunning},
			expected: 0,
		},
		{
			name: "matches multiple states",
			setup: func(t *testing.T, repo repositories.WorkflowRepository) {
				wf1 := newTestWorkflow(t)
				wf1.SetState(internalworkflow.StateRunning)
				require.NoError(t, repo.Save(wf1))

				wf2 := newTestWorkflow(t)
				wf2.SetState(internalworkflow.StateSleeping)
				require.NoError(t, repo.Save(wf2))

				wf3 := newTestWorkflow(t)
				wf3.SetState(internalworkflow.StateFinished)
				require.NoError(t, repo.Save(wf3))
			},
			states:   []internalworkflow.State{internalworkflow.StateRunning, internalworkflow.StateSleeping},
			expected: 2,
		},
		{
			name: "no matching states returns empty",
			setup: func(t *testing.T, repo repositories.WorkflowRepository) {
				wf := newTestWorkflow(t)
				wf.SetState(internalworkflow.StateFinished)
				require.NoError(t, repo.Save(wf))
			},
			states:   []internalworkflow.State{internalworkflow.StateRunning, internalworkflow.StateSleeping},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := repositories.NewMemoryWorkflowRepository()
			tt.setup(t, repo)

			// Act
			ids, err := repo.FindByState(tt.states...)

			// Assert
			require.NoError(t, err)
			assert.Len(t, ids, tt.expected)
		})
	}
}
