package workflow

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"github.com/vladopajic/go-actor/actor"
	"reflect"
	"strings"
)

func typesMap() map[string]reflect.Type {
	return map[string]reflect.Type{
		"int": reflect.TypeOf(0),
	}
}

type State string

const (
	StateStopped  State = "stopped"
	StateRunning  State = "running"
	StateFinished State = "finished"
	StateError    State = "error"
)

type Context actor.Context

type Workflow interface {
	actor.Actor
	SendMessage(ctx Context, msg Message)
}

type workflowWorker struct {
	baseActor   actor.Actor
	mailbox     actor.Mailbox[Message]
	id          string
	schema      Schema
	data        map[string]interface{}
	state       State
	currentNode graph.Node
}

func NewWorkflow(id string, schema Schema) Workflow {
	worker := &workflowWorker{
		mailbox: actor.NewMailbox[Message](),
		id:      id,
		schema:  schema,
		data:    make(map[string]interface{}),
		state:   StateStopped,
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
		log.Info().Msgf("Workflow %s : Stopping", w.id)
		return actor.WorkerEnd

	case msg := <-w.mailbox.ReceiveC():
		log.Info().Msgf("Workflow %s : Received message %s", w.id, msg)
		w.handleMessage(ctx, msg)
		return actor.WorkerContinue
	}
}

func (w *workflowWorker) SendMessage(ctx Context, msg Message) {
	err := w.mailbox.Send(ctx, msg)
	if err != nil {
		log.Error().Err(err).Msgf("Workflow %s : Failed to send message to Workflow", w.id)
	}
	w.mailbox.Start()
}

func (w *workflowWorker) handleMessage(ctx Context, msg Message) {
	switch msg.Type() {
	case MessageStartWorkflow:
		rootNode := w.schema.RootNode()
		w.executeNode(ctx, rootNode, msg.Data())

	case MessageContinueWorkflow:
		outputEdges := w.currentNode.OutputEdges()
		if len(outputEdges) == 0 {
			log.Info().Msgf("Workflow %s : No output edges for node %s", w.id, w.currentNode.ID())
			w.state = StateFinished
			log.Info().Msgf("Workflow %s : Workflow finished with state: %s", w.id, w.state)
			ctx.Done()
			return
		}

		if len(outputEdges) == 0 {
			w.state = StateFinished
			log.Info().Msgf("Workflow %s : Workflow finished with state: %s", w.id, w.state)
			ctx.Done()
		} else if len(outputEdges) == 1 {
			edge, exists := outputEdges["default"]
			if !exists {
				log.Error().Msgf("Workflow %s : No default output edge for node %s", w.id, w.currentNode.ID())
				w.state = StateError
				log.Info().Msgf("Workflow %s : Workflow finished with state: %s", w.id, w.state)
				ctx.Done()
			}
			w.executeNode(ctx, edge.To(), msg.Data())
		} else {
			//w.executeParallelNodes(ctx, outputEdges, msg.Data())
		}

	default:
		// Handle unknown message types
		log.Warn().Msgf("Workflow %s : Received unknown message type %s", w.id, msg.Type())
	}
}

func (w *workflowWorker) executeNode(ctx Context, node graph.Node, rawInputData interface{}) {
	input, _ := w.processRawInput(node, rawInputData)
	w.currentNode = node
	result, _ := node.NodeRef().Execute(input)
	log.Info().Msgf("Workflow %s : Node %s finished with result: %s", w.id, node.ID(), result)

	var output workflow.NodeOutput
	_, isAsync := result.Async()
	if !isAsync {
		output = result.Output()
	}
	//goland:noinspection ALL
	if output.Status() == workflow.NodeOutputStatusSuccess {
		w.SendMessage(ctx, NewMessage(MessageContinueWorkflow, output.Data()))
	} else {
		w.state = StateError
		return
	}
}

func (w *workflowWorker) processRawInput(node graph.Node, rawInputData interface{}) (map[string]interface{}, error) {
	inputData := rawInputData.(map[string]interface{})
	input := make(map[string]interface{})
	nodeConfig := node.Config()
	inputSchema := node.NodeRef().Metadata().Input()

	for _, mapping := range nodeConfig.InputMapping() {
		paramSchema, exists := inputSchema.Parameters[mapping.Mapping]
		if !exists {
			log.Error().Msgf("Workflow %s : Input mapping for parameter %s not found", w.id, mapping.ParamName)
			return nil, fmt.Errorf("input mapping for parameter %s not found", mapping.ParamName)
		}

		paramValue, exists := inputData[mapping.ParamName]
		if !exists {
			log.Error().Msgf("Workflow %s : Input mapping for parameter %s not found", w.id, mapping.ParamName)
			return nil, fmt.Errorf("input mapping for parameter %s not found", mapping.ParamName)
		}

		if strings.HasPrefix(paramSchema.Type, "[]") {
			sliceType := reflect.SliceOf(typesMap()[paramSchema.Type[2:]])
			slice := reflect.MakeSlice(sliceType, 0, 0)
			appendedSlice := reflect.Append(slice, reflect.ValueOf(paramValue))
			value := appendedSlice.Interface()
			input[mapping.Mapping] = value
		} else {

		}
	}

	return input, nil
}
