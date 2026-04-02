package actors

import (
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"
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
	journalRepository repositories.JournalRepository,
) *WorkflowHandlerFactory {
	return &WorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowHandler{
				config:             cfg,
				graphService:       graphService,
				workflowRepository: workflowRepository,
				journalRepo:        journalRepository,
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
		journalRepo        repositories.JournalRepository

		workflow       *internalworkflow.Workflow
		executionTimer *ExecutionTimer
	}

	// WorkflowHandlerInitArgs defines the typed arguments for the WorkflowHandler Actor Init message
	WorkflowHandlerInitArgs struct {
		schemaID   string
		workflowID workflow.ID
	}
)

// Init is called whenever a WorkflowHandler actor is being initialized.
// In ergo v3.2.0, Send to sibling processes works during Init, so we can
// perform all initialization inline without the ActorInit self-send pattern.
func (a *WorkflowHandler) Init(args ...any) error {
	a.Log().Debug("starting process %s with args %s", a.PID(), args)
	a.executionTimer = NewExecutionTimer()

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]")
	}
	initArgs, ok := args[0].(WorkflowHandlerInitArgs)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]; got %T", args[0])
	}

	if a.workflowRepository.Exists(initArgs.workflowID.String()) {
		a.workflow, _ = a.workflowRepository.Get(initArgs.workflowID.String())
		var action workflowactions.Action
		if a.workflow.State() == internalworkflow.StateUntriggered {
			action = a.workflow.Trigger()
			a.persistJournal()
		} else {
			// Load journal from persistence for replay
			entries, loadErr := a.journalRepo.LoadAll(initArgs.workflowID.String())
			if loadErr != nil {
				a.Log().Error("failed to load journal for workflow %s: %s", initArgs.workflowID, loadErr)
			} else {
				a.workflow.Journal().LoadFrom(entries)
			}
			action = a.workflow.Resume()
		}
		if action != nil {
			a.handleWorkflowAction(action)
		}
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
	a.persistJournal()
	a.startWorkflowTimeout()
	a.handleWorkflowAction(action)
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
	case messaging.FunctionResult:
		return a.handleMsgFunctionResult(msg)
	case messaging.AsyncFunctionResult:
		return a.handleMsgAsyncFunctionResult(msg)
	case messaging.Timeout:
		return a.handleMsgTimeout(msg)
	case messaging.WorkflowTimeout:
		return a.handleMsgWorkflowTimeout()
	}

	return nil
}

// Terminate is called whenever a WorkflowHandler actor gets terminated
func (a *WorkflowHandler) Terminate(reason error) {
	a.Log().Info("%s terminated with reason: %s", a.PID(), reason)
}

func (a *WorkflowHandler) handleMsgFunctionResult(msg messaging.Message) error {
	fnResultMsg, ok := msg.Args.(messaging.FunctionResultMessage)
	if !ok {
		a.Log().Error("failed to get function result from %s", msg)
	}

	a.cancelExecutionTimeout(fnResultMsg.ExecID)
	a.workflow.SetResultFor(fnResultMsg.ExecID, &fnResultMsg.Result)

	if fnResultMsg.Result.Async {
		a.Log().Debug("got async function result for workflow %s, execID %s", fnResultMsg.WorkflowID, fnResultMsg.ExecID)
		a.persistJournal()
		return nil
	}
	if fnResultMsg.Result.Output.Status != workflow.FunctionSuccess {
		a.Log().Error(
			"function result for workflow %s, execID %s failed with status %s",
			fnResultMsg.WorkflowID,
			fnResultMsg.ExecID,
			fnResultMsg.Result.Output.Status,
		)
		action := a.workflow.HandleNodeFailure(fnResultMsg.ThreadID, fnResultMsg.ExecID)
		if action == nil {
			a.workflow.SetState(internalworkflow.StateError)
			a.persistJournal()
			return nil
		}
		a.persistJournal()
		a.handleWorkflowAction(action)
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ThreadID)
	a.persistJournal()
	if action.Type() == workflowactions.ActionNoop {
		a.checkWorkflowCompletion()
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

	a.cancelExecutionTimeout(fnResultMsg.ExecID)
	a.workflow.SetResultFor(fnResultMsg.ExecID, &workflow.FunctionResult{
		Async:  true,
		Output: fnResultMsg.Output,
	})
	if fnResultMsg.Output.Status != workflow.FunctionSuccess {
		a.Log().Error(
			"async function result for workflow %s, execID %s failed with status %s",
			fnResultMsg.WorkflowID,
			fnResultMsg.ExecID,
			fnResultMsg.Output.Status,
		)
		action := a.workflow.HandleNodeFailure(fnResultMsg.ExecID.Thread(), fnResultMsg.ExecID)
		if action == nil {
			a.workflow.SetState(internalworkflow.StateError)
			a.persistJournal()
			return nil
		}
		a.persistJournal()
		a.handleWorkflowAction(action)
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ExecID.Thread())
	a.persistJournal()
	if action.Type() == workflowactions.ActionNoop {
		a.checkWorkflowCompletion()
		return nil
	}
	a.handleWorkflowAction(action)

	return nil
}

func (a *WorkflowHandler) persistJournal() {
	newEntries := a.workflow.Journal().NewEntries()
	if len(newEntries) == 0 {
		return
	}
	if err := a.journalRepo.Append(a.workflow.ID().String(), newEntries...); err != nil {
		a.Log().Error("failed to persist journal: %s", err)
		return
	}
	a.workflow.Journal().MarkPersisted()
}

func (a *WorkflowHandler) checkWorkflowCompletion() {
	if !a.workflow.AllThreadsFinished() {
		a.Log().Debug("noop action but not all threads finished yet")
		return
	}
	a.workflow.SetState(internalworkflow.StateFinished)
	a.Log().Info("workflow %s completed with state %s", a.workflow.ID(), a.workflow.State())

	supName := actornames.WorkflowInstanceSupervisorName(a.workflow.ID())
	completedMsg := messaging.NewWorkflowCompletedMessage(a.workflow.ID(), a.workflow.State().String())
	if err := a.Send(gen.Atom(supName), completedMsg); err != nil {
		a.Log().Error("failed to send workflow completed message: %s", err)
	}
}

func (a *WorkflowHandler) handleMsgTimeout(msg messaging.Message) error {
	timeoutMsg, ok := msg.Args.(messaging.TimeoutMessage)
	if !ok {
		return nil
	}

	a.Log().Warning("execution timeout for exec %s", timeoutMsg.ExecID)
	execID := workflow.ExecID(timeoutMsg.ExecID)

	// Create a timeout error result and feed through normal error handling
	result := &workflow.FunctionResult{
		Output: workflow.FunctionOutput{
			Status: workflow.FunctionError,
			Data:   map[string]any{"error": "execution timeout exceeded"},
		},
	}
	a.workflow.SetResultFor(execID, result)

	action := a.workflow.HandleNodeFailure(execID.Thread(), execID)
	if action == nil {
		a.workflow.SetState(internalworkflow.StateError)
		a.persistJournal()
		return nil
	}
	a.persistJournal()
	a.handleWorkflowAction(action)
	return nil
}

func (a *WorkflowHandler) handleMsgWorkflowTimeout() error {
	a.Log().Warning("workflow timeout for %s", a.workflow.ID())
	a.workflow.SetState(internalworkflow.StateError)
	a.persistJournal()
	a.checkWorkflowCompletion()
	return nil
}

func (a *WorkflowHandler) startExecutionTimeout(execID workflow.ExecID, node *internalworkflow.Node) {
	if node.Schema().Timeout == nil || node.Schema().Timeout.Execution == 0 {
		return
	}
	a.executionTimer.Start(a, a.PID(), execID.String(), node.Schema().Timeout.Execution)
}

func (a *WorkflowHandler) cancelExecutionTimeout(execID workflow.ExecID) {
	a.executionTimer.Cancel(execID.String())
}

func (a *WorkflowHandler) startWorkflowTimeout() {
	schema := a.workflow.Schema()
	if schema.Timeout == nil || schema.Timeout.Total == 0 {
		return
	}
	timeoutMsg := messaging.NewWorkflowTimeoutMessage(a.workflow.ID())
	if _, err := a.SendAfter(a.PID(), timeoutMsg, schema.Timeout.Total); err != nil {
		a.Log().Error("failed to set workflow timeout: %s", err)
	}
}

func (a *WorkflowHandler) handleWorkflowAction(action workflowactions.Action) {
	switch action.Type() {
	case workflowactions.ActionRunFunction:
		a.handleWorkflowRunFunctionAction(action)
	case workflowactions.ActionRunParallelFunctions:
		for _, runFuncAction := range action.(*workflowactions.RunParallelFunctionsAction).Actions {
			a.handleWorkflowRunFunctionAction(runFuncAction)
		}
	case workflowactions.ActionRetryFunction:
		retryAction := action.(*workflowactions.RetryFunctionAction)
		a.Log().Info("scheduling retry attempt %d for exec %s in %s",
			retryAction.Attempt, retryAction.FunctionExecID, retryAction.Delay)
		workflowPool := WorkflowFuncPoolName(a.workflow.ID())
		retryMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), &retryAction.RunFunctionAction)
		if _, err := a.SendAfter(gen.Atom(workflowPool), retryMsg, retryAction.Delay); err != nil {
			a.Log().Error("failed to schedule retry: %s", err)
		}
		a.workflow.SetState(internalworkflow.StateRunning)
	}
}

func (a *WorkflowHandler) handleWorkflowRunFunctionAction(action workflowactions.Action) {
	workflowPool := WorkflowFuncPoolName(a.workflow.ID())
	execAction := action.(*workflowactions.RunFunctionAction)

	// Start execution timeout if configured for this node
	if entry, exists := a.workflow.AuditLog().Get(execAction.FunctionExecID.String()); exists {
		if node, err := a.workflow.Graph().FindNode(entry.FunctionNodeID); err == nil {
			a.startExecutionTimeout(execAction.FunctionExecID, node)
		}
	}

	execFnMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), execAction)
	err := a.Send(workflowPool, execFnMsg)
	if err != nil {
		a.Log().Error("failed to send execute function message to %s: %s", workflowPool, err)
		return
	}
	a.workflow.SetState(internalworkflow.StateRunning)
}
