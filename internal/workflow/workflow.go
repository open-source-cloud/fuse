package workflow

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/internal/typeschema"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"github.com/vladopajic/go-actor/actor"
	"regexp"
	"strings"
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
		log.Info().
			Str("workflow", w.id).
			Any("state", w.state).
			Msg("Stopping")
		return actor.WorkerEnd

	case msg := <-w.mailbox.ReceiveC():
		log.Info().
			Str("workflow", w.id).
			Any("msg", msg.Type()).
			Any("data", msg.Data()).
			Msg("Message received")

		w.handleMessage(ctx, msg)
		return actor.WorkerContinue
	}
}

func (w *workflowWorker) SendMessage(ctx Context, msg Message) {
	err := w.mailbox.Send(ctx, msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("workflow", w.id).
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
			log.Error().
				Str("workflow", w.id).
				Msg("No current node")
			return
		}

		currentNodeLogLabel := "node"
		var currentNodeIDs any
		currentNodeIDs = w.currentNode[0]
		if currentNodeCount > 1 {
			currentNodeLogLabel = "nodes"
			currentNodeIDs = w.currentNode
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
			log.Info().
				Str("workflow", w.id).
				Any(currentNodeLogLabel, currentNodeIDs).
				Msg("No output edges")
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
		log.Warn().
			Str("workflow", w.id).
			Any("msg", msg.Type()).
			Msg("Received unknown message type")
	}
}

func (w *workflowWorker) executeNode(node graph.Node, rawInputData any) workflow.NodeOutputData {
	input, _ := w.processRawInput(node, rawInputData)
	w.currentNode = []graph.Node{node}
	result, _ := node.NodeRef().Execute(input)

	var output workflow.NodeOutput
	_, isAsync := result.Async()
	log.Info().
		Str("workflow", w.id).
		Str("node", node.ID()).
		Any("input", input).
		Any("output", map[string]any{
			"async":  isAsync,
			"status": result.Output().Status(),
			"data":   result.Output().Data(),
		}).
		Msg("node executed")
	if !isAsync {
		output = result.Output()
		if output.Status() == workflow.NodeOutputStatusSuccess {
			return output.Data()
		}
		w.state = StateError
	} else {
		log.Warn().
			Str("workflow", w.id).
			Str("node", node.ID()).
			Any("input", input).
			Any("output", map[string]any{
				"async":  isAsync,
				"status": result.Output().Status(),
				"data":   result.Output().Data(),
			}).
			Msg("node is async (TODO)")
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

		var output workflow.NodeOutput
		_, isAsync := result.Async()
		log.Info().
			Str("workflow", w.id).
			Str("node", node.ID()).
			Any("input", input).
			Any("output", map[string]any{
				"async":  isAsync,
				"status": result.Output().Status(),
				"data":   result.Output().Data(),
			}).
			Msg("node executed")
		if !isAsync {
			output = result.Output()
			if output.Status() == workflow.NodeOutputStatusSuccess {
				aggregatedOutput[edge.ID()] = output.Data()
			} else {
				status = output.Status()
				break
			}
		} else {
			log.Warn().
				Str("workflow", w.id).
				Str("node", node.ID()).
				Any("input", input).
				Any("output", result).
				Msg("node is async (TODO)")
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
		paramSchema, exists := inputSchema.Parameters[mapping.Mapping]
		if !exists {
			log.Error().Msgf("Workflow %s : Input mapping for parameter %s not found", w.id, mapping.ParamName)
			return nil, fmt.Errorf("input mapping for parameter %s not found", mapping.ParamName)
		}
		isArray := strings.HasPrefix(paramSchema.Type, "[]")

		param, exists := nodeInputData[mapping.ParamName]
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
