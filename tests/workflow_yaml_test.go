package tests

import (
	"os"
	"testing"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/logic"
	"github.com/open-source-cloud/fuse/pkg/strproc"
	"github.com/stretchr/testify/suite"
)

type WorkflowYAMLTestSuite struct {
	suite.Suite
	providers map[string]engine.NodeProvider
}

func (s *WorkflowYAMLTestSuite) SetupTest() {
	s.providers = map[string]engine.NodeProvider{
		"string": strproc.NewStringProcessorProvider(),
		"logic":  debug.NewLogicProcessorProvider(),
	}
}

func (s *WorkflowYAMLTestSuite) TestLoadValidYAML() {
	// Create a temporary YAML file
	yamlContent := `
id: "test-workflow"
name: "Test Workflow"
description: "A test workflow"
nodes:
  - id: "node1"
    type: "string_processor"
    provider: "string"
    config:
      operation: "uppercase"
      input: "hello"
edges:
  - from: "node1"
    to: "node1"
`
	tmpfile, err := os.CreateTemp("", "test-workflow-*.yaml")
	s.Require().NoError(err)
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			s.T().Logf("Failed to remove temporary file: %v", err)
		}
	}()

	_, err = tmpfile.Write([]byte(yamlContent))
	s.Require().NoError(err)
	if err := tmpfile.Close(); err != nil {
		s.T().Logf("Failed to close temporary file: %v", err)
	}

	// Load the workflow
	ywf, err := engine.LoadWorkflowFromYAML(tmpfile.Name())
	s.Require().NoError(err)
	s.Require().NotNil(ywf)
	s.Equal("test-workflow", ywf.ID)
	s.Equal("Test Workflow", ywf.Name)
	s.Equal("A test workflow", ywf.Description)
	s.Len(ywf.Nodes, 1)
	s.Len(ywf.Edges, 1)

	// Convert to workflow
	wf, err := engine.ConvertYAMLToWorkflow(ywf, s.providers)
	s.Require().NoError(err)
	s.Require().NotNil(wf)
	s.Equal("test-workflow", wf.ID)
	s.Equal("Test Workflow", wf.Name)
	s.Equal("A test workflow", wf.Description)
	s.Len(wf.Nodes, 1)
	s.Len(wf.Edges, 1)
}

func (s *WorkflowYAMLTestSuite) TestLoadInvalidYAML() {
	// Create a temporary YAML file with invalid content
	yamlContent := `
id: "test-workflow"
name: "Test Workflow"
nodes:
  - id: "node1"
    type: "string_processor"
    provider: "invalid-provider"
    config:
      operation: "uppercase"
      input: "hello"
`
	tmpfile, err := os.CreateTemp("", "test-workflow-*.yaml")
	s.Require().NoError(err)
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			s.T().Logf("Failed to remove temporary file: %v", err)
		}
	}()

	_, err = tmpfile.Write([]byte(yamlContent))
	s.Require().NoError(err)
	if err := tmpfile.Close(); err != nil {
		s.T().Logf("Failed to close temporary file: %v", err)
	}

	// Load the workflow
	ywf, err := engine.LoadWorkflowFromYAML(tmpfile.Name())
	s.Require().NoError(err)
	s.Require().NotNil(ywf)

	// Convert to workflow should fail due to invalid provider
	wf, err := engine.ConvertYAMLToWorkflow(ywf, s.providers)
	s.Require().Error(err)
	s.Nil(wf)
}

func (s *WorkflowYAMLTestSuite) TestLoadNonExistentFile() {
	ywf, err := engine.LoadWorkflowFromYAML("non-existent.yaml")
	s.Require().Error(err)
	s.Nil(ywf)
}

func TestWorkflowYAMLSuite(t *testing.T) {
	suite.Run(t, new(WorkflowYAMLTestSuite))
}
