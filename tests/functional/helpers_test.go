package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func newTestWorkflow(t *testing.T) *internalworkflow.Workflow {
	t.Helper()
	schema := mocks.SmallTestGraphSchema()
	graph, err := internalworkflow.NewGraph(schema)
	require.NoError(t, err)
	return internalworkflow.New(workflow.NewID(), graph)
}

func newTestWorkflowWithID(t *testing.T, id workflow.ID) *internalworkflow.Workflow {
	t.Helper()
	schema := mocks.SmallTestGraphSchema()
	graph, err := internalworkflow.NewGraph(schema)
	require.NoError(t, err)
	return internalworkflow.New(id, graph)
}
