package workflow

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/audit"
	"github.com/open-source-cloud/fuse/internal/typeschema"
	"github.com/open-source-cloud/fuse/pkg/graph"
	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/vladopajic/go-actor/actor"
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
	audit.Info().WorkflowState(w.id, w.state).Msg("Stop() called")
	w.state = StateStopped
}

func (w *workflowWorker) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		audit.Info().WorkflowState(w.id, w.state).Msg("Stopped")
		return actor.WorkerEnd

	case msg := <-w.mailbox.ReceiveC():
		audit.Info().WorkflowMessage(w.id, msg.Type(), msg.Data()).Msg("Message received")
		if err := w.handleMessage(ctx, msg); err != nil {
			audit.Info().WorkflowState(w.id, w.state).Msg("Stopped")
			return actor.WorkerEnd
		}
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

func (w *workflowWorker) handleMessage(ctx Context, msg Message) error {
	switch msg.Type() {
	case MessageStartWorkflow:
		rootNode := w.schema.RootNode()
		output, err := w.executeNode(ctx, rootNode, msg.Data())
		if err != nil {
			audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute root node")
			w.state = StateError
			return err
		}
		w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, output))
	case MessageContinueWorkflow:
		currentNodeCount := len(w.currentNode)
		if currentNodeCount == 0 {
			err := fmt.Errorf("no current node")
			audit.Error().Workflow(w.id).Err(err).Msg("No current node")
			return err
		}

		outputEdges := map[string]graph.Edge{}
		for _, node := range w.currentNode {
			outputMetadata := node.NodeRef().Metadata().Output()
			nodeOutputEdges := node.OutputEdges()
			for k, edge := range nodeOutputEdges {
				if edge.IsConditional() && outputMetadata.ConditionalOutput {
					inputData := msg.Data()
					edgeMetadata := outputMetadata.Edges[edge.Condition().Name]
					if edge.Condition().Value == inputData[edgeMetadata.ConditionalEdge.Condition] {
						outputEdges[k] = edge
					} else {
						audit.Debug().Workflow(w.id).Msg("Conditional edge not met")
					}
				} else {
					outputEdges[k] = edge
				}
			}
		}
		switch len(outputEdges) {
		case 0:
			audit.Info().Workflow(w.id).Nodes(w.currentNode).Msg("No output edges")
			w.state = StateFinished
			return fmt.Errorf("no output edges")
		case 1:
			var edge graph.Edge
			for _, edgeRef := range outputEdges {
				edge = edgeRef
				break
			}
			//goland:noinspection ALL
			output, err := w.executeNode(ctx, edge.To(), msg.Data())
			if err != nil {
				audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute root node")
				w.state = StateError
				return err
			}
			if output != nil {
				w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, output))
			}
		default:
			output, err := w.executeParallelNodes(ctx, outputEdges, msg.Data())
			if err != nil {
				audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute parallel nodes")
				w.state = StateError
				return err
			}
			w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, output))
		}

	default:
		// Handle unknown message types
		audit.Warn().WorkflowMessage(w.id, msg.Type(), msg.Data()).Msg("Unknown message type")
		return fmt.Errorf("unknown message type %s", msg.Type())
	}

	return nil
}

func (w *workflowWorker) executeNode(ctx Context, node graph.Node, rawInputData map[string]any) (workflow.NodeOutputData, error) {
	input, err := w.createNodeInput(node, rawInputData)
	if err != nil {
		audit.Error().Workflow(w.id).Err(err).Msg("Failed to create node input")
		w.state = StateError
		return nil, err
	}

	nodeRef := node.NodeRef()
	w.currentNode = []graph.Node{node}
	result, err := nodeRef.Execute(input)
	if err != nil {
		audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute node")
		w.state = StateError
		return nil, err
	}

	audit.Info().Workflow(w.id).NodeInputOutput(node.ID(), input.Raw(), result.Map()).Msg("node executed")

	var output workflow.NodeOutput
	async, isAsync := result.Async()

	if isAsync {
		go func() {
			done := <-async
			if done.Status() == workflow.NodeOutputStatusSuccess {
				w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, done.Data()))
			}
		}()
		return nil, nil
	}

	output = result.Output()
	if output.Status() == workflow.NodeOutputStatusSuccess {
		return output.Data(), nil
	}

	w.state = StateError
	// TODO: Improve error handling
	return nil, fmt.Errorf("node failed with output %v", output.Data())
}

func (w *workflowWorker) executeParallelNodes(ctx Context, outputEdges map[string]graph.Edge, rawInputData map[string]any) (workflow.NodeOutputData, error) {
	aggregatedOutput := store.New()

	w.currentNode = make([]graph.Node, 0, len(outputEdges))
	status := workflow.NodeOutputStatusSuccess
	asyncCount := 0
	asyncQueue := make(chan struct {
		EdgeID string
		Output workflow.NodeOutput
	})

	for _, edge := range outputEdges {
		node := edge.To()
		input, err := w.createNodeInput(node, rawInputData)
		if err != nil {
			audit.Error().Workflow(w.id).Err(err).Msg("Failed to create node input")
			status = workflow.NodeOutputStatusError
			break
		}

		w.currentNode = append(w.currentNode, node)

		nodeRef := node.NodeRef()
		result, err := nodeRef.Execute(input)
		if err != nil {
			audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute node")
			status = workflow.NodeOutputStatusError
			break
		}

		audit.Info().Workflow(w.id).NodeInputOutput(node.ID(), input, result.Map()).Msg("node executed")

		async, isAsync := result.Async()
		if isAsync {
			asyncCount++
			go func() {
				done := <-async
				asyncQueue <- struct {
					EdgeID string
					Output workflow.NodeOutput
				}{EdgeID: edge.ID(), Output: done}
			}()
			continue
		}
		output := result.Output()
		if output.Status() != workflow.NodeOutputStatusSuccess {
			status = output.Status()
			break
		}
		aggregatedOutput.Set(fmt.Sprintf("edges.%s", edge.ID()), output.Data())
	}
	//goland:noinspection ALL
	if asyncCount == 0 {
		if status == workflow.NodeOutputStatusSuccess {
			return aggregatedOutput.Raw(), nil
		}
		return nil, fmt.Errorf("node failed with output %v", aggregatedOutput.Raw())
	}
	go func() {
		for asyncCount > 0 {
			done := <-asyncQueue
			if done.Output.Status() == workflow.NodeOutputStatusSuccess {
				aggregatedOutput.Set(fmt.Sprintf("edges.%s", done.EdgeID), done.Output.Data())
			}
			asyncCount--
		}
		w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, aggregatedOutput.Raw()))
	}()

	return nil, nil
}

func (w *workflowWorker) createNodeInput(node graph.Node, rawInputData map[string]any) (*workflow.NodeInput, error) {
	audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("RawInputData: %v", rawInputData)

	inputStore, err := store.Init(rawInputData)
	if err != nil {
		audit.Error().Workflow(w.id).Err(err).Msg("Failed to init input store")
		return nil, err
	}

	audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("InputStore: %v", inputStore)

	nodeInput := workflow.NewNodeInput()
	nodeConfig := node.Config()
	inputSchema := node.NodeRef().Metadata().Input()

	for i, mapping := range nodeConfig.InputMapping() {
		audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeConfig.%d.Mapping.Source: %v", i, mapping.Source)
		audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeConfig.%d.Mapping.Origin: %v", i, mapping.Origin)
		audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeConfig.%d.Mapping.Mapping: %v", i, mapping.Mapping)

		paramSchema, exists := inputSchema.Parameters[mapping.Mapping]
		audit.Debug().Workflow(w.id).Node(node.ID()).
			Msgf("NodeMetadata.Schema.Exists: %v; CustomParameters: %v", exists, inputSchema.CustomParameters)
		audit.Debug().Workflow(w.id).Node(node.ID()).
			Msgf("NodeMetadata.Schema.ParamSchema.Name: %v", paramSchema.Name)
		audit.Debug().Workflow(w.id).Node(node.ID()).
			Msgf("NodeMetadata.Schema.ParamSchema.Type: %v", paramSchema.Type)
		audit.Debug().Workflow(w.id).Node(node.ID()).
			Msgf("NodeMetadata.Schema.ParamSchema.Required: %v", paramSchema.Required)

		isCustomParameter := inputSchema.CustomParameters && !exists
		if mapping.Source != graph.InputSourceSchema && !isCustomParameter && !exists {
			audit.Error().Workflow(w.id).Node(node.ID()).
				Str("source", mapping.Source).Any("origin", mapping.Origin).Msg("Input mapping for source.origin not found")
			return nil, fmt.Errorf("input mapping for source.origin %s.%s not found", mapping.Source, mapping.Origin)
		}

		var rawValue any
		if mapping.Source == graph.InputSourceSchema {
			rawValue = mapping.Origin
		} else {
			inputKey := mapping.Origin.(string)
			if len(node.InputEdges()) > 1 {
				inputKey = fmt.Sprintf("%s.%s", mapping.Source, mapping.Origin)
			}
			audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("inputKey: %v", inputKey)
			rawValue = inputStore.Get(inputKey)
			if rawValue == nil {
				audit.Error().Workflow(w.id).Node(node.ID()).
					Str("inputKey", mapping.Source).Msg("Input value for source not found")
				return nil, fmt.Errorf("input value for inputKey %s not found", inputKey)
			}
		}

		audit.Debug().Workflow(w.id).Node(node.ID()).Any("rawValue", rawValue).Msg("InputStore.rawValue")

		isArray := strings.HasPrefix(paramSchema.Type, "[]")
		var paramValue any
		if mapping.Source == graph.InputSourceSchema || isCustomParameter {
			paramValue = rawValue
		} else {
			paramValue, err = typeschema.ParseValue(paramSchema.Type, rawValue)
			if err != nil {
				audit.Error().Workflow(w.id).Node(node.ID()).Err(err).Msg("Failed to parse input value")
				return nil, err
			}
		}

		audit.Debug().Workflow(w.id).Node(node.ID()).Any("paramValue", paramValue).Msg("NodeMetadata.Schema.ParamValueParsed")

		// TODO: Add validation based on the paramSchema before set

		// TODO: Improve set handling
		if isArray {
			currentArray := nodeInput.Get(mapping.Mapping)
			if currentArray != nil {
				nodeInput.Set(mapping.Mapping, append(currentArray.([]any), paramValue.([]any)...))
			} else {
				nodeInput.Set(mapping.Mapping, paramValue.([]any))
			}
		} else {
			nodeInput.Set(mapping.Mapping, paramValue)
		}

	}

	audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeInput: %v", nodeInput.Raw())

	return nodeInput, nil
}
