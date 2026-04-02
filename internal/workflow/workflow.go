// Package workflow has all the types and functions for defining and handling Workflows
package workflow

import (
	"reflect"
	"strings"

	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"

	"github.com/open-source-cloud/fuse/internal/typeschema"
	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/open-source-cloud/fuse/pkg/strutil"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	// State defines the State type
	State string
)

func (s State) String() string {
	return string(s)
}

//goland:noinspection GoUnusedConst
const (
	// StateUntriggered Workflow untriggered state (new)
	StateUntriggered State = "untriggered"
	// StateRunning Workflow running state
	StateRunning State = "running"
	// StateSleeping Workflow sleeping state
	StateSleeping State = "sleeping"
	// StateFinished Workflow finished state (finished with success)
	StateFinished State = "finished"
	// StateError Workflow error state (finished with error)
	StateError State = "error"
	// StateCancelled Workflow cancelled state (terminated by user/system)
	StateCancelled State = "cancelled"
)

// New creates a new Workflow from an already generated ID and a provided WorkflowGraph
func New(id workflow.ID, graph *Graph) *Workflow {
	return &Workflow{
		id:               id,
		graph:            graph,
		journal:          NewJournal(),
		auditLog:         NewAuditLog(),
		retryTracker:     NewRetryTracker(),
		threads:          newThreads(),
		aggregatedOutput: store.New(),
		state: RunningState{
			currentState: StateUntriggered,
		},
	}
}

type (
	// Workflow defines a Workflow
	Workflow struct {
		id               workflow.ID
		graph            *Graph
		journal          *Journal
		auditLog         *AuditLog
		retryTracker     *RetryTracker
		threads          *threads
		aggregatedOutput *store.KV
		state            RunningState
	}

	// RunningState defines the Workflow running state
	RunningState struct {
		currentState State
	}
)

// Trigger triggers a new workflow, results in an Action to be acted upon by the responsible actor
func (w *Workflow) Trigger() workflowactions.Action {
	execID := workflow.NewExecID(0)
	triggerNode := w.graph.Trigger()

	triggerThread := w.threads.New(triggerNode.thread, execID)
	w.auditLog.NewEntry(triggerThread.ID(), triggerNode.ID(), execID.String(), nil)

	w.journal.Append(JournalEntry{
		Type:     JournalThreadCreated,
		ThreadID: triggerThread.ID(),
		ExecID:   execID.String(),
	})
	w.journal.Append(JournalEntry{
		Type:           JournalStepStarted,
		ThreadID:       triggerThread.ID(),
		FunctionNodeID: triggerNode.ID(),
		ExecID:         execID.String(),
	})

	return &workflowactions.RunFunctionAction{
		ThreadID:       triggerThread.ID(),
		FunctionID:     triggerNode.FunctionID(),
		FunctionExecID: execID,
		Args:           map[string]any{},
	}
}

// Resume resumes a previously started Workflow by replaying its journal entries
// to reconstruct state, then determining the next action to take.
func (w *Workflow) Resume() workflowactions.Action {
	entries := w.journal.Entries()
	if len(entries) == 0 {
		return &workflowactions.NoopAction{}
	}

	lastCompletedThreadIDs := w.replayJournalEntries(entries)
	return w.buildResumeAction(entries, lastCompletedThreadIDs)
}

// replayJournalEntries replays journal entries to reconstruct workflow state.
// Returns the IDs of threads that completed during replay.
func (w *Workflow) replayJournalEntries(entries []JournalEntry) []uint16 {
	var lastCompletedThreadIDs []uint16

	for _, entry := range entries {
		switch entry.Type {
		case JournalThreadCreated:
			execID := workflow.ExecID(entry.ExecID)
			w.threads.New(entry.ThreadID, execID)
		case JournalStepStarted:
			w.auditLog.NewEntry(entry.ThreadID, entry.FunctionNodeID, entry.ExecID, entry.Input)
		case JournalStepCompleted:
			w.SetResultFor(workflow.ExecID(entry.ExecID), entry.Result)
		case JournalThreadDone:
			t := w.threads.Get(entry.ThreadID)
			if t != nil {
				t.SetState(ThreadFinished)
			}
			lastCompletedThreadIDs = append(lastCompletedThreadIDs, entry.ThreadID)
		case JournalStateChanged:
			w.state.currentState = entry.State
		}
	}

	return lastCompletedThreadIDs
}

// buildResumeAction determines the next action after journal replay.
func (w *Workflow) buildResumeAction(entries []JournalEntry, lastCompletedThreadIDs []uint16) workflowactions.Action {
	pendingThreads := w.findPendingThreads(entries)

	if len(pendingThreads) == 0 {
		// All steps completed — try Next() on last finished threads to advance
		for _, threadID := range lastCompletedThreadIDs {
			action := w.Next(threadID)
			if action.Type() != workflowactions.ActionNoop {
				return action
			}
		}
		return &workflowactions.NoopAction{}
	}

	// Re-execute pending steps
	if len(pendingThreads) == 1 {
		return w.replayPendingThread(pendingThreads[0])
	}
	parallel := &workflowactions.RunParallelFunctionsAction{
		Actions: make([]*workflowactions.RunFunctionAction, 0, len(pendingThreads)),
	}
	for _, pt := range pendingThreads {
		action := w.replayPendingThread(pt)
		if runAction, ok := action.(*workflowactions.RunFunctionAction); ok {
			parallel.Actions = append(parallel.Actions, runAction)
		}
	}
	if len(parallel.Actions) == 0 {
		return &workflowactions.NoopAction{}
	}
	return parallel
}

type pendingThread struct {
	threadID       uint16
	functionNodeID string
	execID         string
	input          map[string]any
}

// findPendingThreads finds threads that have a StepStarted but no StepCompleted/StepFailed.
func (w *Workflow) findPendingThreads(entries []JournalEntry) []pendingThread {
	started := make(map[string]pendingThread) // execID -> pendingThread
	completed := make(map[string]bool)        // execID -> completed

	for _, entry := range entries {
		switch entry.Type {
		case JournalStepStarted:
			started[entry.ExecID] = pendingThread{
				threadID:       entry.ThreadID,
				functionNodeID: entry.FunctionNodeID,
				execID:         entry.ExecID,
				input:          entry.Input,
			}
		case JournalStepCompleted, JournalStepFailed:
			completed[entry.ExecID] = true
		}
	}

	var pending []pendingThread
	for execID, pt := range started {
		if !completed[execID] {
			pending = append(pending, pt)
		}
	}
	return pending
}

// replayPendingThread creates a RunFunctionAction for a thread that was in-progress when execution stopped.
func (w *Workflow) replayPendingThread(pt pendingThread) workflowactions.Action {
	node, err := w.graph.FindNode(pt.functionNodeID)
	if err != nil {
		return &workflowactions.NoopAction{}
	}
	return &workflowactions.RunFunctionAction{
		ThreadID:       pt.threadID,
		FunctionID:     node.FunctionID(),
		FunctionExecID: workflow.ExecID(pt.execID),
		Args:           pt.input,
	}
}

// Next requests the next Action to be enacted by the responsible actor on this workflow
func (w *Workflow) Next(threadID uint16) workflowactions.Action {
	currentThread := w.threads.Get(threadID)
	currentAuditEntry, _ := w.auditLog.Get(currentThread.CurrentExecID().String())
	currentNode, _ := w.graph.FindNode(currentAuditEntry.FunctionNodeID)

	switch len(currentNode.OutputEdges()) {
	case 0:
		currentThread.SetState(StateFinished)
		w.journal.Append(JournalEntry{
			Type:     JournalThreadDone,
			ThreadID: currentThread.ID(),
		})
		return &workflowactions.NoopAction{}
	case 1:
		edge := currentNode.OutputEdges()[0]
		if currentNode.thread != edge.To().thread {
			currentThread.SetState(StateFinished)
			w.journal.Append(JournalEntry{
				Type:     JournalThreadDone,
				ThreadID: currentThread.ID(),
			})
		}
		if !w.threads.AreAllParentsFinishedFor(edge.To().parentThreads) {
			return &workflowactions.NoopAction{}
		}
		return w.newRunFunctionAction(currentThread, edge)
	default: // >1
		return w.nextWithMultipleOutputEdges(currentThread, currentNode)
	}
}

func (w *Workflow) nextWithMultipleOutputEdges(currentThread *thread, currentNode *Node) workflowactions.Action {
	edges := w.filterOutputEdgesByConditionals(currentNode)

	edgeCount := len(edges)
	// if no edges after conditional filtering, just stop
	if edgeCount == 0 {
		return &workflowactions.NoopAction{}
	}
	// if we only have 1 output after filtering conditional edges, let's just run that one
	if edgeCount == 1 {
		if currentNode.thread != edges[0].To().thread {
			currentThread.SetState(StateFinished)
			w.journal.Append(JournalEntry{
				Type:     JournalThreadDone,
				ThreadID: currentThread.ID(),
			})
		}
		return w.newRunFunctionAction(currentThread, edges[0])
	}

	// more than 1 edge after filtering, let's run them in parallel
	parallelAction := &workflowactions.RunParallelFunctionsAction{
		Actions: make([]*workflowactions.RunFunctionAction, 0, len(currentNode.OutputEdges())),
	}
	currentThread.SetState(StateFinished)
	w.journal.Append(JournalEntry{
		Type:     JournalThreadDone,
		ThreadID: currentThread.ID(),
	})
	for _, edge := range edges {
		if !w.threads.AreAllParentsFinishedFor(edge.To().parentThreads) {
			return &workflowactions.NoopAction{}
		}
		parallelAction.Actions = append(parallelAction.Actions, w.newRunFunctionAction(currentThread, edge))
	}
	return parallelAction
}

func (w *Workflow) filterOutputEdgesByConditionals(currentNode *Node) []*Edge {
	if !currentNode.IsConditional() {
		return currentNode.OutputEdges()
	}

	var matchedEdges []*Edge
	var defaultEdge *Edge

	for _, edge := range currentNode.OutputEdges() {
		condition := edge.Condition()
		if condition == nil {
			matchedEdges = append(matchedEdges, edge)
			continue
		}

		if condition.Type == ConditionDefault {
			defaultEdge = edge
			continue
		}

		matches, err := EvaluateCondition(condition, w.aggregatedOutput, currentNode)
		if err != nil {
			log.Error().Err(err).Str("edge", edge.ID()).Msg("condition evaluation failed")
			continue
		}
		if matches {
			matchedEdges = append(matchedEdges, edge)
		}
	}

	// If no conditions matched and there's a default edge, use it
	if len(matchedEdges) == 0 && defaultEdge != nil {
		matchedEdges = append(matchedEdges, defaultEdge)
	}

	return matchedEdges
}

// SetResultFor sets the result of a function execution in the workflow's AuditLog
func (w *Workflow) SetResultFor(functionExecID workflow.ExecID, result *workflow.FunctionResult) {
	entry, exists := w.auditLog.Get(functionExecID.String())
	if !exists {
		return
	}
	entry.Result = result
	w.aggregatedOutput.Set(entry.FunctionNodeID, result.Output.Data)

	entryType := JournalStepCompleted
	if result.Output.Status != workflow.FunctionSuccess {
		entryType = JournalStepFailed
	}
	w.journal.Append(JournalEntry{
		Type:           entryType,
		ThreadID:       entry.ThreadID,
		FunctionNodeID: entry.FunctionNodeID,
		ExecID:         functionExecID.String(),
		Result:         result,
	})
}

// AggregatedOutputSnapshot returns a shallow copy of per-node outputs accumulated during execution.
func (w *Workflow) AggregatedOutputSnapshot() map[string]any {
	return w.aggregatedOutput.Snapshot()
}

// ID Workflow ID
func (w *Workflow) ID() workflow.ID {
	return w.id
}

// State Workflow state
func (w *Workflow) State() State {
	return w.state.currentState
}

// SetState changes Workflow state
func (w *Workflow) SetState(state State) {
	w.state.currentState = state
	w.journal.Append(JournalEntry{
		Type:  JournalStateChanged,
		State: state,
	})
}

// AllThreadsFinished returns true if all threads in this workflow have reached the finished state
func (w *Workflow) AllThreadsFinished() bool {
	return w.threads.AllFinished()
}

// Graph returns the workflow's graph
func (w *Workflow) Graph() *Graph {
	return w.graph
}

// Journal returns the workflow's execution journal
func (w *Workflow) Journal() *Journal {
	return w.journal
}

// AuditLog Workflow audit log
func (w *Workflow) AuditLog() *AuditLog {
	return w.auditLog
}

// Schema graph schema that defines the Workflow
func (w *Workflow) Schema() *GraphSchema {
	return w.graph.schema
}

// AuditLogJSON generates JSON from current AuditLog
func (w *Workflow) AuditLogJSON() string {
	json, _ := w.auditLog.MarshalJSON()
	return string(json)
}

// AuditLogTrace trace log helper to serialize AuditLog (only on trace level)
func (w *Workflow) AuditLogTrace() string {
	if zerolog.GlobalLevel() == zerolog.TraceLevel {
		return w.AuditLogJSON()
	}
	return ""
}

func (w *Workflow) newRunFunctionAction(currentThread *thread, edge *Edge) *workflowactions.RunFunctionAction {
	node := edge.To()
	execID := workflow.NewExecID(node.thread)

	newOrCurrentThread := currentThread
	var args map[string]any
	if currentThread.ID() != node.thread {
		newOrCurrentThread = w.threads.New(node.thread, execID)
		w.journal.Append(JournalEntry{
			Type:          JournalThreadCreated,
			ThreadID:      newOrCurrentThread.ID(),
			ExecID:        execID.String(),
			ParentThreads: node.parentThreads,
		})
		args = w.resolveJoinInputs(node)
	} else {
		currentThread.SetCurrentExecID(execID)
		args = w.inputMapping(edge, edge.Input())
	}

	w.auditLog.NewEntry(newOrCurrentThread.ID(), edge.To().ID(), execID.String(), args)
	w.journal.Append(JournalEntry{
		Type:           JournalStepStarted,
		ThreadID:       newOrCurrentThread.ID(),
		FunctionNodeID: edge.To().ID(),
		ExecID:         execID.String(),
		Input:          args,
	})
	return &workflowactions.RunFunctionAction{
		ThreadID:       newOrCurrentThread.ID(),
		FunctionID:     edge.To().FunctionID(),
		FunctionExecID: execID,
		Args:           args,
	}
}

func (w *Workflow) resolveJoinInputs(node *Node) map[string]any {
	// Collect branch inputs from all input edges
	var branchInputs []BranchInput
	for _, inputEdge := range node.InputEdges() {
		if inputEdge.To() == node {
			branchData := w.inputMapping(inputEdge, inputEdge.Input())
			branchInputs = append(branchInputs, BranchInput{
				EdgeID:   inputEdge.ID(),
				ThreadID: inputEdge.From().Thread(),
				Data:     branchData,
			})
		}
	}

	// Apply merge strategy
	mergeConfig := DefaultMergeConfig()
	if node.Schema().Merge != nil {
		mergeConfig = *node.Schema().Merge
	}
	return ApplyMergeStrategy(mergeConfig, branchInputs)
}

func (w *Workflow) inputMapping(edge *Edge, mappings []InputMapping) map[string]any {
	args := store.New()

	log.Debug().Msgf("mappings: %+v", mappings)
	log.Debug().Msgf("edge: %+v, from: %+v, to: %+v", edge, edge.From(), edge.To())

	for _, mapping := range mappings {
		nodeTo := edge.To()
		if nodeTo == nil {
			log.Error().Str("edge", edge.ID()).Str("param", mapping.MapTo).
				Msg("Node to is nil")
			break
		}
		nodeToMetadata := nodeTo.FunctionMetadata()
		if nodeToMetadata == nil {
			log.Error().Str("edge", edge.ID()).Str("param", mapping.MapTo).
				Msg("Node to metadata is nil")
			break
		}

		inputParamSchema, exists := nodeToMetadata.Input.Parameters[mapping.MapTo]
		if !nodeToMetadata.Input.CustomParameters && !exists {
			log.Warn().Str("edge", edge.ID()).Str("param", mapping.MapTo).
				Msg("Input ParamSchema not found for input mapping")
			continue
		}

		allowCustomInputParameters := nodeToMetadata.Input.CustomParameters

		switch mapping.Source {
		case SourceSchema:
			if !w.validateInputMapping(&inputParamSchema, mapping.Value) {
				log.Error().
					Str("edge", edge.ID()).
					Str("param", mapping.MapTo).
					Any("value", mapping.Value).
					Msg("Failed param validation")
				continue
			}
			args.Set(mapping.MapTo, mapping.Value)
		case SourceFlow:
			outputParamName := strutil.AfterFirstDot(mapping.Variable)

			nodeFrom := edge.From()
			if nodeFrom == nil {
				log.Error().Str("edge", edge.ID()).Str("param", outputParamName).
					Msg("Node from is nil")
				break
			}
			nodeFromMetadata := nodeFrom.FunctionMetadata()
			if nodeFromMetadata == nil {
				log.Error().Str("edge", edge.ID()).Str("param", outputParamName).
					Msg("Node from metadata is nil")
				break
			}

			outputParamSchema, exists := nodeFromMetadata.Output.Parameters[outputParamName]
			if !allowCustomInputParameters && !exists {
				log.Error().Str("edge", edge.ID()).Str("param", outputParamName).
					Msgf("Output ParamSchema not found for input mapping")
				continue
			}

			isArray := strings.HasPrefix(inputParamSchema.Type, "[]")
			rawValue := w.aggregatedOutput.Get(mapping.Variable)
			var value any
			if inputParamSchema.Type == "" && allowCustomInputParameters {
				value = rawValue
			} else {
				var err error
				value, err = typeschema.ParseValue(inputParamSchema.Type, rawValue)
				if err != nil {
					log.Error().
						Err(err).
						Str("edge", edge.ID()).
						Str("param", mapping.MapTo).
						Any("value", mapping.Value).
						Msg("Error parsing value")
					continue
				}
			}

			if !w.validateInputMapping(&outputParamSchema, value) {
				log.Error().
					Str("edge", edge.ID()).
					Str("param", mapping.MapTo).
					Any("value", value).
					Msg("Failed param validation")
				continue
			}

			if isArray {
				currentArray := args.Get(mapping.MapTo)
				if currentArray != nil {
					currentArraySlice := reflect.ValueOf(currentArray)
					valueSlice := reflect.ValueOf(value)
					args.Set(mapping.MapTo, reflect.AppendSlice(currentArraySlice, valueSlice).Interface())
				} else {
					args.Set(mapping.MapTo, value)
				}
				continue
			}

			// if the value is nil, and there is a default value, use the default value
			if value == nil && inputParamSchema.Default != nil {
				value = inputParamSchema.Default
			}

			args.Set(mapping.MapTo, value)
		}
	}

	log.Debug().Msgf("Args: %+v", args.Raw())

	return args.Raw()
}

func (w *Workflow) validateInputMapping(schema *workflow.ParameterSchema, value any) bool {
	return ValidateInputMapping(schema, value) == nil
}

// HandleNodeFailure handles a function failure with retry policy and error edge routing.
// Returns nil if the workflow should transition to StateError.
func (w *Workflow) HandleNodeFailure(threadID uint16, execID workflow.ExecID) workflowactions.Action {
	entry, exists := w.auditLog.Get(execID.String())
	if !exists {
		return nil
	}
	node, err := w.graph.FindNode(entry.FunctionNodeID)
	if err != nil {
		return nil
	}

	retryPolicy := w.getRetryPolicy(node)
	if retryPolicy != nil {
		attempts := w.retryTracker.Increment(execID.String())
		if attempts <= retryPolicy.MaxAttempts {
			delay := retryPolicy.DelayFor(attempts - 1)
			w.journal.Append(JournalEntry{
				Type:           JournalStepRetrying,
				ThreadID:       threadID,
				FunctionNodeID: entry.FunctionNodeID,
				ExecID:         execID.String(),
			})
			return &workflowactions.RetryFunctionAction{
				RunFunctionAction: workflowactions.RunFunctionAction{
					ThreadID:       threadID,
					FunctionID:     node.FunctionID(),
					FunctionExecID: execID,
					Args:           entry.Input,
				},
				Delay:   delay,
				Attempt: attempts,
			}
		}
	}

	// Retries exhausted — check for error edges
	w.retryTracker.Clear(execID.String())
	errorEdges := w.findErrorEdges(node)
	if len(errorEdges) > 0 {
		return w.newRunFunctionAction(w.threads.Get(threadID), errorEdges[0])
	}

	// No error edges — caller sets StateError
	return nil
}

func (w *Workflow) getRetryPolicy(node *Node) *RetryPolicy {
	return node.Schema().Retry
}

func (w *Workflow) findErrorEdges(node *Node) []*Edge {
	var errorEdges []*Edge
	for _, edge := range node.OutputEdges() {
		if edge.schema.OnError {
			errorEdges = append(errorEdges, edge)
		}
	}
	return errorEdges
}
