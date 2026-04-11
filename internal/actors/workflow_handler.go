package actors

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/events"
	"github.com/open-source-cloud/fuse/internal/metrics"
	"github.com/open-source-cloud/fuse/internal/packages/functions/system"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/tracing"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/google/uuid"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
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
	awakeableRepository repositories.AwakeableRepository,
	store objectstore.ObjectStore,
	traceRepo repositories.TraceRepository,
	eventBus events.EventBus,
	fuseMetrics *metrics.FuseMetrics,
	tracingProvider *tracing.Provider,
) *WorkflowHandlerFactory {
	return &WorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowHandler{
				config:             cfg,
				graphService:       graphService,
				workflowRepository: workflowRepository,
				journalRepo:        journalRepository,
				awakeableRepo:      awakeableRepository,
				objectStore:        store,
				traceRepo:          traceRepo,
				eventBus:           eventBus,
				fuseMetrics:        fuseMetrics,
				tracingProvider:    tracingProvider,
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
		awakeableRepo      repositories.AwakeableRepository
		objectStore        objectstore.ObjectStore
		traceRepo          repositories.TraceRepository
		eventBus           events.EventBus
		fuseMetrics        *metrics.FuseMetrics
		tracingProvider    *tracing.Provider

		workflow       *internalworkflow.Workflow
		executionTimer *ExecutionTimer

		// OTel root span covering the entire workflow lifetime.
		rootSpan trace.Span
		spanCtx  context.Context
    
		// ForEach tracking: one ForEachState per active foreach execID.
		forEachStates       map[string]*internalworkflow.ForEachState
		// iterThreadToForEach maps a live iteration thread ID back to its parent
		// foreach execID so the completion handler can find the ForEachState.
		iterThreadToForEach map[uint16]string
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
	a.spanCtx = context.Background()
	a.forEachStates = make(map[string]*internalworkflow.ForEachState)
	a.iterThreadToForEach = make(map[uint16]string)

	if len(args) != 1 {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]")
	}
	initArgs, ok := args[0].(WorkflowHandlerInitArgs)
	if !ok {
		return fmt.Errorf("workflow actor init args must be 1 == [WorkflowHandlerInitArgs]; got %T", args[0])
	}

	if a.workflowRepository.Exists(initArgs.workflowID.String()) {
		a.workflow, _ = a.workflowRepository.Get(initArgs.workflowID.String())
		if err := a.graphService.EnsureNodeMetadata(a.workflow.Graph()); err != nil {
			a.Log().Error("failed to populate graph node metadata for workflow %s: %s", initArgs.workflowID, err)
			return gen.TerminateReasonPanic
		}
		a.startRootSpan()
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
	a.startRootSpan()

	action := a.workflow.Trigger()
	a.persistJournal()
	a.startWorkflowTimeout()
	a.handleWorkflowAction(action)
	return nil
}

// startRootSpan begins the OTel root span for this workflow and increments active workflow metrics.
// Must be called after a.workflow is set.
func (a *WorkflowHandler) startRootSpan() {
	a.spanCtx, a.rootSpan = a.tracingProvider.StartSpan(
		context.Background(),
		"workflow.execute",
		attribute.String("workflow.id", a.workflow.ID().String()),
		attribute.String("workflow.schema_id", a.workflow.Graph().ID()),
	)
	a.fuseMetrics.WorkflowsActive.Inc()
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
	case messaging.CancelWorkflow:
		return a.handleMsgCancelWorkflow(msg)
	case messaging.SleepWakeUp:
		return a.handleMsgSleepWakeUp(msg)
	case messaging.AwakeableResolvedMsg:
		return a.handleMsgAwakeableResolved(msg)
	case messaging.SubWorkflowCompleted:
		return a.handleMsgSubWorkflowCompleted(msg)
	case messaging.RetryNode:
		return a.handleMsgRetryNode(msg)
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

	if a.isTerminalState() {
		a.Log().Warning("ignoring function result for %s workflow %s", a.workflow.State(), a.workflow.ID())
		return nil
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
			a.completeWithError()
			return nil
		}
		a.persistJournal()
		a.handleWorkflowAction(action)
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ThreadID)
	a.persistJournal()
	if action.Type() == workflowactions.ActionNoop {
		if a.handleForEachIterationComplete(fnResultMsg.ThreadID) {
			return nil
		}
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

	if a.isTerminalState() {
		a.Log().Warning("ignoring async function result for %s workflow %s", a.workflow.State(), a.workflow.ID())
		return nil
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
			a.completeWithError()
			return nil
		}
		a.persistJournal()
		a.handleWorkflowAction(action)
		return nil
	}

	action := a.workflow.Next(fnResultMsg.ExecID.Thread())
	a.persistJournal()
	if action.Type() == workflowactions.ActionNoop {
		if a.handleForEachIterationComplete(fnResultMsg.ExecID.Thread()) {
			return nil
		}
		a.checkWorkflowCompletion()
		return nil
	}
	a.handleWorkflowAction(action)

	return nil
}

// persistWorkflowState persists both the journal and the workflow state to the repository.
// Call this after any state transition that should be visible to external queries (e.g. running → sleeping).
func (a *WorkflowHandler) persistWorkflowState() {
	a.persistJournal()
	if err := a.workflowRepository.Save(a.workflow); err != nil {
		a.Log().Error("failed to persist workflow state for %s: %s", a.workflow.ID(), err)
	}
}

func (a *WorkflowHandler) persistSnapshot() {
	snap := internalworkflow.BuildExecutionSnapshot(
		a.workflow.ID().String(),
		a.workflow.Graph().ID(),
		a.workflow.State(),
		a.workflow.Journal().Entries(),
		a.workflow.AggregatedOutputSnapshot(),
	)

	data, err := json.Marshal(snap)
	if err != nil {
		a.Log().Error("failed to marshal execution snapshot for %s: %s", a.workflow.ID(), err)
		return
	}

	key := fmt.Sprintf("workflows/%s/execution-snapshot.json", a.workflow.ID())
	if putErr := a.objectStore.Put(context.Background(), key, data); putErr != nil {
		a.Log().Error("failed to write execution snapshot for %s: %s", a.workflow.ID(), putErr)
		return
	}

	if refErr := a.workflowRepository.SetSnapshotRef(a.workflow.ID().String(), key); refErr != nil {
		a.Log().Error("failed to set snapshot ref for %s: %s", a.workflow.ID(), refErr)
	}
}

func (a *WorkflowHandler) persistTrace() {
	trace := internalworkflow.BuildTrace(
		a.workflow.ID().String(),
		a.workflow.Graph().ID(),
		a.workflow.Journal().Entries(),
	)
	if err := a.traceRepo.Save(trace); err != nil {
		a.Log().Error("failed to persist execution trace for %s: %s", a.workflow.ID(), err)
	}
}

func (a *WorkflowHandler) publishLifecycleEvent() {
	var eventType string
	switch a.workflow.State() {
	case internalworkflow.StateFinished:
		eventType = events.EventWorkflowCompleted
	case internalworkflow.StateError:
		eventType = events.EventWorkflowFailed
	case internalworkflow.StateCancelled:
		eventType = events.EventWorkflowCancelled
	default:
		return
	}

	if err := a.eventBus.Publish(events.Event{
		Type:   eventType,
		Source: a.workflow.ID().String(),
		Data: map[string]any{
			"workflowId": a.workflow.ID().String(),
			"schemaId":   a.workflow.Graph().ID(),
			"status":     a.workflow.State().String(),
		},
	}); err != nil {
		a.Log().Error("failed to publish lifecycle event for %s: %s", a.workflow.ID(), err)
	}
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
	a.sendWorkflowCompleted()
}

func (a *WorkflowHandler) completeWithError() {
	a.workflow.SetState(internalworkflow.StateError)
	a.persistJournal()
	a.sendWorkflowCompleted()
}

func (a *WorkflowHandler) isTerminalState() bool {
	s := a.workflow.State()
	return s == internalworkflow.StateFinished || s == internalworkflow.StateError || s == internalworkflow.StateCancelled
}

func (a *WorkflowHandler) sendWorkflowCompleted() {
	a.Log().Info("workflow %s completed with state %s", a.workflow.ID(), a.workflow.State())

	// Record metrics and end OTel root span.
	a.fuseMetrics.WorkflowsActive.Dec()
	switch a.workflow.State() {
	case internalworkflow.StateFinished:
		a.fuseMetrics.WorkflowsCompleted.Inc()
		a.rootSpan.SetStatus(codes.Ok, "")
	case internalworkflow.StateError:
		a.fuseMetrics.WorkflowsFailed.Inc()
		a.rootSpan.SetStatus(codes.Error, "workflow ended in error state")
	case internalworkflow.StateCancelled:
		a.fuseMetrics.WorkflowsCancelled.Inc()
		a.rootSpan.SetStatus(codes.Error, "workflow cancelled")
	}
	a.rootSpan.SetAttributes(attribute.String("workflow.final_state", a.workflow.State().String()))
	a.rootSpan.End()

	// Persist the terminal state, execution snapshot, and trace
	a.persistJournal()
	if err := a.workflowRepository.Save(a.workflow); err != nil {
		a.Log().Error("failed to persist terminal state for workflow %s: %s", a.workflow.ID(), err)
	}
	a.persistSnapshot()
	a.persistTrace()
	a.publishLifecycleEvent()

	completedMsg := messaging.NewWorkflowCompletedMessage(a.workflow.ID(), a.workflow.State().String())
	if err := a.Send(a.Parent(), completedMsg); err != nil {
		a.Log().Error("failed to send workflow completed message: %s", err)
	}

	// If this is a child workflow, notify the parent
	a.notifyParentIfSubWorkflow()
}

func (a *WorkflowHandler) handleMsgRetryNode(msg messaging.Message) error {
	retryMsg, err := msg.RetryNodeMessage()
	if err != nil {
		a.Log().Error("failed to parse retry node message: %s", err)
		return nil
	}

	a.Log().Info("manual retry requested for exec %s in workflow %s", retryMsg.ExecID, retryMsg.WorkflowID)

	action, retryErr := a.workflow.RetryNode(retryMsg.ExecID)
	if retryErr != nil {
		a.Log().Error("retry node failed for exec %s: %s", retryMsg.ExecID, retryErr)
		return nil
	}

	a.persistWorkflowState()
	a.handleWorkflowAction(action)
	return nil
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
		a.completeWithError()
		return nil
	}
	a.persistJournal()
	a.handleWorkflowAction(action)
	return nil
}

func (a *WorkflowHandler) handleMsgCancelWorkflow(msg messaging.Message) error {
	cancelMsg, ok := msg.Args.(messaging.CancelWorkflowMessage)
	if !ok {
		return nil
	}

	currentState := a.workflow.State()
	if currentState == internalworkflow.StateFinished ||
		currentState == internalworkflow.StateError ||
		currentState == internalworkflow.StateCancelled {
		a.Log().Warning("cannot cancel workflow %s in state %s", cancelMsg.WorkflowID, currentState)
		return nil
	}

	a.workflow.SetState(internalworkflow.StateCancelled)
	a.executionTimer.CancelAll()
	a.persistJournal()

	// Cascade cancel to active sub-workflows
	children, _ := a.workflowRepository.FindActiveSubWorkflows(a.workflow.ID().String())
	for _, child := range children {
		childCancelMsg := messaging.NewCancelWorkflowMessage(child.ChildWorkflowID, "parent cancelled")
		if err := a.Send(gen.Atom(actornames.WorkflowSupervisorName), childCancelMsg); err != nil {
			a.Log().Error("failed to cascade cancel to sub-workflow %s: %s", child.ChildWorkflowID, err)
		}
	}

	completedMsg := messaging.NewWorkflowCompletedMessage(a.workflow.ID(), internalworkflow.StateCancelled.String())
	if err := a.Send(a.Parent(), completedMsg); err != nil {
		a.Log().Error("failed to send cancellation completed message: %s", err)
	}

	// Notify parent if this is a child workflow
	a.notifyParentIfSubWorkflow()
	return nil
}

func (a *WorkflowHandler) handleMsgWorkflowTimeout() error {
	a.Log().Warning("workflow timeout for %s", a.workflow.ID())
	a.completeWithError()
	return nil
}

func (a *WorkflowHandler) startExecutionTimeout(execID workflow.ExecID, node *internalworkflow.Node) {
	if node.Schema().Timeout == nil || node.Schema().Timeout.Execution == 0 {
		return
	}
	a.executionTimer.Start(a, a.PID(), execID.String(), node.Schema().Timeout.Execution.Duration())
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
	if _, err := a.SendAfter(a.PID(), timeoutMsg, schema.Timeout.Total.Duration()); err != nil {
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
	case workflowactions.ActionRunSubWorkflow:
		a.handleSubWorkflowAction(action.(*workflowactions.RunSubWorkflowAction))
	case workflowactions.ActionSleep:
		a.handleSleepAction(action.(*workflowactions.SleepAction))
	case workflowactions.ActionWaitForEvent:
		a.handleWaitForEventAction(action.(*workflowactions.WaitForEventAction))
	case workflowactions.ActionRetryFunction:
		retryAction := action.(*workflowactions.RetryFunctionAction)
		a.Log().Info("scheduling retry attempt %d for exec %s in %s",
			retryAction.Attempt, retryAction.FunctionExecID, retryAction.Delay)
		workflowPool := WorkflowFuncPoolName(a.workflow.ID())
		retryMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), &retryAction.RunFunctionAction, a.tracingProvider.InjectCarrier(a.spanCtx))
		if _, err := a.SendAfter(gen.Atom(workflowPool), retryMsg, retryAction.Delay); err != nil {
			a.Log().Error("failed to schedule retry: %s", err)
		}
		a.workflow.SetState(internalworkflow.StateRunning)
	}
}

func (a *WorkflowHandler) handleWorkflowRunFunctionAction(action workflowactions.Action) {
	execAction := action.(*workflowactions.RunFunctionAction)

	// Intercept system functions — they are handled directly, not dispatched to the pool
	switch execAction.FunctionID {
	case system.SleepFullFunctionID:
		a.handleSystemSleep(execAction)
		return
	case system.WaitFullFunctionID:
		a.handleSystemWait(execAction)
		return
	case system.SubWorkflowFullFunctionID:
		a.handleSystemSubWorkflow(execAction)
		return
	case system.ForEachFullFunctionID:
		a.handleSystemForEach(execAction)
		return
	}

	workflowPool := WorkflowFuncPoolName(a.workflow.ID())

	// Start execution timeout if configured for this node
	if entry, exists := a.workflow.AuditLog().Get(execAction.FunctionExecID.String()); exists {
		if node, err := a.workflow.Graph().FindNode(entry.FunctionNodeID); err == nil {
			a.startExecutionTimeout(execAction.FunctionExecID, node)
		}
	}

	execFnMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), execAction, a.tracingProvider.InjectCarrier(a.spanCtx))
	err := a.Send(workflowPool, execFnMsg)
	if err != nil {
		a.Log().Error("failed to send execute function message to %s: %s", workflowPool, err)
		return
	}
	a.workflow.SetState(internalworkflow.StateRunning)
}

// --- Sleep / Wait ---

func (a *WorkflowHandler) handleSystemSleep(action *workflowactions.RunFunctionAction) {
	durationStr, _ := action.Args["duration"].(string)
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		a.Log().Error("invalid sleep duration %q: %s", durationStr, err)
		return
	}
	reason, _ := action.Args["reason"].(string)

	a.handleSleepAction(&workflowactions.SleepAction{
		ThreadID: action.ThreadID,
		ExecID:   action.FunctionExecID,
		Duration: duration,
		Reason:   reason,
	})
}

func (a *WorkflowHandler) handleSystemWait(action *workflowactions.RunFunctionAction) {
	awakeableID := uuid.New().String()
	var timeout time.Duration
	if timeoutStr, ok := action.Args["timeout"].(string); ok && timeoutStr != "" {
		parsed, err := time.ParseDuration(timeoutStr)
		if err != nil {
			a.Log().Error("invalid wait timeout %q: %s", timeoutStr, err)
		} else {
			timeout = parsed
		}
	}
	filter, _ := action.Args["filter"].(string)

	a.handleWaitForEventAction(&workflowactions.WaitForEventAction{
		ThreadID:    action.ThreadID,
		ExecID:      action.FunctionExecID,
		AwakeableID: awakeableID,
		Timeout:     timeout,
		Filter:      filter,
	})
}

func (a *WorkflowHandler) handleSleepAction(action *workflowactions.SleepAction) {
	a.workflow.SetState(internalworkflow.StateSleeping)
	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalSleepStarted,
		ThreadID: action.ThreadID,
		ExecID:   action.ExecID.String(),
		Data:     map[string]any{"duration": action.Duration.String(), "reason": action.Reason},
	})
	a.persistWorkflowState()

	msg := messaging.NewSleepWakeUpMessage(a.workflow.ID(), action.ExecID, action.ThreadID)
	if _, err := a.SendAfter(a.PID(), msg, action.Duration); err != nil {
		a.Log().Error("failed to schedule sleep wake-up: %s", err)
	}
}

func (a *WorkflowHandler) handleWaitForEventAction(action *workflowactions.WaitForEventAction) {
	a.workflow.SetState(internalworkflow.StateSleeping)
	now := time.Now()
	awakeable := &internalworkflow.Awakeable{
		ID:         action.AwakeableID,
		WorkflowID: a.workflow.ID(),
		ExecID:     action.ExecID,
		ThreadID:   action.ThreadID,
		CreatedAt:  now,
		Timeout:    action.Timeout,
		DeadlineAt: now.Add(action.Timeout),
		Status:     internalworkflow.AwakeablePending,
	}
	if err := a.awakeableRepo.Save(awakeable); err != nil {
		a.Log().Error("failed to save awakeable: %s", err)
	}
	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalAwakeableCreated,
		ThreadID: action.ThreadID,
		ExecID:   action.ExecID.String(),
		Data:     map[string]any{"awakeableId": action.AwakeableID, "timeout": action.Timeout.String()},
	})
	a.persistWorkflowState()

	if action.Timeout > 0 {
		timeoutMsg := messaging.NewTimeoutMessage(action.ExecID.String())
		if _, err := a.SendAfter(a.PID(), timeoutMsg, action.Timeout); err != nil {
			a.Log().Error("failed to schedule awakeable timeout: %s", err)
		}
	}
}

func (a *WorkflowHandler) handleMsgSleepWakeUp(msg messaging.Message) error {
	wakeUpMsg, ok := msg.Args.(messaging.SleepWakeUpMessage)
	if !ok {
		return nil
	}

	if a.workflow.State() == internalworkflow.StateCancelled {
		a.Log().Warning("ignoring sleep wake-up for cancelled workflow %s", a.workflow.ID())
		return nil
	}

	a.workflow.SetState(internalworkflow.StateRunning)
	a.workflow.SetResultFor(wakeUpMsg.ExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(map[string]any{
			"sleptFor": "completed",
		}),
	})
	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalSleepCompleted,
		ThreadID: wakeUpMsg.ThreadID,
		ExecID:   wakeUpMsg.ExecID.String(),
	})
	a.persistJournal()

	action := a.workflow.Next(wakeUpMsg.ThreadID)
	if action.Type() == workflowactions.ActionNoop {
		a.checkWorkflowCompletion()
		return nil
	}
	a.handleWorkflowAction(action)
	return nil
}

func (a *WorkflowHandler) handleMsgAwakeableResolved(msg messaging.Message) error {
	resolvedMsg, ok := msg.Args.(messaging.AwakeableResolvedMessage)
	if !ok {
		return nil
	}

	if a.workflow.State() == internalworkflow.StateCancelled {
		a.Log().Warning("ignoring awakeable resolved for cancelled workflow %s", a.workflow.ID())
		return nil
	}

	a.workflow.SetState(internalworkflow.StateRunning)
	a.workflow.SetResultFor(resolvedMsg.ExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(map[string]any{
			"data":     resolvedMsg.Data,
			"timedOut": false,
		}),
	})
	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalAwakeableResolved,
		ThreadID: resolvedMsg.ThreadID,
		ExecID:   resolvedMsg.ExecID.String(),
		Data:     map[string]any{"awakeableId": resolvedMsg.AwakeableID},
	})
	a.persistJournal()

	action := a.workflow.Next(resolvedMsg.ThreadID)
	if action.Type() == workflowactions.ActionNoop {
		a.checkWorkflowCompletion()
		return nil
	}
	a.handleWorkflowAction(action)
	return nil
}

func (a *WorkflowHandler) notifyParentIfSubWorkflow() {
	ref, err := a.workflowRepository.FindSubWorkflowRef(a.workflow.ID().String())
	if err != nil || ref == nil || ref.Async {
		return
	}
	parentHandlerName := actornames.WorkflowHandlerName(ref.ParentWorkflowID)
	subCompletedMsg := messaging.NewSubWorkflowCompletedMessage(
		ref.ParentWorkflowID,
		ref.ParentThreadID,
		ref.ParentExecID,
		a.workflow.ID(),
		a.workflow.State().String(),
		a.workflow.AggregatedOutputSnapshot(),
	)
	if err := a.Send(gen.Atom(parentHandlerName), subCompletedMsg); err != nil {
		a.Log().Error("failed to notify parent workflow %s: %s", ref.ParentWorkflowID, err)
	}
}

// --- Sub-workflows ---

func (a *WorkflowHandler) handleSystemSubWorkflow(action *workflowactions.RunFunctionAction) {
	schemaID, _ := action.Args["schemaId"].(string)
	input, _ := action.Args["input"].(map[string]any)
	async, _ := action.Args["async"].(bool)

	a.handleSubWorkflowAction(&workflowactions.RunSubWorkflowAction{
		ParentWorkflowID: a.workflow.ID(),
		ParentThreadID:   action.ThreadID,
		ParentExecID:     action.FunctionExecID,
		SchemaID:         schemaID,
		Input:            input,
		Async:            async,
	})
}

func (a *WorkflowHandler) handleSubWorkflowAction(action *workflowactions.RunSubWorkflowAction) {
	childWorkflowID := workflow.NewID()

	ref := &internalworkflow.SubWorkflowRef{
		ParentWorkflowID: action.ParentWorkflowID,
		ParentThreadID:   action.ParentThreadID,
		ParentExecID:     action.ParentExecID,
		ChildWorkflowID:  childWorkflowID,
		ChildSchemaID:    action.SchemaID,
		Async:            action.Async,
	}
	if err := a.workflowRepository.SaveSubWorkflowRef(ref); err != nil {
		a.Log().Error("failed to save sub-workflow ref: %s", err)
		return
	}

	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalSubWorkflowStarted,
		ThreadID: action.ParentThreadID,
		ExecID:   action.ParentExecID.String(),
		Data: map[string]any{
			"childWorkflowId": childWorkflowID.String(),
			"childSchemaId":   action.SchemaID,
			"async":           action.Async,
		},
	})

	triggerMsg := messaging.NewTriggerWorkflowMessage(action.SchemaID, childWorkflowID)
	if err := a.Send(gen.Atom(actornames.WorkflowSupervisorName), triggerMsg); err != nil {
		a.Log().Error("failed to trigger sub-workflow: %s", err)
		return
	}

	if action.Async {
		a.workflow.SetResultFor(action.ParentExecID, &workflow.FunctionResult{
			Output: workflow.NewFunctionSuccessOutput(map[string]any{
				"workflowId": childWorkflowID.String(),
				"status":     "triggered",
				"output":     nil,
			}),
		})
		a.persistJournal()
		nextAction := a.workflow.Next(action.ParentThreadID)
		if nextAction.Type() == workflowactions.ActionNoop {
			a.checkWorkflowCompletion()
			return
		}
		a.handleWorkflowAction(nextAction)
	} else {
		a.workflow.SetState(internalworkflow.StateSleeping)
		a.persistWorkflowState()
	}
}

// --- ForEach ---

// handleSystemForEach intercepts a system/foreach RunFunctionAction and starts
// the iteration loop.  Empty collections complete immediately via the "done" edge.
func (a *WorkflowHandler) handleSystemForEach(action *workflowactions.RunFunctionAction) {
	// Extract items — accept both []any (JSON) and any typed slice.
	items := toAnySlice(action.Args["items"])
	if len(items) == 0 {
		a.completeForEachImmediate(action, nil)
		return
	}

	batchSize := toPositiveInt(action.Args["batchSize"], 1)
	concurrency := toPositiveInt(action.Args["concurrency"], 1)

	// Resolve the foreach node ID from the audit log.
	auditEntry, exists := a.workflow.AuditLog().Get(action.FunctionExecID.String())
	if !exists {
		a.Log().Error("foreach: audit entry not found for exec %s", action.FunctionExecID)
		a.completeWithError()
		return
	}
	nodeID := auditEntry.FunctionNodeID

	state := internalworkflow.NewForEachState(
		action.FunctionExecID,
		action.ThreadID,
		nodeID,
		items,
		batchSize,
		concurrency,
	)
	a.forEachStates[action.FunctionExecID.String()] = state

	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalForEachStarted,
		ThreadID: action.ThreadID,
		ExecID:   action.FunctionExecID.String(),
		Data: map[string]any{
			"totalItems":  len(items),
			"batchSize":   batchSize,
			"concurrency": concurrency,
		},
	})

	// Spawn the initial batch(es) up to the configured concurrency.
	initialCount := state.InitialBatchCount()
	for i := 0; i < initialCount; i++ {
		a.spawnForEachBatch(state, i)
	}

	a.workflow.SetState(internalworkflow.StateRunning)
	a.persistJournal()
}

// spawnForEachBatch creates a new iteration thread for batch at batchIndex and
// dispatches it to the workflow function pool.
func (a *WorkflowHandler) spawnForEachBatch(state *internalworkflow.ForEachState, batchIndex int) {
	batch := state.GetBatch(batchIndex)
	isLast := batchIndex == state.TotalBatches-1

	var iterInput map[string]any
	if state.BatchSize == 1 && len(batch) == 1 {
		iterInput = map[string]any{
			"item":   batch[0],
			"index":  batchIndex,
			"total":  len(state.Items),
			"isLast": isLast,
		}
	} else {
		iterInput = map[string]any{
			"batch":  batch,
			"index":  batchIndex,
			"total":  len(state.Items),
			"isLast": isLast,
		}
	}

	runAction, iterThreadID, err := a.workflow.StartForEachIteration(state.NodeID, iterInput)
	if err != nil {
		a.Log().Error("foreach: failed to start iteration %d: %s", batchIndex, err)
		return
	}

	state.StartBatch(iterThreadID, batchIndex)
	a.iterThreadToForEach[iterThreadID] = state.ExecID.String()

	workflowPool := WorkflowFuncPoolName(a.workflow.ID())
	execFnMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), runAction, a.tracingProvider.InjectCarrier(a.spanCtx))
	if err := a.Send(workflowPool, execFnMsg); err != nil {
		a.Log().Error("foreach: failed to dispatch iteration %d: %s", batchIndex, err)
	}
}

// handleForEachIterationComplete is called when Next() returns Noop for a thread.
// It checks whether the thread belongs to a ForEach iteration; if so it records
// the completion and either spawns the next batch or finalises the loop.
// Returns true if the thread was a foreach iteration (caller should skip the
// normal checkWorkflowCompletion path).
func (a *WorkflowHandler) handleForEachIterationComplete(threadID uint16) bool {
	forEachExecIDStr, isForEach := a.iterThreadToForEach[threadID]
	if !isForEach {
		return false
	}
	delete(a.iterThreadToForEach, threadID)

	state, exists := a.forEachStates[forEachExecIDStr]
	if !exists {
		return true
	}

	iterResult := a.workflow.LastResultForThread(threadID)
	nextBatch, allDone := state.RecordCompletion(threadID, iterResult)

	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalForEachIterationCompleted,
		ThreadID: threadID,
		ExecID:   forEachExecIDStr,
		Data:     map[string]any{"batchesCompleted": state.Completed},
	})
	a.persistJournal()

	if allDone {
		delete(a.forEachStates, forEachExecIDStr)

		a.workflow.Journal().Append(internalworkflow.JournalEntry{
			Type:     internalworkflow.JournalForEachCompleted,
			ThreadID: state.ThreadID,
			ExecID:   forEachExecIDStr,
			Data:     map[string]any{"totalBatches": state.TotalBatches},
		})

		a.workflow.CompleteForEach(state.ExecID, state.Results)
		a.persistJournal()

		action := a.workflow.Next(state.ThreadID)
		if action.Type() == workflowactions.ActionNoop {
			a.checkWorkflowCompletion()
			return true
		}
		a.handleWorkflowAction(action)
		return true
	}

	if nextBatch >= 0 {
		a.spawnForEachBatch(state, nextBatch)
	}

	return true
}

// completeForEachImmediate handles the empty-items case by immediately setting
// the foreach result to empty and following the "done" edge.
func (a *WorkflowHandler) completeForEachImmediate(action *workflowactions.RunFunctionAction, items []any) {
	a.workflow.CompleteForEach(action.FunctionExecID, items)
	a.persistJournal()

	nextAction := a.workflow.Next(action.ThreadID)
	if nextAction.Type() == workflowactions.ActionNoop {
		a.checkWorkflowCompletion()
		return
	}
	a.handleWorkflowAction(nextAction)
}

// toAnySlice coerces value to []any.  It handles the common cases produced by
// JSON deserialisation (already []any) and typed slices passed programmatically.
func toAnySlice(value any) []any {
	if value == nil {
		return nil
	}
	if s, ok := value.([]any); ok {
		return s
	}
	// Fallback: unsupported type — return nil so the handler treats it as empty.
	return nil
}

// toPositiveInt coerces value to a positive int, returning fallback when
// the value is absent, zero, or cannot be converted.
func toPositiveInt(value any, fallback int) int {
	if value == nil {
		return fallback
	}
	var n int
	switch v := value.(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case float64:
		n = int(v)
	default:
		return fallback
	}
	if n < 1 {
		return fallback
	}
	return n
}

func (a *WorkflowHandler) handleMsgSubWorkflowCompleted(msg messaging.Message) error {
	completedMsg, ok := msg.Args.(messaging.SubWorkflowCompletedMessage)
	if !ok {
		return nil
	}

	if a.workflow.State() == internalworkflow.StateCancelled {
		a.Log().Warning("ignoring sub-workflow completed for cancelled workflow %s", a.workflow.ID())
		return nil
	}

	a.workflow.SetState(internalworkflow.StateRunning)

	outputStatus := workflow.FunctionSuccess
	if completedMsg.ChildFinalState != internalworkflow.StateFinished.String() {
		outputStatus = workflow.FunctionError
	}

	a.workflow.SetResultFor(completedMsg.ParentExecID, &workflow.FunctionResult{
		Output: workflow.FunctionOutput{
			Status: outputStatus,
			Data: map[string]any{
				"workflowId": completedMsg.ChildWorkflowID.String(),
				"status":     completedMsg.ChildFinalState,
				"output":     completedMsg.ChildOutput,
			},
		},
	})
	a.workflow.Journal().Append(internalworkflow.JournalEntry{
		Type:     internalworkflow.JournalSubWorkflowCompleted,
		ThreadID: completedMsg.ParentThreadID,
		ExecID:   completedMsg.ParentExecID.String(),
		Data: map[string]any{
			"childWorkflowId": completedMsg.ChildWorkflowID.String(),
			"childFinalState": completedMsg.ChildFinalState,
		},
	})
	a.persistJournal()

	action := a.workflow.Next(completedMsg.ParentThreadID)
	if action.Type() == workflowactions.ActionNoop {
		a.checkWorkflowCompletion()
		return nil
	}
	a.handleWorkflowAction(action)
	return nil
}
