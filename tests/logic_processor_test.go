package tests

import (
	"context"
	"testing"

	"github.com/gustavobertoi/core-workflow-poc/internal/workflow"
	"github.com/gustavobertoi/core-workflow-poc/pkg/logic"
	"github.com/stretchr/testify/suite"
)

type LogicProcessorTestSuite struct {
	suite.Suite
	provider workflow.NodeProvider
}

func (s *LogicProcessorTestSuite) SetupTest() {
	s.provider = logic.NewLogicProcessorProvider()
}

func (s *LogicProcessorTestSuite) TestAnd() {
	config := map[string]interface{}{
		"type":   logic.NodeTypeAnd,
		"values": []interface{}{true, true},
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), true)
	s.Require().NoError(err)
	s.Equal(true, result)

	result, err = node.Execute(context.TODO(), false)
	s.Require().NoError(err)
	s.Equal(false, result)
}

func (s *LogicProcessorTestSuite) TestOr() {
	config := map[string]interface{}{
		"type":   logic.NodeTypeOr,
		"values": []interface{}{false, true},
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), false)
	s.Require().NoError(err)
	s.Equal(true, result)

	result, err = node.Execute(context.TODO(), true)
	s.Require().NoError(err)
	s.Equal(true, result)
}

func (s *LogicProcessorTestSuite) TestInvalidConfig() {
	config := map[string]interface{}{
		"invalid": "config",
	}

	node, err := s.provider.CreateNode(config)
	s.Require().Error(err)
	s.Nil(node)
}

func (s *LogicProcessorTestSuite) TestEmptyValues() {
	config := map[string]interface{}{
		"type":   logic.NodeTypeAnd,
		"values": []interface{}{},
	}

	node, err := s.provider.CreateNode(config)
	s.Require().Error(err)
	s.Nil(node)
}

func (s *LogicProcessorTestSuite) TestInvalidValueType() {
	config := map[string]interface{}{
		"type":   logic.NodeTypeAnd,
		"values": []interface{}{"not-a-boolean", "also-not-a-boolean"},
	}

	node, err := s.provider.CreateNode(config)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	result, err := node.Execute(context.TODO(), true)
	s.Require().Error(err)
	s.Nil(result)
}

func TestLogicProcessorSuite(t *testing.T) {
	suite.Run(t, new(LogicProcessorTestSuite))
}
