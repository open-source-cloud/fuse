package actors

import (
	"encoding/json"
	"fmt"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowHandlerFactory redefines the WorkflowHandler factory generic type for better readability
type WorkflowHandlerFactory ActorFactory[*WorkflowHandler]

// NewWorkflowHandlerFactory DI method for creating the WorkflowHandler factory
func NewWorkflowHandlerFactory(
	cfg *config.Config,
	graphRepo repos.GraphRepo,
	workflowRepo repos.WorkflowRepo,
) *WorkflowHandlerFactory {
	return &WorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowHandler{
				config:       cfg,
				graphRepo:    graphRepo,
				workflowRepo: workflowRepo,
			}
		},
	}
}

type (
	// WorkflowHandler defines the WorkflowHandler actor
	WorkflowHandler struct {
		act.Actor

		config       *config.Config
		graphRepo    repos.GraphRepo
		workflowRepo repos.WorkflowRepo

		workflow *workflow.Workflow
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

	if a.workflowRepo.Exists(initArgs.workflowID.String()) {
		a.workflow, _ = a.workflowRepo.Get(initArgs.workflowID.String())
		var action workflow.Action
		if a.workflow.State() == workflow.StateUntriggered {
			action = a.workflow.Trigger()
		} else {
			// TODO : add Resume
			action = a.workflow.Resume()
		}
		a.handleWorkflowAction(action)
		return nil
	}

	// doesnt exist - create
	graphRef, err := a.graphRepo.Get(initArgs.schemaID)
	if err != nil {
		a.Log().Error("failed to get graph for schema id %s: %s", initArgs.schemaID, err)
		return gen.TerminateReasonPanic
	}
	a.workflow = workflow.New(initArgs.workflowID, graphRef)
	if a.workflowRepo.Save(a.workflow) != nil {
		a.Log().Error("failed to save workflow for id %s: %s", initArgs.workflowID, err)
		return nil
	}
	a.Log().Debug("created new workflow with id %s", initArgs.workflowID)

	action := a.workflow.Trigger()
	a.handleWorkflowAction(action)
	return nil
}

func (a *WorkflowHandler) handleMsgFunctionResult(msg messaging.Message) error {
	fnResultMsg, ok := msg.Args.(messaging.FunctionResultMessage)
	if !ok {
		a.Log().Error("failed to get function result from %s", msg)
	}

	a.workflow.SetResultFor(fnResultMsg.ExecID, &fnResultMsg.Result)

	if fnResultMsg.Result.Async {
		a.Log().Debug("got async function result for workflow %s, execID %s", fnResultMsg.WorkflowID, fnResultMsg.ExecID)
		// TODO handle async
		return nil
	}
	if fnResultMsg.Result.Output.Status != pkgworkflow.FunctionSuccess {
		a.Log().Error(
			"function result for workflow %s, execID %s failed with status %s",
			fnResultMsg.WorkflowID,
			fnResultMsg.ExecID,
			fnResultMsg.Result.Output.Status,
		)
		a.workflow.SetState(workflow.StateError)
		// TODO handle function failure
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ThreadID)
	if action.Type() == workflow.ActionNoop {
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

	a.workflow.SetResultFor(fnResultMsg.ExecID, &pkgworkflow.FunctionResult{
		Async:  true,
		Output: fnResultMsg.Output,
	})
	if fnResultMsg.Output.Status != pkgworkflow.FunctionSuccess {
		a.Log().Error(
			"async function result for workflow %s, execID %s failed with status %s",
			fnResultMsg.WorkflowID,
			fnResultMsg.ExecID,
			fnResultMsg.Output.Status,
		)
		a.workflow.SetState(workflow.StateError)
		// TODO handle function failure
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ExecID.Thread())
	if action.Type() == workflow.ActionNoop {
		a.Log().Warning("got noop action from workflow")
		return nil
	}
	a.handleWorkflowAction(action)

	return nil
}

func (a *WorkflowHandler) handleWorkflowAction(action workflow.Action) {
	switch action.Type() {
	case workflow.ActionRunFunction:
		a.handleWorkflowRunFunctionAction(action)
	case workflow.ActionRunParallelFunctions:
		for _, runFuncAction := range action.(*workflow.RunParallelFunctionsAction).Actions {
			a.handleWorkflowRunFunctionAction(runFuncAction)
		}
	}
}

func (a *WorkflowHandler) handleWorkflowRunFunctionAction(action workflow.Action) {
	workflowPool := WorkflowFuncPoolName(a.workflow.ID())
	execAction := action.(*workflow.RunFunctionAction)

	execFnMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), execAction)
	err := a.Send(workflowPool, execFnMsg)
	if err != nil {
		a.Log().Error("failed to send execute function message to %s: %s", workflowPool, err)
		return
	}
	a.workflow.SetState(workflow.StateRunning)

}
