package tests

import (
	"context"
	"testing"

	"github.com/open-source-cloud/core-workflow-poc/internal/workflow"
	"github.com/open-source-cloud/core-workflow-poc/pkg/strproc"
	"github.com/stretchr/testify/suite"
)

type StringProcessorTestSuite struct {
	suite.Suite
	provider workflow.NodeProvider
}

func (s *StringProcessorTestSuite) SetupTest() {
	s.provider = strproc.NewStringProcessorProvider()
}

func (s *StringProcessorTestSuite) TestUppercase() {
	config := map[string]interface{}{
		"operation": strproc.OperationUppercase,
		"input":     "hello",
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), nil)
	s.Require().NoError(err)
	s.Equal("HELLO", result)
}

func (s *StringProcessorTestSuite) TestLowercase() {
	config := map[string]interface{}{
		"operation": strproc.OperationLowercase,
		"input":     "HELLO",
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), nil)
	s.Require().NoError(err)
	s.Equal("hello", result)
}

func (s *StringProcessorTestSuite) TestTrim() {
	config := map[string]interface{}{
		"operation": strproc.OperationTrim,
		"input":     "  hello  ",
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), nil)
	s.Require().NoError(err)
	s.Equal("hello", result)
}

func (s *StringProcessorTestSuite) TestInvalidOperation() {
	config := map[string]interface{}{
		"operation": "invalid-operation",
		"input":     "hello",
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), nil)
	s.Require().Error(err)
	s.Nil(result)
}

func (s *StringProcessorTestSuite) TestMissingInput() {
	config := map[string]interface{}{
		"operation": strproc.OperationUppercase,
	}

	node, err := s.provider.CreateNode(config)
	s.Require().Error(err)
	s.Nil(node)
}

func TestStringProcessorSuite(t *testing.T) {
	suite.Run(t, new(StringProcessorTestSuite))
}
