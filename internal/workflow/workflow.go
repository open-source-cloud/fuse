package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"github.com/vladopajic/go-actor/actor"
)

type State string

const (
	StateStopped  State = "stopped"
	StateRunning  State = "running"
	StateFinished State = "finished"
	StateError    State = "error"
)

type workflowWorker struct {
	baseActor actor.Actor
	mailbox   actor.Mailbox[workflow.Message]
	id        string
	schema    Schema
	data      map[string]interface{}
	state     State
}

func NewWorkflow(id string, schema Schema) workflow.Workflow {
	worker := &workflowWorker{
		mailbox: actor.NewMailbox[workflow.Message](),
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

func (w *workflowWorker) SendMessage(ctx workflow.Context, msg workflow.Message) {
	err := w.mailbox.Send(ctx, msg)
	if err != nil {
		log.Error().Err(err).Msgf("Workflow %s : Failed to send message to Workflow", w.id)
	}
	w.mailbox.Start()
}

func (w *workflowWorker) handleMessage(ctx workflow.Context, msg workflow.Message) {
	switch msg.Type() {
	case workflow.MessageStartWorkflow:
		input := msg.Data().(map[string]interface{})
		rootNode := w.schema.RootNode()
		result, _ := rootNode.NodeRef().Execute(input)
		log.Info().Msgf("Workflow %s : Node %s finished with result: %s", w.id, rootNode.ID(), result)

		var output workflow.NodeOutput
		_, isAsync := result.Async()
		if !isAsync {
			output = result.Output()
		}
		//goland:noinspection ALL
		if output.Status() == workflow.NodeOutputStatusSuccess {
			w.SendMessage(ctx, workflow.NewMessage(workflow.MessageContinueWorkflow, output.Data()))
		} else {
			w.state = StateError
		}
		log.Info().Msgf("Workflow %s : Workflow finished with state: %s", w.id, w.state)
		ctx.Done()

	default:
		// Handle unknown message types
		log.Warn().Msgf("Workflow %s : Received unknown message type %s", w.id, msg.Type())
	}
}
