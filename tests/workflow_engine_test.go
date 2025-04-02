package tests

import (
	"context"
	"testing"

	"github.com/gustavobertoi/core-workflow-poc/pkg/logic"
	"github.com/gustavobertoi/core-workflow-poc/pkg/strproc"
	"github.com/gustavobertoi/core-workflow-poc/workflow"
	"github.com/stretchr/testify/suite"
)

type WorkflowEngineTestSuite struct {
	suite.Suite
	engine    *workflow.DefaultEngine
	providers map[string]workflow.NodeProvider
}

func (s *WorkflowEngineTestSuite) SetupTest() {
	s.engine = workflow.NewDefaultEngine()
	s.providers = map[string]workflow.NodeProvider{
		"string": strproc.NewStringProcessorProvider(),
		"logic":  logic.NewLogicProcessorProvider(),
	}
}

func (s *WorkflowEngineTestSuite) TestExecuteEmptyWorkflow() {
	wf := &workflow.Workflow{
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

	wf := &workflow.Workflow{
		ID:    "single-node-workflow",
		Name:  "Single Node Workflow",
		Nodes: []workflow.Node{node, outputNode},
		Edges: []workflow.Edge{
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

	wf := &workflow.Workflow{
		ID:    "invalid-node-workflow",
		Name:  "Invalid Node Workflow",
		Nodes: []workflow.Node{node, outputNode},
		Edges: []workflow.Edge{
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
