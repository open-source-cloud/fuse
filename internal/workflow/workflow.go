package workflow

import "github.com/open-source-cloud/fuse/pkg/uuid"

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

const (
	ActionRunFunction ActionType = "function:run"
	ActionRunFunctions
)

func New(id ID, graph *Graph) *Workflow {
	return &Workflow{
		ID:    id,
		graph: graph,
		state: RunningState{
			currentState: StateUntriggered,
		},
	}
}

type (
	Workflow struct {
		ID    ID
		graph *Graph
		state RunningState
	}

	RunningState struct {
		currentState State
	}

	Action interface {
		Type() ActionType
	}

	RunFunctionAction struct {
		FunctionID string
		Args       map[string]any
	}
)

func (w *Workflow) Trigger() Action {
	triggerNode := w.graph.Root()
	return &RunFunctionAction{
		FunctionID: triggerNode.FunctionID(),
		Args:   map[string]any{},
	}
}

func (w *Workflow) State() State {
	return w.state.currentState
}

func (w *Workflow) SetState(state State) {
	w.state.currentState = state
}

func (a *RunFunctionAction) Type() ActionType {
	return ActionRunFunction
}

//
//// State type for workflow states
//type State string
//
//const (
//	// StateUntriggered workflow is stopped
//	StateUntriggered State = "stopped"
//	// StateRunning workflow is running
//	StateRunning State = "running"
//	// StateFinished workflow has finished successfully
//	StateFinished State = "finished"
//	// StateError workflow has finished with errors
//	StateError State = "error"
//)
//
//// Context the context type used by workflow actors
//type Context actor.Context
//
//// Workflow describes the interface for a Workflow actor
//type Workflow interface {
//	actor.Actor
//	SendMessage(ctx Context, msg actors.message)
//}
//
//type workflowWorker struct {
//	baseActor   actor.Actor
//	mailbox     actor.Mailbox[actors.message]
//	id          string
//	schema      GraphSchema
//	data        map[string]any
//	state       State
//	currentNode []graph.Node
//}
//
//// NewWorkflow creates a new workflow actor worker
//func NewWorkflow(id string, schema GraphSchema) Workflow {
//	worker := &workflowWorker{
//		mailbox:     actor.NewMailbox[actors.message](),
//		id:          id,
//		schema:      schema,
//		data:        make(map[string]any),
//		state:       StateUntriggered,
//		currentNode: []graph.Node{},
//	}
//	worker.baseActor = actor.New(worker)
//	return worker
//}
//
//func (w *workflowWorker) Start() {
//	w.state = StateRunning
//	w.baseActor.Start()
//}
//
//func (w *workflowWorker) Stop() {
//	w.baseActor.Stop()
//	audit.Info().WorkflowState(w.id, w.state).Msg("Stop() called")
//	w.state = StateUntriggered
//}
//
//func (w *workflowWorker) DoWork(ctx actor.Context) actor.WorkerStatus {
//	select {
//	case <-ctx.Done():
//		audit.Info().WorkflowState(w.id, w.state).Msg("Stopped")
//		return actor.WorkerEnd
//
//	case msg := <-w.mailbox.ReceiveC():
//		audit.Info().WorkflowMessage(w.id, msg).Msg("message received")
//		if err := w.handleMessage(ctx, msg); err != nil {
//			audit.Info().WorkflowState(w.id, w.state).Msg("Stopped")
//			return actor.WorkerEnd
//		}
//		return actor.WorkerContinue
//	}
//}
//
//func (w *workflowWorker) SendMessage(ctx Context, msg actors.message) {
//	err := w.mailbox.Send(ctx, msg)
//	if err != nil {
//		audit.Error().
//			Workflow(w.id).
//			Err(err).
//			Msg("Failed to send message to Workflow")
//	}
//	w.mailbox.Start()
//}
//
//func (w *workflowWorker) handleMessage(ctx Context, msg actors.message) error {
//	switch msg.Type() {
//	case workflowmsg.Start:
//		rootNode := w.schema.RootNode()
//		output, err := w.executeNode(ctx, rootNode, msg.Args())
//		if err != nil {
//			audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute root node")
//			w.state = StateError
//			return err
//		}
//		w.SendMessage(ctx, actors.NewMessage(workflowmsg.Continue, actors.MessageData(output)))
//	case workflowmsg.Continue:
//		currentNodeCount := len(w.currentNode)
//		if currentNodeCount == 0 {
//			err := fmt.Errorf("no current node")
//			audit.Error().Workflow(w.id).Err(err).Msg("No current node")
//			return err
//		}
//
//		outputEdges := map[string]graph.Edge{}
//		for _, node := range w.currentNode {
//			outputMetadata := node.NodeRef().Metadata().Output()
//			nodeOutputEdges := node.OutputEdges()
//			for k, edge := range nodeOutputEdges {
//				if edge.IsConditional() && outputMetadata.ConditionalOutput {
//					inputData := msg.Args()
//					edgeMetadata := outputMetadata.Edges[edge.Condition().Name]
//					if edge.Condition().Value == inputData[edgeMetadata.ConditionalEdge.Condition] {
//						outputEdges[k] = edge
//					} else {
//						audit.Debug().Workflow(w.id).Msg("Conditional edge not met")
//					}
//				} else {
//					outputEdges[k] = edge
//				}
//			}
//		}
//		switch len(outputEdges) {
//		case 0:
//			audit.Info().Workflow(w.id).Nodes(w.currentNode).Msg("No output edges")
//			w.state = StateFinished
//			return fmt.Errorf("no output edges")
//		case 1:
//			var edge graph.Edge
//			for _, edgeRef := range outputEdges {
//				edge = edgeRef
//				break
//			}
//			//goland:noinspection ALL
//			output, err := w.executeNode(ctx, edge.To(), msg.Args())
//			if err != nil {
//				audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute root node")
//				w.state = StateError
//				return err
//			}
//			if output != nil {
//				w.SendMessage(ctx, actors.NewMessage(workflowmsg.Continue, actors.MessageData(output)))
//			}
//		default:
//			output, err := w.executeParallelNodes(ctx, outputEdges, msg.Args())
//			if err != nil {
//				audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute parallel nodes")
//				w.state = StateError
//				return err
//			}
//			w.SendMessage(ctx, actors.NewMessage(workflowmsg.Continue, actors.MessageData(output)))
//		}
//
//	default:
//		// Handle unknown message types
//		audit.Warn().WorkflowMessage(w.id, msg).Msg("Unknown message type")
//		return fmt.Errorf("unknown message type %s", msg.Type())
//	}
//
//	return nil
//}
//
//func (w *workflowWorker) executeNode(ctx Context, node graph.Node, rawInputData map[string]any) (workflow.FunctionOutputData, error) {
//	input, err := w.createNodeInput(node, rawInputData)
//	if err != nil {
//		audit.Error().Workflow(w.id).Err(err).Msg("Failed to create node input")
//		w.state = StateError
//		return nil, err
//	}
//
//	nodeRef := node.NodeRef()
//	w.currentNode = []graph.Node{node}
//	result, err := nodeRef.Execute(input)
//	if err != nil {
//		audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute node")
//		w.state = StateError
//		return nil, err
//	}
//
//	audit.Info().Workflow(w.id).NodeInputOutput(node.ID(), input.Raw(), result.Map()).Msg("node executed")
//
//	var output workflow.FunctionOutput
//	async, isAsync := result.Async()
//
//	if isAsync {
//		go func() {
//			done := <-async
//			if done.Status() == workflow.FunctionSuccess {
//				w.SendMessage(ctx, actors.NewMessage(workflowmsg.Continue, actors.MessageData(done.Args())))
//			}
//		}()
//		return nil, nil
//	}
//
//	output = result.Output()
//	if output.Status() == workflow.FunctionSuccess {
//		return output.Args(), nil
//	}
//
//	w.state = StateError
//	// TODO: Improve error handling
//	return nil, fmt.Errorf("node failed with output %v", output.Args())
//}
//
//func (w *workflowWorker) executeParallelNodes(ctx Context, outputEdges map[string]graph.Edge, rawInputData map[string]any) (workflow.FunctionOutputData, error) {
//	aggregatedOutput := store.New()
//
//	w.currentNode = make([]graph.Node, 0, len(outputEdges))
//	status := workflow.FunctionSuccess
//	asyncCount := 0
//	asyncQueue := make(chan struct {
//		EdgeID string
//		Output workflow.FunctionOutput
//	})
//
//	for _, edge := range outputEdges {
//		node := edge.To()
//		input, err := w.createNodeInput(node, rawInputData)
//		if err != nil {
//			audit.Error().Workflow(w.id).Err(err).Msg("Failed to create node input")
//			status = workflow.FunctionError
//			break
//		}
//
//		w.currentNode = append(w.currentNode, node)
//
//		nodeRef := node.NodeRef()
//		result, err := nodeRef.Execute(input)
//		if err != nil {
//			audit.Error().Workflow(w.id).Err(err).Msg("Failed to execute node")
//			status = workflow.FunctionError
//			break
//		}
//
//		audit.Info().Workflow(w.id).NodeInputOutput(node.ID(), input, result.Map()).Msg("node executed")
//
//		async, isAsync := result.Async()
//		if isAsync {
//			asyncCount++
//			go func() {
//				done := <-async
//				asyncQueue <- struct {
//					EdgeID string
//					Output workflow.FunctionOutput
//				}{EdgeID: edge.ID(), Output: done}
//			}()
//			continue
//		}
//		output := result.Output()
//		if output.Status() != workflow.FunctionSuccess {
//			status = output.Status()
//			break
//		}
//		aggregatedOutput.Set(fmt.Sprintf("edges.%s", edge.ID()), output.Args())
//	}
//	//goland:noinspection ALL
//	if asyncCount == 0 {
//		if status == workflow.FunctionSuccess {
//			return aggregatedOutput.Raw(), nil
//		}
//		return nil, fmt.Errorf("node failed with output %v", aggregatedOutput.Raw())
//	}
//	go func() {
//		for asyncCount > 0 {
//			done := <-asyncQueue
//			if done.Output.Status() == workflow.FunctionSuccess {
//				aggregatedOutput.Set(fmt.Sprintf("edges.%s", done.EdgeID), done.Output.Args())
//			}
//			asyncCount--
//		}
//		w.SendMessage(ctx, actors.NewMessage(workflowmsg.Continue, actors.MessageData(aggregatedOutput.Raw())))
//	}()
//
//	return nil, nil
//}
//
//func (w *workflowWorker) createNodeInput(node graph.Node, rawInputData map[string]any) (*workflow.FunctionInput, error) {
//	audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("RawInputData: %v", rawInputData)
//
//	inputStore, err := store.Init(rawInputData)
//	if err != nil {
//		audit.Error().Workflow(w.id).Err(err).Msg("Failed to init input store")
//		return nil, err
//	}
//
//	audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("InputStore: %v", inputStore)
//
//	nodeInput := workflow.NewFunctionInput()
//	nodeConfig := node.Config()
//	inputSchema := node.NodeRef().Metadata().Input()
//
//	for i, mapping := range nodeConfig.InputMapping() {
//		audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeConfig.%d.Mapping.Source: %v", i, mapping.Source)
//		audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeConfig.%d.Mapping.Origin: %v", i, mapping.Origin)
//		audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("NodeConfig.%d.Mapping.Mapping: %v", i, mapping.Mapping)
//
//		paramSchema, exists := inputSchema.Parameters[mapping.Mapping]
//		audit.Debug().Workflow(w.id).Node(node.ID()).
//			Msgf("FunctionMetadata.GraphSchema.Exists: %v; CustomParameters: %v", exists, inputSchema.CustomParameters)
//		audit.Debug().Workflow(w.id).Node(node.ID()).
//			Msgf("FunctionMetadata.GraphSchema.ParamSchema.Name: %v", paramSchema.Name)
//		audit.Debug().Workflow(w.id).Node(node.ID()).
//			Msgf("FunctionMetadata.GraphSchema.ParamSchema.Type: %v", paramSchema.Type)
//		audit.Debug().Workflow(w.id).Node(node.ID()).
//			Msgf("FunctionMetadata.GraphSchema.ParamSchema.Required: %v", paramSchema.Required)
//
//		isCustomParameter := inputSchema.CustomParameters && !exists
//		if mapping.Source != graph.InputSourceSchema && !isCustomParameter && !exists {
//			audit.Error().Workflow(w.id).Node(node.ID()).
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
//			audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("inputKey: %v", inputKey)
//			rawValue = inputStore.Get(inputKey)
//			if rawValue == nil {
//				audit.Error().Workflow(w.id).Node(node.ID()).
//					Str("inputKey", mapping.Source).Msg("Input value for source not found")
//				return nil, fmt.Errorf("input value for inputKey %s not found", inputKey)
//			}
//		}
//
//		audit.Debug().Workflow(w.id).Node(node.ID()).Any("rawValue", rawValue).Msg("InputStore.rawValue")
//
//		isArray := strings.HasPrefix(paramSchema.Type, "[]")
//		var paramValue any
//		if mapping.Source == graph.InputSourceSchema || isCustomParameter {
//			paramValue = rawValue
//		} else {
//			paramValue, err = typeschema.ParseValue(paramSchema.Type, rawValue)
//			if err != nil {
//				audit.Error().Workflow(w.id).Node(node.ID()).Err(err).Msg("Failed to parse input value")
//				return nil, err
//			}
//		}
//
//		audit.Debug().Workflow(w.id).Node(node.ID()).Any("paramValue", paramValue).Msg("FunctionMetadata.GraphSchema.ParamValueParsed")
//
//		// TODO: Add validation based on the paramSchema before set
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
//	audit.Debug().Workflow(w.id).Node(node.ID()).Msgf("FunctionInput: %v", nodeInput.Raw())
//
//	return nodeInput, nil
//}
