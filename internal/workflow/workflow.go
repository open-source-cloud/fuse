package workflow

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/audit"
	"regexp"
	"strings"

	"github.com/open-source-cloud/fuse/internal/typeschema"
	"github.com/open-source-cloud/fuse/pkg/graph"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"github.com/vladopajic/go-actor/actor"
)

// State type for workflow states
type State string

const (
	// StateStopped workflow is stopped
	StateStopped State = "stopped"
	// StateRunning workflow is running
	StateRunning State = "running"
	// StateFinished workflow has finished successfully
	StateFinished State = "finished"
	// StateError workflow has finished with errors
	StateError State = "error"
)

// Context the context type used by workflow actors
type Context actor.Context

// Workflow describes the interface for a Workflow actor
type Workflow interface {
	actor.Actor
	SendMessage(ctx Context, msg Message)
}

type workflowWorker struct {
	baseActor   actor.Actor
	mailbox     actor.Mailbox[Message]
	id          string
	schema      Schema
	data        map[string]any
	state       State
	currentNode []graph.Node
}

// NewWorkflow creates a new workflow actor worker
func NewWorkflow(id string, schema Schema) Workflow {
	worker := &workflowWorker{
		mailbox:     actor.NewMailbox[Message](),
		id:          id,
		schema:      schema,
		data:        make(map[string]any),
		state:       StateStopped,
		currentNode: []graph.Node{},
	}
	worker.baseActor = actor.New(worker)
	return worker
}

func (w *workflowWorker) Start() {
	w.state = StateRunning
	w.baseActor.Start()
}

func (w *workflowWorker) Stop() {
	w.baseActor.Stop()
	w.state = StateStopped
}

func (w *workflowWorker) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		audit.Info().WorkflowState(w.id, w.state).Msg("Stopping")
		return actor.WorkerEnd

	case msg := <-w.mailbox.ReceiveC():
		audit.Info().WorkflowMessage(w.id, msg.Type(), msg.Data()).Msg("Message received")
		w.handleMessage(ctx, msg)
		return actor.WorkerContinue
	}
}

func (w *workflowWorker) SendMessage(ctx Context, msg Message) {
	err := w.mailbox.Send(ctx, msg)
	if err != nil {
		audit.Error().
			Workflow(w.id).
			Err(err).
			Msg("Failed to send message to Workflow")
	}
	w.mailbox.Start()
}

func (w *workflowWorker) handleMessage(ctx Context, msg Message) {
	switch msg.Type() {
	case MessageStartWorkflow:
		rootNode := w.schema.RootNode()
		output := w.executeNode(rootNode, msg.Data())
		w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, output))

	case MessageContinueWorkflow:
		currentNodeCount := len(w.currentNode)
		if currentNodeCount == 0 {
			audit.Error().Workflow(w.id).Msg("No current node")
			return
		}

		outputEdges := map[string]graph.Edge{}
		for _, node := range w.currentNode {
			nodeOutputEdges := node.OutputEdges()
			for k, edge := range nodeOutputEdges {
				outputEdges[k] = edge
			}
		}
		var output workflow.NodeOutputData
		switch len(outputEdges) {
		case 0:
			audit.Info().Workflow(w.id).Nodes(w.currentNode).Msg("No output edges")
			w.state = StateFinished
			ctx.Done()
			return
		case 1:
			var edge graph.Edge
			for _, edgeRef := range outputEdges {
				edge = edgeRef
				break
			}
			//goland:noinspection ALL
			output = w.executeNode(edge.To(), msg.Data())
		default:
			output = w.executeParallelNodes(outputEdges, msg.Data())
		}

		w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, output))

	default:
		// Handle unknown message types
		audit.Warn().WorkflowMessage(w.id, msg.Type(), msg.Data()).Msg("Unknown message type")
	}
}

func (w *workflowWorker) executeNode(node graph.Node, rawInputData any) workflow.NodeOutputData {
	input, _ := w.processRawInput(node, rawInputData)
	w.currentNode = []graph.Node{node}
	result, _ := node.NodeRef().Execute(input)

	audit.Info().Workflow(w.id).NodeInputOutput(node.ID(), input, result.Map()).Msg("node executed")

	var output workflow.NodeOutput
	_, isAsync := result.Async()
	if !isAsync {
		output = result.Output()
		if output.Status() == workflow.NodeOutputStatusSuccess {
			return output.Data()
		}
		w.state = StateError
	} else {
		audit.Warn().Workflow(w.id).NodeInputOutput(node.ID(), input, result.Map()).Msg("node is async (TODO)")
	}

	return nil
}

func (w *workflowWorker) executeParallelNodes(outputEdges map[string]graph.Edge, rawInputData any) workflow.NodeOutputData {
	aggregatedOutput := make(map[string]any)
	w.currentNode = make([]graph.Node, 0, len(outputEdges))
	status := workflow.NodeOutputStatusSuccess
	for _, edge := range outputEdges {
		node := edge.To()
		input, _ := w.processRawInput(node, rawInputData)
		w.currentNode = append(w.currentNode, node)
		result, _ := node.NodeRef().Execute(input)

		audit.Info().Workflow(w.id).NodeInputOutput(node.ID(), input, result.Map()).Msg("node executed")
		var output workflow.NodeOutput
		_, isAsync := result.Async()
		if !isAsync {
			output = result.Output()
			if output.Status() == workflow.NodeOutputStatusSuccess {
				aggregatedOutput[edge.ID()] = output.Data()
			} else {
				status = output.Status()
				break
			}
		} else {
			audit.Warn().Workflow(w.id).NodeInputOutput(node.ID(), input, result.Map()).Msg("node is async (TODO)")
		}
	}

	if status == workflow.NodeOutputStatusSuccess {
		return aggregatedOutput
	}

	return nil
}

func (w *workflowWorker) processRawInput(node graph.Node, rawInputData any) (map[string]any, error) {
	inputData := rawInputData.(map[string]any)
	nodeConfig := node.Config()
	inputSchema := node.NodeRef().Metadata().Input()
	processedInput := make(map[string]any)

	for _, mapping := range nodeConfig.InputMapping() {
		nodeInputData := inputData
		re := regexp.MustCompile(`(\w+)\[([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})]`)
		matches := re.FindStringSubmatch(mapping.Source)
		if len(matches) > 2 {
			uuid := matches[2]
			nodeInputData = inputData[uuid].(map[string]any)
		}

		log.Debug().Msgf("processRawInput.mapping.Source: %v", mapping.Source)
		log.Debug().Msgf("processRawInput.mapping.ParamName: %v", mapping.ParamName)
		log.Debug().Msgf("processRawInput.mapping.Mapping: %v", mapping.Mapping)

		log.Debug().Msgf("processRawInput.nodeInputData: %v", nodeInputData)
		log.Debug().Msgf("processRawInput.mapping: %v", mapping)
		log.Debug().Msgf("processRawInput.inputSchema: %v", inputSchema)

		paramSchema, exists := inputSchema.Parameters[mapping.Mapping]

		log.Debug().Msgf("processRawInput.paramSchema: %v", paramSchema)
		log.Debug().Msgf("processRawInput.exists: %v", exists)

		if !exists {
			log.Error().Msgf("Workflow %s : Input mapping for parameter %s not found", w.id, mapping.ParamName)
			return nil, fmt.Errorf("input mapping for parameter %s not found", mapping.ParamName)
		}
		isArray := strings.HasPrefix(paramSchema.Type, "[]")

		param, exists := nodeInputData[mapping.ParamName]
		log.Debug().Msgf("processRawInput.param: %v", param)
		log.Debug().Msgf("processRawInput.exists: %v", exists)

		if !exists {
			log.Error().Msgf("Workflow %s : Input mapping for parameter %s not found", w.id, mapping.ParamName)
			return nil, fmt.Errorf("input mapping for parameter %s not found", mapping.ParamName)
		}

		paramValue, _ := typeschema.ParseValue(paramSchema.Type, param)
		if isArray {
			currentArray := processedInput[mapping.Mapping]
			if currentArray != nil {
				processedInput[mapping.Mapping] = append(currentArray.([]any), paramValue.([]any)...)
			} else {
				processedInput[mapping.Mapping] = paramValue.([]any)
			}
		} else {
			processedInput[mapping.Mapping] = paramValue
		}
	}

	return processedInput, nil
}
