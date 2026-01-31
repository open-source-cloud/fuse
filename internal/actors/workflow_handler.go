package actors

import (
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowHandlerFactory redefines the WorkflowHandler factory generic type for better readability
type WorkflowHandlerFactory ActorFactory[*WorkflowHandler]

// NewWorkflowHandlerFactory DI method for creating the WorkflowHandler factory
func NewWorkflowHandlerFactory(
	cfg *config.Config,
	graphService services.GraphService,
	workflowRepository repositories.WorkflowRepository,
) *WorkflowHandlerFactory {
	return &WorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowHandler{
				config:             cfg,
				graphService:       graphService,
				workflowRepository: workflowRepository,
			}
		},
	}
}

type (
	// WorkflowHandler defines the WorkflowHandler actor
	WorkflowHandler struct {
		act.Actor

		config             *config.Config
		graphService       services.GraphService
		workflowRepository repositories.WorkflowRepository

		workflow        *internalworkflow.Workflow
		streamCallbacks []workflow.StreamCallback
	}

	// WorkflowHandlerInitArgs defines the typed arguments for the WorkflowHandler Actor Init message
	WorkflowHandlerInitArgs struct {
		schemaID   string
		workflowID workflow.ID
	}
)

// Init is called whenever a WorkflowHandler actor is being initialized
func (a *WorkflowHandler) Init(args ...any) error {
	a.Log().Debug("starting process %s with args %s", a.PID(), args)

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]")
	}
	initArgs, ok := args[0].(WorkflowHandlerInitArgs)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]; got %T", args[0])
	}

	err := a.Send(a.PID(), messaging.NewActorInitMessage(initArgs))
	if err != nil {
		return err
	}

	return nil
}

// HandleMessage processes messages that are sent to a WorkflowHandler actor
func (a *WorkflowHandler) HandleMessage(from gen.PID, message any) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		a.Log().Error("message from %s is not a messaging.Message", from)
		return nil
	}
	a.Log().Info("got message from %s - %s", from, msg.Type)
	jsonArgs, _ := json.Marshal(msg.Args)
	a.Log().Debug("args: %s", string(jsonArgs))

	switch msg.Type {
	case messaging.ActorInit:
		return a.handleMsgActorInit(msg)
	case messaging.FunctionResult:
		return a.handleMsgFunctionResult(msg)
	case messaging.AsyncFunctionResult:
		return a.handleMsgAsyncFunctionResult(msg)
	case messaging.StreamWorkflow:
		return a.handleMsgStreamWorkflow(msg)
	}

	return nil
}

// Terminate is called whenever a WorkflowHandler actor gets terminated
func (a *WorkflowHandler) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}

func (a *WorkflowHandler) handleMsgActorInit(msg messaging.Message) error {
	initArgs, ok := msg.Args.(WorkflowHandlerInitArgs)
	if !ok {
		a.Log().Error("failed to get workflow init args from %s", msg)
		return nil
	}

	if a.workflowRepository.Exists(initArgs.workflowID.String()) {
		a.workflow, _ = a.workflowRepository.Get(initArgs.workflowID.String())
		var action workflowactions.Action
		if a.workflow.State() == internalworkflow.StateUntriggered {
			action = a.workflow.Trigger()
		} else {
			// TODO : add Resume
			action = a.workflow.Resume()
		}
		a.handleWorkflowAction(action)
		return nil
	}

	// doesnt exist - create
	graphRef, err := a.graphService.FindByID(initArgs.schemaID)
	if err != nil {
		a.Log().Error("failed to get graph for schema id %s: %s", initArgs.schemaID, err)
		return gen.TerminateReasonPanic
	}
	a.workflow = internalworkflow.New(initArgs.workflowID, graphRef)
	if a.workflowRepository.Save(a.workflow) != nil {
		a.Log().Error("failed to save workflow for id %s: %s", initArgs.workflowID, err)
		return nil
	}
	a.Log().Debug("created new workflow with id %s", initArgs.workflowID)

	action := a.workflow.Trigger()
	a.emitStreamEvent(internalworkflow.StreamEvent{
		Type:       internalworkflow.StreamEventWorkflowStarted,
		WorkflowID: a.workflow.ID(),
		State:      a.workflow.State(),
	})
	a.handleWorkflowAction(action)
	return nil
}

func (a *WorkflowHandler) handleMsgFunctionResult(msg messaging.Message) error {
	fnResultMsg, ok := msg.Args.(messaging.FunctionResultMessage)
	if !ok {
		a.Log().Error("failed to get function result from %s", msg)
	}

	a.workflow.SetResultFor(fnResultMsg.ExecID, &fnResultMsg.Result)

	// Emit node result event
	currentAuditEntry, _ := a.workflow.AuditLog().Get(fnResultMsg.ExecID.String())
	if currentAuditEntry != nil {
		a.emitStreamEvent(internalworkflow.StreamEvent{
			Type:       internalworkflow.StreamEventNodeResult,
			WorkflowID: fnResultMsg.WorkflowID,
			NodeID:     currentAuditEntry.FunctionNodeID,
			ThreadID:   fnResultMsg.ThreadID,
			ExecID:     fnResultMsg.ExecID,
			Data:       fnResultMsg.Result.Output.Data,
		})
	}

	if fnResultMsg.Result.Async {
		a.Log().Debug("got async function result for workflow %s, execID %s", fnResultMsg.WorkflowID, fnResultMsg.ExecID)
		// TODO handle async
		return nil
	}
	if fnResultMsg.Result.Output.Status != workflow.FunctionSuccess {
		a.Log().Error(
			"function result for workflow %s, execID %s failed with status %s",
			fnResultMsg.WorkflowID,
			fnResultMsg.ExecID,
			fnResultMsg.Result.Output.Status,
		)
		a.workflow.SetState(internalworkflow.StateError)
		a.emitStreamEvent(internalworkflow.StreamEvent{
			Type:       internalworkflow.StreamEventWorkflowError,
			WorkflowID: fnResultMsg.WorkflowID,
			State:      internalworkflow.StateError,
			Error:      fmt.Sprintf("function failed with status %s", fnResultMsg.Result.Output.Status),
		})
		// TODO handle function failure
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ThreadID)
	if action.Type() == workflowactions.ActionNoop {
		a.Log().Warning("got noop action from workflow")
		return nil
	}
	a.handleWorkflowAction(action)

	return nil
}

func (a *WorkflowHandler) handleMsgAsyncFunctionResult(msg messaging.Message) error {
	fnResultMsg, ok := msg.Args.(messaging.AsyncFunctionResultMessage)
	if !ok {
		a.Log().Error("failed to get async function result from %s", msg)
	}

	a.workflow.SetResultFor(fnResultMsg.ExecID, &workflow.FunctionResult{
		Async:  true,
		Output: fnResultMsg.Output,
	})

	// Emit node result event
	currentAuditEntry, _ := a.workflow.AuditLog().Get(fnResultMsg.ExecID.String())
	if currentAuditEntry != nil {
		a.emitStreamEvent(internalworkflow.StreamEvent{
			Type:      internalworkflow.StreamEventNodeResult,
			WorkflowID: fnResultMsg.WorkflowID,
			NodeID:    currentAuditEntry.FunctionNodeID,
			ThreadID:  fnResultMsg.ExecID.Thread(),
			ExecID:    fnResultMsg.ExecID,
			Data:      fnResultMsg.Output.Data,
		})
	}

	if fnResultMsg.Output.Status != workflow.FunctionSuccess {
		a.Log().Error(
			"async function result for workflow %s, execID %s failed with status %s",
			fnResultMsg.WorkflowID,
			fnResultMsg.ExecID,
			fnResultMsg.Output.Status,
		)
		a.workflow.SetState(internalworkflow.StateError)
		a.emitStreamEvent(internalworkflow.StreamEvent{
			Type:      internalworkflow.StreamEventWorkflowError,
			WorkflowID: fnResultMsg.WorkflowID,
			State:     internalworkflow.StateError,
			Error:     fmt.Sprintf("function failed with status %s", fnResultMsg.Output.Status),
		})
		// TODO handle function failure
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ExecID.Thread())
	if action.Type() == workflowactions.ActionNoop {
		a.Log().Warning("got noop action from workflow")
		// Check if workflow is finished
		if a.workflow.State() == internalworkflow.StateFinished {
			a.emitStreamEvent(internalworkflow.StreamEvent{
				Type:      internalworkflow.StreamEventWorkflowCompleted,
				WorkflowID: fnResultMsg.WorkflowID,
				State:     internalworkflow.StateFinished,
			})
		}
		return nil
	}
	a.handleWorkflowAction(action)

	return nil
}

func (a *WorkflowHandler) handleWorkflowAction(action workflowactions.Action) {
	switch action.Type() {
	case workflowactions.ActionRunFunction:
		a.handleWorkflowRunFunctionAction(action)
	case workflowactions.ActionRunParallelFunctions:
		for _, runFuncAction := range action.(*workflowactions.RunParallelFunctionsAction).Actions {
			a.handleWorkflowRunFunctionAction(runFuncAction)
		}
	}
}

func (a *WorkflowHandler) handleWorkflowRunFunctionAction(action workflowactions.Action) {
	workflowPool := WorkflowFuncPoolName(a.workflow.ID())
	execAction := action.(*workflowactions.RunFunctionAction)

	// Emit node executing event
	currentAuditEntry, _ := a.workflow.AuditLog().Get(execAction.FunctionExecID.String())
	if currentAuditEntry != nil {
		a.emitStreamEvent(internalworkflow.StreamEvent{
			Type:      internalworkflow.StreamEventNodeExecuting,
			WorkflowID: a.workflow.ID(),
			NodeID:    currentAuditEntry.FunctionNodeID,
			ThreadID:  execAction.ThreadID,
			ExecID:    execAction.FunctionExecID,
		})
	}

	execFnMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), execAction)
	err := a.Send(workflowPool, execFnMsg)
	if err != nil {
		a.Log().Error("failed to send execute function message to %s: %s", workflowPool, err)
		return
	}
	a.workflow.SetState(internalworkflow.StateRunning)
	a.emitStreamEvent(internalworkflow.StreamEvent{
		Type:       internalworkflow.StreamEventWorkflowStateChanged,
		WorkflowID: a.workflow.ID(),
		State:      a.workflow.State(),
	})
}

func (a *WorkflowHandler) handleMsgStreamWorkflow(msg messaging.Message) error {
	streamMsg, err := msg.StreamWorkflowMessage()
	if err != nil {
		a.Log().Error("failed to get stream workflow message from %s: %s", msg, err)
		return nil
	}

	if streamMsg.Callback == nil {
		// Unregister callback
		callbacks := make([]workflow.StreamCallback, 0)
		for _, cb := range a.streamCallbacks {
			// Compare function pointers - this is a simple approach
			// In production, you might want a more sophisticated callback management
			if cb != nil {
				callbacks = append(callbacks, cb)
			}
		}
		a.streamCallbacks = callbacks
		a.Log().Debug("unregistered stream callback for workflow %s", streamMsg.WorkflowID)
	} else {
		// Register callback
		a.streamCallbacks = append(a.streamCallbacks, streamMsg.Callback)
		a.Log().Debug("registered stream callback for workflow %s", streamMsg.WorkflowID)
	}

	return nil
}

func (a *WorkflowHandler) emitStreamEvent(event internalworkflow.StreamEvent) {
	if a.workflow == nil {
		return
	}

	event.WorkflowID = a.workflow.ID()
	for _, callback := range a.streamCallbacks {
		chunk := workflow.StreamChunk{
			Type: workflow.StreamChunkData,
			Data: map[string]any{
				"event_type":  string(event.Type),
				"workflow_id": event.WorkflowID.String(),
				"node_id":     event.NodeID,
				"thread_id":   event.ThreadID,
				"exec_id":     event.ExecID.String(),
				"state":       string(event.State),
				"data":        event.Data,
				"error":       event.Error,
			},
		}
		if err := callback(chunk); err != nil {
			a.Log().Error("failed to emit stream event: %s", err)
		}
	}
}
