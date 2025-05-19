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
		// TODO - finished this thread
		currentThread.SetState(StateFinished)
		return &NoopAction{}
	case 1:
		edge := currentNode.OutputEdges()[0]
		node := edge.To()
		if currentNode.thread != node.thread {
			currentThread.SetState(StateFinished)
		}
		if !w.threads.AreAllParentsFinishedFor(node.parentThreads) {
			return &NoopAction{}
		}

		var mappings []InputMapping
		if threadID != node.thread {
			for _, inputEdge := range node.InputEdges() {
				if inputEdge.To() == node {
					mappings = append(mappings, inputEdge.Input()...)
				}
			}
		} else {
			mappings = edge.Input()
		}

		execID := uuid.V7()
		args := w.inputMapping(edge, mappings)

		currentThread.SetCurrentExecID(execID)
		w.auditLog.NewEntry(currentThread.ID(), edge.To().ID(), execID, args)
		return &RunFunctionAction{
			ThreadID:       currentThread.ID(),
			FunctionID:     edge.To().FunctionID(),
			FunctionExecID: execID,
			Args:           args,
		}
	default: // >1
		currentThread.SetState(StateFinished)
		parallelAction := &RunParallelFunctionsAction{
			Actions: make([]*RunFunctionAction, 0, len(currentNode.OutputEdges())),
		}
		for _, edge := range currentNode.OutputEdges() {
			node := edge.To()
			if !w.threads.AreAllParentsFinishedFor(node.parentThreads) {
				return &NoopAction{}
			}

			var mappings []InputMapping
			if threadID != node.thread {
				for _, inputEdge := range node.InputEdges() {
					if inputEdge.To() == node {
						mappings = append(mappings, inputEdge.Input()...)
					}
				}
			} else {
				mappings = edge.Input()
			}

			execID := uuid.V7()
			args := w.inputMapping(edge, mappings)

			newParallelThread := w.threads.NewChild(node.thread, execID, node.parentThreads)
			w.auditLog.NewEntry(newParallelThread.ID(), edge.To().ID(), execID, args)
			parallelAction.Actions = append(parallelAction.Actions, &RunFunctionAction{
				ThreadID:       newParallelThread.ID(),
				FunctionID:     edge.To().FunctionID(),
				FunctionExecID: execID,
				Args:           args,
			})
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
			log.Info().Msg(outputParamSchema.Name)

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

//func (w *workflowWorker) createNodeInput(node graph.Node, rawInputData map[string]any) (*workflow.FunctionInput, error) {
//	for i, mapping := range nodeConfig.InputMapping() {
//		paramSchema, exists := inputSchema.Parameters[mapping.Mapping]
//		audit.Debug().Workflow(w.id).Node(node.id()).

//		isCustomParameter := inputSchema.CustomParameters && !exists
//		if mapping.Source != graph.InputSourceSchema && !isCustomParameter && !exists {
//			audit.Error().Workflow(w.id).Node(node.id()).
//				Str("source", mapping.Source).Any("origin", mapping.Origin).Msg("Input mapping for source.origin not found")
//			return nil, fmt.Errorf("input mapping for source.origin %s.%s not found", mapping.Source, mapping.Origin)
//		}
//
//		var rawValue any
//		if mapping.Source == graph.InputSourceSchema {
//			rawValue = mapping.Origin
//		} else {
//			inputKey := mapping.Origin.(string)
//			if len(node.InputEdges()) > 1 {
//				inputKey = fmt.Sprintf("%s.%s", mapping.Source, mapping.Origin)
//			}
//			audit.Debug().Workflow(w.id).Node(node.id()).Msgf("inputKey: %v", inputKey)
//			rawValue = inputStore.Get(inputKey)
//			if rawValue == nil {
//				audit.Error().Workflow(w.id).Node(node.id()).
//					Str("inputKey", mapping.Source).Msg("Input value for source not found")
//				return nil, fmt.Errorf("input value for inputKey %s not found", inputKey)
//			}
//		}
//
//		isArray := strings.HasPrefix(paramSchema.Type, "[]")
//		var paramValue any
//		if mapping.Source == graph.InputSourceSchema || isCustomParameter {
//			paramValue = rawValue
//		} else {
//			paramValue, err = typeschema.ParseValue(paramSchema.Type, rawValue)
//			if err != nil {
//				audit.Error().Workflow(w.id).Node(node.id()).Err(err).Msg("Failed to parse input value")
//				return nil, err
//			}
//		}
//
//		// TODO: Improve set handling
//		if isArray {
//			currentArray := nodeInput.Get(mapping.Mapping)
//			if currentArray != nil {
//				nodeInput.Set(mapping.Mapping, append(currentArray.([]any), paramValue.([]any)...))
//			} else {
//				nodeInput.Set(mapping.Mapping, paramValue.([]any))
//			}
//		} else {
//			nodeInput.Set(mapping.Mapping, paramValue)
//		}
//
//	}
//
//	audit.Debug().Workflow(w.id).Node(node.id()).Msgf("FunctionInput: %v", nodeInput.ToMap())
//
//	return nodeInput, nil
//}
