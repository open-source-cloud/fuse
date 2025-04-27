// Package workflow for internal workflow
package workflow

import (
	"github.com/open-source-cloud/fuse/internal/audit"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"github.com/vladopajic/go-actor/actor"
)

// Engine describes the engine interface
type Engine interface {
	actor.Actor
	AddSchema(schema Schema)
	SendMessage(msg EngineMessage)
}

type engine struct {
	baseActor            actor.Actor
	externalMessagesChan chan EngineMessage
	mailbox              actor.Mailbox[any]
	schemas              map[string]Schema
	workflows            map[string]Workflow
}

// NewEngine creates the engine actor
func NewEngine() Engine {
	worker := &engine{
		externalMessagesChan: make(chan EngineMessage),
		mailbox:              actor.NewMailbox[any](),
		schemas:              make(map[string]Schema),
		workflows:            make(map[string]Workflow),
	}
	worker.baseActor = actor.New(worker)

	return worker
}

func (e *engine) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		for _, workflowActor := range e.workflows {
			workflowActor.Stop()
		}
		audit.Info().Msg("Stopping engine")
		return actor.WorkerEnd

	case msg := <-e.externalMessagesChan:
		audit.Info().EngineMessage(msg.Type(), msg.Data()).Msg("received external engineMessage")
		e.handleMessage(ctx, msg)
		return actor.WorkerContinue

	case msg := <-e.mailbox.ReceiveC():
		audit.Info().Any("msg", msg).Msg("received engineMessage")
		return actor.WorkerContinue
	}
}

func (e *engine) Start() {
	e.baseActor.Start()
}

func (e *engine) Stop() {
	e.baseActor.Stop()
}

func (e *engine) AddSchema(schema Schema) {
	e.schemas[schema.ID()] = schema
}

func (e *engine) SendMessage(msg EngineMessage) {
	e.externalMessagesChan <- msg
}

func (e *engine) handleMessage(ctx actor.Context, msg EngineMessage) {
	switch msg.Type() {
	case EngineMessageStartWorkflow:
		schemaID, ok := msg.Data().(string)
		if !ok {
			audit.Error().Msg("Invalid engineMessage data")
			return
		}
		workflowSchema, ok := e.schemas[schemaID]
		if !ok {
			audit.Error().Schema(schemaID).Msg("Schema not found")
			return
		}
		newWorkflowUUID := uuid.V7()
		audit.Info().Workflow(newWorkflowUUID).Schema(schemaID).Msg("Start new workflow")
		workflowActor := NewWorkflow(newWorkflowUUID, workflowSchema)
		workflowActor.Start()
		e.workflows[newWorkflowUUID] = workflowActor
		workflowActor.SendMessage(
			ctx,
			NewMessage(MessageStartWorkflow, make(map[string]any)),
		)

	default:
		audit.Warn().Any("msgType", msg.Type()).Msg("Unhandled engine engineMessage type")
	}
}
