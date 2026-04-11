package actors

import (
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/concurrency"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowFuncFactory redefines a WorkflowFunc worker actor factory type for better readability
type WorkflowFuncFactory ActorFactory[*WorkflowFunc]

// NewWorkflowFuncFactory creates a dependency injection factory of WorkflowFunc worker actor
func NewWorkflowFuncFactory(packageRegistry packages.Registry, concurrencyMgr *concurrency.Manager, rateLimiter *concurrency.RateLimiter) *WorkflowFuncFactory {
	return &WorkflowFuncFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowFunc{
				packageRegistry:    packageRegistry,
				concurrencyManager: concurrencyMgr,
				rateLimiter:        rateLimiter,
			}
		},
	}
}

// WorkflowFunc defines a WorkflowFunc worker actor - used as a worker actor in WorkflowFuncPool actor for processing
//
//	workflow Functions execution
type WorkflowFunc struct {
	act.Actor

	packageRegistry    packages.Registry
	concurrencyManager *concurrency.Manager
	rateLimiter        *concurrency.RateLimiter
}

// Init called when a WorkflowFunc worker actor is being initialized
func (a *WorkflowFunc) Init(_ ...any) error {
	a.Log().Debug("starting process %s", a.PID())

	return nil
}

// HandleMessage handles messages sent to a WorkflowFunc worker actor
func (a *WorkflowFunc) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return nil
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)
	jsonArgs, _ := json.Marshal(msg.Args)
	a.Log().Debug("args: %s", string(jsonArgs))

	if msg.Type != messaging.ExecuteFunction {
		a.Log().Error("message from %s is not a messaging.ExecuteFunction; got %s", from, msg.Type)
		return nil
	}
	msgPayload, err := msg.ExecuteFunctionMessage()
	if err != nil {
		a.Log().Error("failed to get execute function message from message: %s", msg)
		return nil
	}
	pkg, err := a.packageRegistry.Get(msgPayload.PackageID)
	if err != nil {
		a.Log().Error("LoadedPackage %s is not registered", msgPayload.PackageID)
		return nil
	}

	// Acquire concurrency slot if configured
	metadata, _ := pkg.GetFunctionMetadata(msgPayload.FunctionID)
	functionID := fmt.Sprintf("%s/%s", msgPayload.PackageID, msgPayload.FunctionID)
	if metadata != nil && metadata.Concurrency != nil {
		release := a.concurrencyManager.AcquireFunction(functionID, metadata.Concurrency.Limit)
		defer release()
	}

	// Apply rate limiting if configured
	if metadata != nil && metadata.RateLimit != nil {
		if rlErr := a.rateLimiter.Acquire(functionID, *metadata.RateLimit, ""); rlErr != nil {
			// Rate limit exceeded with reject strategy
			rlResult := workflow.FunctionResult{
				Output: workflow.FunctionOutput{
					Status: workflow.FunctionError,
					Data:   map[string]any{"error": rlErr.Error()},
				},
			}
			resultMsg := messaging.NewFunctionResultMessage(msgPayload.WorkflowID, msgPayload.ThreadID, msgPayload.ExecID, rlResult)
			return a.Send(actornames.WorkflowHandlerName(msgPayload.WorkflowID), resultMsg)
		}
	}

	input, err := workflow.NewFunctionInputWith(msgPayload.Input)
	if err != nil {
		a.Log().Error("failed to create function input: %s", err)
		return nil
	}

	result, err := pkg.ExecuteFunction(
		a,
		msgPayload.FunctionID,
		workflow.NewExecutionInfo(msgPayload.WorkflowID, msgPayload.ExecID, input),
	)
	if err != nil {
		if result.Output.Status != workflow.FunctionError {
			a.Log().Error("failed to execute function %s: %s", msgPayload.FunctionID, err)
			result = workflow.FunctionResult{
				Output: workflow.FunctionOutput{
					Status: workflow.FunctionError,
					Data:   map[string]any{"error": err.Error()},
				},
			}
		} else {
			a.Log().Warning("function %s returned non-nil error with FunctionError result; delivering result to workflow", msgPayload.FunctionID, err)
		}
	}
	jsonResult, _ := json.Marshal(result)
	a.Log().Debug("execute function %s result: %s", msgPayload.FunctionID, string(jsonResult))

	resultMsg := messaging.NewFunctionResultMessage(msgPayload.WorkflowID, msgPayload.ThreadID, msgPayload.ExecID, result)
	err = a.Send(actornames.WorkflowHandlerName(msgPayload.WorkflowID), resultMsg)
	if err != nil {
		return err
	}

	return nil
}
