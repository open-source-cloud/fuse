// Package workflow for internal workflow
package workflow

import "github.com/rs/zerolog/log"

// Engine Workflow Engine interface
type Engine interface {
}

type engine struct {

	//supervisor actors.SupervisorMessenger
	//schemas    map[string]Schema
	//workflows  map[string]Workflow
	//mailbox    actor.Mailbox[actors.message]
	//baseActor  actor.Actor
}

// NewEngine creates the engine actor
func NewEngine() Engine {
	log.Info().Msg("engine started")
	return &engine{}
}

//func NewEngine(supervisor actors.SupervisorMessenger) Engine {
//	worker := &engine{
//		supervisor: supervisor,
//		mailbox:    actor.NewMailbox[actors.message](),
//		schemas:    make(map[string]Schema),
//		workflows:  make(map[string]Workflow),
//	}
//	worker.baseActor = actor.New(worker)
//
//	return worker
//}

//func (e *engine) DoWork(ctx actor.Context) actor.WorkerStatus {
//	select {
//	case <-ctx.Done():
//		for _, workflowActor := range e.workflows {
//			workflowActor.Stop()
//		}
//		audit.Info().Msg("Stopping engine")
//		return actor.WorkerEnd
//
//	case msg := <-e.mailbox.ReceiveC():
//		audit.Info().ActorMessage(msg).Msg("received engineMessage")
//		e.handleMessage(ctx, msg)
//		return actor.WorkerContinue
//	}
//}
//
//func (e *engine) Start() {
//	e.baseActor.Start()
//}
//
//func (e *engine) Stop() {
//	e.baseActor.Stop()
//}
//
//func (e *engine) AddSchema(schema Schema) {
//	e.schemas[schema.ID()] = schema
//}
//
//func (e *engine) handleMessage(ctx actor.Context, msg actors.message) {
//	switch msg.Type() {
//	case enginemsg.StartWorkflow:
//		schemaID := msg.Data().Str("schema_id")
//		if schemaID == "" {
//			audit.Error().ActorMessage(msg).Msg("Invalid engine message data")
//			return
//		}
//		workflowSchema, ok := e.schemas[schemaID]
//		if !ok {
//			audit.Error().Schema(schemaID).Msg("Schema not found")
//			return
//		}
//		newWorkflowUUID := uuid.V7()
//		audit.Info().Workflow(newWorkflowUUID).Schema(schemaID).Msg("Start new workflow")
//		workflowActor := NewWorkflow(newWorkflowUUID, workflowSchema)
//		workflowActor.Start()
//		e.workflows[newWorkflowUUID] = workflowActor
//		workflowActor.SendMessage(
//			ctx,
//			actors.NewMessage(workflowmsg.Start, make(map[string]any)),
//		)
//
//	default:
//		audit.Warn().ActorMessage(msg).Msg("Unhandled engine message")
//	}
//}
//
//func (e *engine) SendMessage(ctx actor.Context, msg actors.message) {
//	err := e.mailbox.Send(ctx, msg)
//	if err != nil {
//		audit.Error().ActorMessage(msg).Err(err).Msg("Failed to send message")
//	}
//	e.mailbox.Start()
//}
