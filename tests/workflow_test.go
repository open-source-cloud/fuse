package tests

import (
	"context"
	"os"
	"testing"

	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/logic"
	"github.com/open-source-cloud/fuse/pkg/strproc"
	"github.com/stretchr/testify/suite"
)

type WorkflowTestSuite struct {
	suite.Suite
	engine    *workflow.DefaultEngine
	providers map[string]workflow.NodeProvider
}

func (s *WorkflowTestSuite) SetupTest() {
	s.engine = workflow.NewDefaultEngine()
	s.providers = map[string]workflow.NodeProvider{
		"string": strproc.NewStringProcessorProvider(),
		"logic":  logic.NewLogicProcessorProvider(),
	}
}

func (s *WorkflowTestSuite) TestStringWorkflow() {
	// Create a temporary YAML file
	yamlContent := `
id: "string-workflow"
name: "String Processing Workflow"
description: "A workflow that demonstrates string processing operations"

nodes:
  - id: "input"
    type: "string_processor"
    provider: "string"
    config:
      operation: "uppercase"
      input: "Hello, World!"

  - id: "process"
    type: "string_processor"
    provider: "string"
    config:
      operation: "lowercase"
      input: "${input.output}"

  - id: "output"
    type: "string_processor"
    provider: "string"
    config:
      operation: "trim"
      input: "${process.output}"

edges:
  - from: "input"
    to: "process"
  - from: "process"
    to: "output"
`
	tmpfile, err := os.CreateTemp("", "test-string-workflow-*.yaml")
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

	// Load workflow
	wf, err := workflow.LoadWorkflowFromYAML(tmpfile.Name())
	s.Require().NoError(err)
	s.Require().NotNil(wf)

	// Convert to internal workflow
	workflow, err := workflow.ConvertYAMLToWorkflow(wf, s.providers)
	s.Require().NoError(err)
	s.Require().NotNil(workflow)

	// Execute workflow
	result, err := s.engine.ExecuteWorkflow(context.TODO(), workflow, nil)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Verify result
	s.Equal("hello, world!", result)
}

func (s *WorkflowTestSuite) TestLogicalWorkflow() {
	// Create a temporary YAML file
	yamlContent := `
id: "logical-workflow"
name: "Logical Operations Workflow"
description: "A workflow that demonstrates logical operations"

nodes:
  - id: "and"
    type: "and"
    provider: "logic"
    config:
      type: "and"
      values:
        - true
        - false

  - id: "or"
    type: "or"
    provider: "logic"
    config:
      type: "or"
      values:
        - true
        - false

  - id: "output"
    type: "string_processor"
    provider: "string"
    config:
      operation: "uppercase"
      input: "${or.output}"

edges:
  - from: "and"
    to: "or"
  - from: "or"
    to: "output"
`
	tmpfile, err := os.CreateTemp("", "test-logical-workflow-*.yaml")
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

	// Load workflow
	wf, err := workflow.LoadWorkflowFromYAML(tmpfile.Name())
	s.Require().NoError(err)
	s.Require().NotNil(wf)

	// Convert to internal workflow
	workflow, err := workflow.ConvertYAMLToWorkflow(wf, s.providers)
	s.Require().NoError(err)
	s.Require().NotNil(workflow)

	// Execute workflow
	result, err := s.engine.ExecuteWorkflow(context.TODO(), workflow, nil)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Verify result
	s.Equal(true, result)
}

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}
