package debug

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// FailFunctionID is the id of the fail function
const FailFunctionID = "fail"

// failCounters tracks the number of invocations per workflow execution + node,
// allowing the function to fail N times before succeeding.
var failCounters = &failCounterStore{
	counters: make(map[string]*atomic.Int64),
}

type failCounterStore struct {
	mu       sync.Mutex
	counters map[string]*atomic.Int64
}

func (s *failCounterStore) increment(key string) int64 {
	s.mu.Lock()
	counter, ok := s.counters[key]
	if !ok {
		counter = &atomic.Int64{}
		s.counters[key] = counter
	}
	s.mu.Unlock()

	return counter.Add(1)
}

// ResetFailCounters resets all fail counters (useful for testing)
func ResetFailCounters() {
	failCounters.mu.Lock()
	failCounters.counters = make(map[string]*atomic.Int64)
	failCounters.mu.Unlock()
}

// FailFunctionMetadata returns the metadata for the fail function
func FailFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "failCount",
					Type:        "int",
					Required:    true,
					Description: "Number of times to fail before succeeding",
				},
				{
					Name:        "message",
					Type:        "string",
					Required:    false,
					Description: "Optional message to include in output",
				},
			},
			Edges: workflow.InputEdgeMetadata{
				Count:      0,
				Parameters: make([]workflow.ParameterSchema, 0),
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "result",
					Type:        "string",
					Description: "Result message after succeeding",
				},
			},
			Edges: make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// FailFunction fails the first N invocations (based on failCount input) and
// succeeds on subsequent calls. The counter is keyed by workflow execution ID
// and node, so each execution context tracks independently.
func FailFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	input := execInfo.Input

	failCount := input.GetInt("failCount")
	message := input.GetStr("message")

	// Build a key scoped to this workflow execution
	key := fmt.Sprintf("%s:%s", execInfo.WorkflowID, execInfo.ExecID)

	callNumber := failCounters.increment(key)

	if callNumber <= int64(failCount) {
		errMsg := fmt.Sprintf("intentional failure %d/%d", callNumber, failCount)
		if message != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, message)
		}
		return workflow.NewFunctionResultError(fmt.Errorf("%s", errMsg))
	}

	resultMsg := fmt.Sprintf("succeeded after %d failures", failCount)
	if message != "" {
		resultMsg = fmt.Sprintf("%s: %s", resultMsg, message)
	}

	return workflow.NewFunctionResultSuccessWith(map[string]any{
		"result": resultMsg,
	}), nil
}
