package workflow

import (
	"fmt"
	"maps"

	"github.com/expr-lang/expr"
	"github.com/open-source-cloud/fuse/pkg/store"
)

// EvaluateCondition evaluates an edge condition against the current workflow state
func EvaluateCondition(condition *EdgeCondition, aggregatedOutput *store.KV, currentNode *Node) (bool, error) {
	switch condition.Type {
	case ConditionExpression:
		return evaluateExpression(condition.Expression, aggregatedOutput, currentNode)
	case ConditionDefault:
		return true, nil
	case ConditionExact:
		return evaluateExactCondition(condition, aggregatedOutput, currentNode), nil
	default:
		// Empty type = legacy exact match (backward compatible)
		return evaluateExactCondition(condition, aggregatedOutput, currentNode), nil
	}
}

func evaluateExactCondition(condition *EdgeCondition, aggregatedOutput *store.KV, currentNode *Node) bool {
	conditionalSource := currentNode.FunctionMetadata().Output.ConditionalOutputField
	conditionalValue := aggregatedOutput.Get(fmt.Sprintf("%s.%s", currentNode.ID(), conditionalSource))
	return condition.Value == conditionalValue
}

func evaluateExpression(expression string, aggregatedOutput *store.KV, currentNode *Node) (bool, error) {
	// Build environment with all node outputs as top-level keys
	env := make(map[string]any)
	maps.Copy(env, aggregatedOutput.Raw())

	// Add node-scoped shorthand: "output" maps to the current node's output data
	if nodeData, ok := aggregatedOutput.Raw()[currentNode.ID()]; ok {
		if nodeMap, ok := nodeData.(map[string]any); ok {
			env["output"] = nodeMap
		}
	}
	if _, exists := env["output"]; !exists {
		env["output"] = make(map[string]any)
	}

	// Compile and evaluate
	program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("failed to compile expression %q: %w", expression, err)
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression %q: %w", expression, err)
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("expression %q did not return bool, got %T", expression, result)
	}
	return boolResult, nil
}
