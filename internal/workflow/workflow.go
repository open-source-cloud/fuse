// Package workflow has all the types and functions for defining and handling Workflows
package workflow

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"

	"github.com/open-source-cloud/fuse/internal/typeschema"
	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/open-source-cloud/fuse/pkg/utils"
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
)

// New creates a new Workflow from an already generated ID and a provided WorkflowGraph
func New(id workflow.ID, graph *Graph) *Workflow {
	return &Workflow{
		id:               id,
		graph:            graph,
		auditLog:         NewAuditLog(),
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
		auditLog         *AuditLog
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

	return &workflowactions.RunFunctionAction{
		ThreadID:       triggerThread.ID(),
		FunctionID:     triggerNode.FunctionID(),
		FunctionExecID: execID,
		Args:           map[string]any{},
	}
}

// Resume resumes a previously started Workflow that needed to be re-created from data
func (w *Workflow) Resume() workflowactions.Action {
	// TODO add logic to re-start an already started Workflow that got reloaded from storage
	return nil
}

// Next requests the next Action to be enacted by the responsible actor on this workflow
func (w *Workflow) Next(threadID uint16) workflowactions.Action {
	currentThread := w.threads.Get(threadID)
	currentAuditEntry, _ := w.auditLog.Get(currentThread.CurrentExecID().String())
	currentNode, _ := w.graph.FindNode(currentAuditEntry.FunctionNodeID)

	switch len(currentNode.OutputEdges()) {
	case 0:
		currentThread.SetState(StateFinished)
		// TODO: if ALL threads are finished, finish actor-tree for this workflow
		return &workflowactions.NoopAction{}
	case 1:
		edge := currentNode.OutputEdges()[0]
		if currentNode.thread != edge.To().thread {
			currentThread.SetState(StateFinished)
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
		}
		return w.newRunFunctionAction(currentThread, edges[0])
	}

	// more than 1 edge after filtering, let's run them in parallel
	parallelAction := &workflowactions.RunParallelFunctionsAction{
		Actions: make([]*workflowactions.RunFunctionAction, 0, len(currentNode.OutputEdges())),
	}
	currentThread.SetState(StateFinished)
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
	conditionalEdges := currentNode.FunctionMetadata().Output.Edges
	conditionalSource := currentNode.FunctionMetadata().Output.ConditionalOutputField
	conditionalValue := w.aggregatedOutput.Get(fmt.Sprintf("%s.%s", currentNode.ID(), conditionalSource))

	edges := make([]*Edge, 0, len(currentNode.OutputEdges()))
	for _, edge := range currentNode.OutputEdges() {
		edgeCondition := edge.Condition()
		if edgeCondition.Value == conditionalValue {
			_, exists := conditionalEdges[edgeCondition.Name]
			if !exists {
				log.Error().Str("edge", edge.ID()).Str("condition", edgeCondition.Name).
					Msg("Conditional edge not found")
				continue
			}
			edges = append(edges, edge)
		}
	}
	return edges
}

// SetResultFor sets the result of a function execution in the workflow's AuditLog
func (w *Workflow) SetResultFor(functionExecID workflow.ExecID, result *workflow.FunctionResult) {
	entry, exists := w.auditLog.Get(functionExecID.String())
	if !exists {
		return
	}
	entry.Result = result
	w.aggregatedOutput.Set(entry.FunctionNodeID, result.Output.Data)
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
	var mappings []InputMapping
	if currentThread.ID() != node.thread {
		newOrCurrentThread = w.threads.New(node.thread, execID)
		for _, inputEdge := range node.InputEdges() {
			if inputEdge.To() == node {
				mappings = append(mappings, inputEdge.Input()...)
			}
		}
	} else {
		currentThread.SetCurrentExecID(execID)
		mappings = edge.Input()
	}
	args := w.inputMapping(edge, mappings)

	w.auditLog.NewEntry(newOrCurrentThread.ID(), edge.To().ID(), execID.String(), args)
	return &workflowactions.RunFunctionAction{
		ThreadID:       newOrCurrentThread.ID(),
		FunctionID:     edge.To().FunctionID(),
		FunctionExecID: execID,
		Args:           args,
	}
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
			outputParamName := utils.AfterFirstDot(mapping.Variable)

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

func (w *Workflow) validateInputMapping(_ *workflow.ParameterSchema, _ any) bool {
	// TODO implement input mapping validations
	return true
}
