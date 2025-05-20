package workflow

import (
	"github.com/open-source-cloud/fuse/internal/typeschema"
	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/open-source-cloud/fuse/pkg/utils"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"reflect"
	"strings"
)

type (
	State      string
	ID         string
	ActionType string
)

func (s State) String() string {
	return string(s)
}
func (id ID) String() string {
	return string(id)
}
func NewID() ID {
	return ID(uuid.V7())
}

const (
	StateUntriggered State = "untriggered"
	StateRunning     State = "running"
	StateSleeping    State = "sleeping"
	StateFinished    State = "finished"
	StateError       State = "error"
)

func New(id ID, graph *Graph) *Workflow {
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
	Workflow struct {
		id               ID
		graph            *Graph
		auditLog         *AuditLog
		threads          *threads
		aggregatedOutput *store.KV
		state            RunningState
	}

	RunningState struct {
		currentState State
	}
)

func (w *Workflow) Trigger() Action {
	execID := uuid.V7()
	triggerNode := w.graph.Trigger()

	triggerThread := w.threads.New(triggerNode.thread, execID)
	w.auditLog.NewEntry(triggerThread.ID(), triggerNode.ID(), execID, nil)

	return &RunFunctionAction{
		ThreadID:       triggerThread.ID(),
		FunctionID:     triggerNode.FunctionID(),
		FunctionExecID: execID,
		Args:           map[string]any{},
	}
}

func (w *Workflow) Next(threadID int) Action {
	currentThread := w.threads.Get(threadID)
	currentAuditEntry, _ := w.auditLog.Get(currentThread.CurrentExecID())
	currentNode, _ := w.graph.FindNode(currentAuditEntry.FunctionNodeID)

	switch len(currentNode.OutputEdges()) {
	case 0:
		currentThread.SetState(StateFinished)
		// TODO: if ALL threads are finished, finish actor-tree for this workflow
		return &NoopAction{}
	case 1:
		edge := currentNode.OutputEdges()[0]
		if currentNode.thread != edge.To().thread {
			currentThread.SetState(StateFinished)
		}
		if !w.threads.AreAllParentsFinishedFor(edge.To().parentThreads) {
			return &NoopAction{}
		}
		return w.newRunFunctionAction(currentThread, edge)
	default: // >1
		currentThread.SetState(StateFinished)
		parallelAction := &RunParallelFunctionsAction{
			Actions: make([]*RunFunctionAction, 0, len(currentNode.OutputEdges())),
		}
		for _, edge := range currentNode.OutputEdges() {
			if !w.threads.AreAllParentsFinishedFor(edge.To().parentThreads) {
				return &NoopAction{}
			}
			parallelAction.Actions = append(parallelAction.Actions, w.newRunFunctionAction(currentThread, edge))
		}
		return parallelAction
	}
}

func (w *Workflow) SetResultFor(functionExecID string, result *workflow.FunctionResult) {
	entry, exists := w.auditLog.Get(functionExecID)
	if !exists {
		return
	}
	entry.Result = result
	w.aggregatedOutput.Set(entry.FunctionNodeID, result.Output.Data)
}

func (w *Workflow) ID() ID {
	return w.id
}

func (w *Workflow) State() State {
	return w.state.currentState
}

func (w *Workflow) SetState(state State) {
	w.state.currentState = state
}

func (w *Workflow) AuditLog() *AuditLog {
	return w.auditLog
}

func (w *Workflow) Schema() *GraphSchema {
	return w.graph.schema
}

func (w *Workflow) AuditLogJSON() string {
	json, _ := w.auditLog.MarshalJSON()
	return string(json)
}

func (w *Workflow) AuditLogTrace() string {
	if zerolog.GlobalLevel() == zerolog.TraceLevel {
		return w.AuditLogJSON()
	}
	return ""
}

func (w *Workflow) newRunFunctionAction(currentThread *thread, edge *Edge) *RunFunctionAction {
	node := edge.To()
	execID := uuid.V7()

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

	w.auditLog.NewEntry(newOrCurrentThread.ID(), edge.To().ID(), execID, args)
	return &RunFunctionAction{
		ThreadID:       newOrCurrentThread.ID(),
		FunctionID:     edge.To().FunctionID(),
		FunctionExecID: execID,
		Args:           args,
	}
}

func (w *Workflow) inputMapping(edge *Edge, mappings []InputMapping) map[string]any {
	args := store.New()
	for _, mapping := range mappings {
		inputParamSchema, exists := edge.To().FunctionMetadata().Input.Parameters[mapping.MapTo]
		if !exists && !edge.To().FunctionMetadata().Input.CustomParameters {
			log.Error().Str("edge", edge.ID()).Str("param", mapping.MapTo).
				Msg("Input ParamSchema not found for input mapping")
		}

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
		case SourceEdges:
			outputParamName := utils.AfterFirstDot(mapping.Variable)
			outputParamSchema, exists := edge.From().FunctionMetadata().Output.Parameters[outputParamName]
			if !exists {
				log.Error().Str("edge", edge.ID()).Str("param", outputParamName).
					Msgf("Output ParamSchema not found for input mapping")
				continue
			}

			isArray := strings.HasPrefix(inputParamSchema.Type, "[]")
			rawValue := w.aggregatedOutput.Get(mapping.Variable)
			value, err := typeschema.ParseValue(inputParamSchema.Type, rawValue)
			if err != nil {
				log.Error().
					Err(err).
					Str("edge", edge.ID()).
					Str("param", mapping.MapTo).
					Any("value", mapping.Value).
					Msg("Error parsing value")
				continue
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
			args.Set(mapping.MapTo, value)
		}
	}
	return args.Raw()
}

func (w *Workflow) validateInputMapping(paramSchema *workflow.ParameterSchema, value any) bool {
	return true
}
