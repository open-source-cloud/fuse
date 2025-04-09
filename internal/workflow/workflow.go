package workflow

import (
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

type Workflow interface {
	actor.Actor
	SendMessage(ctx actor.Context, msg Message)
}

type workflow struct {
	baseActor actor.Actor
	mailbox   actor.Mailbox[Message]
	id        string
	schema    Schema
	data      map[string]interface{}
	state     State
}

func NewWorkflow(id string, schema Schema) Workflow {
	worker := &workflow{
		mailbox: actor.NewMailbox[Message](),
		id:      id,
		schema:  schema,
		data:    make(map[string]interface{}),
		state:   StateStopped,
	}
	worker.baseActor = actor.New(worker)
	return worker
}

func (w *workflow) Start() {
	w.state = StateRunning
	w.baseActor.Start()
}

func (w *workflow) Stop() {
	w.baseActor.Stop()
	w.state = StateStopped
}

func (w *workflow) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		log.Info().Msgf("Stopping workflow %s", w.id)
		return actor.WorkerEnd

	case msg := <-w.mailbox.ReceiveC():
		log.Info().Msgf("workflow id %s received engineMessage %s", w.id, msg)
		w.handleMessage(ctx, msg)
		return actor.WorkerContinue
	}
}

func (w *workflow) SendMessage(ctx actor.Context, msg Message) {
	err := w.mailbox.Send(ctx, msg)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to send engineMessage to workflow %s", w.id)
	}
	w.mailbox.Start()
}

func (w *workflow) handleMessage(ctx actor.Context, msg Message) {
	switch msg.Type() {
	case MessageStartWorkflow:
		rootNode := w.schema.FindNodeByIndex(0)
		_, _ = rootNode.NodeRef().Execute()

	default:
		// Handle unknown message types
		log.Warn().Msgf("Received unknown message type for workflow %s: %v", w.id, msg.Type)
	}
}
