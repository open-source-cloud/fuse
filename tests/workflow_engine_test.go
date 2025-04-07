package tests

import (
	"context"
	"testing"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/logic"
	"github.com/open-source-cloud/fuse/pkg/strproc"
	"github.com/stretchr/testify/suite"
)

type WorkflowEngineTestSuite struct {
	suite.Suite
	engine    *engine.DefaultEngine
	providers map[string]engine.NodeProvider
}

func (s *WorkflowEngineTestSuite) SetupTest() {
	s.engine = engine.NewDefaultEngine()
	s.providers = map[string]engine.NodeProvider{
		"string": strproc.NewStringProcessorProvider(),
		"logic":  debug.NewLogicProcessorProvider(),
	}
}

func (s *WorkflowEngineTestSuite) TestExecuteEmptyWorkflow() {
	wf := &engine.Workflow{
		ID:   "empty-workflow",
		Name: "Empty Workflow",
	}

	result, err := s.engine.ExecuteWorkflow(context.TODO(), wf, nil)
	s.Require().Error(err)
	s.Equal("workflow has no nodes", err.Error())
	s.Nil(result)
}

func (s *WorkflowEngineTestSuite) TestExecuteSingleNodeWorkflow() {
	// Create a simple workflow with one node
	node, err := s.providers["string"].CreateNode(map[string]interface{}{
		"operation": strproc.OperationUppercase,
		"input":     "hello",
	})
	s.Require().NoError(err)

	// Create a dummy output node
	outputNode, err := s.providers["string"].CreateNode(map[string]interface{}{
		"operation": strproc.OperationTrim,
		"input":     "",
	})
	s.Require().NoError(err)

	wf := &engine.Workflow{
		ID:    "single-node-workflow",
		Name:  "Single GraphNode Workflow",
		Nodes: []engine.Node{node, outputNode},
		Edges: []engine.Edge{
			{
				FromNodeID: node.ID(),
				ToNodeID:   outputNode.ID(),
			},
		},
	}

	result, err := s.engine.ExecuteWorkflow(context.TODO(), wf, nil)
	s.Require().NoError(err)
	s.Equal("HELLO", result)
}

func (s *WorkflowEngineTestSuite) TestExecuteWorkflowWithInvalidNode() {
	// Create a workflow with an invalid node
	node, err := s.providers["string"].CreateNode(map[string]interface{}{
		"operation": "invalid-operation",
		"input":     "hello",
	})
	s.Require().NoError(err)
	s.Require().NotNil(node)

	// Create a dummy output node
	outputNode, err := s.providers["string"].CreateNode(map[string]interface{}{
		"operation": strproc.OperationTrim,
		"input":     "",
	})
	s.Require().NoError(err)

	wf := &engine.Workflow{
		ID:    "invalid-node-workflow",
		Name:  "Invalid GraphNode Workflow",
		Nodes: []engine.Node{node, outputNode},
		Edges: []engine.Edge{
			{
				FromNodeID: node.ID(),
				ToNodeID:   outputNode.ID(),
			},
		},
	}

	result, err := s.engine.ExecuteWorkflow(context.TODO(), wf, nil)
	s.Require().Error(err)
	s.Nil(result)
}

func TestWorkflowEngineSuite(t *testing.T) {
	suite.Run(t, new(WorkflowEngineTestSuite))
}
