// Package app supervisor actor
package app

import (
	"github.com/open-source-cloud/fuse/internal/actormodel"
	"github.com/open-source-cloud/fuse/internal/audit"
	"github.com/open-source-cloud/fuse/internal/config"
	"github.com/open-source-cloud/fuse/internal/database"
	"github.com/open-source-cloud/fuse/internal/providers"
	"github.com/open-source-cloud/fuse/internal/server"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/vladopajic/go-actor/actor"
)

// Supervisor app supervisor interface
type Supervisor interface {
	actormodel.SupervisorMessenger
	actor.Actor

	AddSchema(schema workflow.Schema)
}

type supervisor struct {
	baseActor        actor.Actor
	started          bool
	mailbox          actor.Mailbox[actormodel.Message]
	cfg              *config.Config
	db               *database.ArangoClient
	engine           workflow.Engine
	server           server.Server
	providerRegistry *providers.Registry
}

// NewSupervisor creates a new app supervisor
func NewSupervisor(config *config.Config) Supervisor {
	app := &supervisor{
		cfg:     config,
		mailbox: actor.NewMailbox[actormodel.Message](),
	}
	app.baseActor = actor.Combine(app.mailbox, actor.New(app)).Build()

	return app
}

func (a *supervisor) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		if a.engine != nil {
			a.engine.Stop()
		}
		if a.server != nil {
			a.server.Stop()
		}
		a.engine = nil
		a.server = nil
		a.providerRegistry = nil
		a.db = nil
		audit.Info().Msg("Stopping App")
		return actor.WorkerEnd

	case msg := <-a.mailbox.ReceiveC():
		audit.Info().ActorMessage(msg).Msg("received appMessage")
		switch msg.Type() {
		case "temp":
		default:
			audit.Warn().ActorMessage(msg).Msg("Unhandled a message")
		}
		return actor.WorkerContinue
	}
}

func (a *supervisor) Start() {
	if !a.started {
		a.createDatabase()
		a.createProviderRegistry()

		if a.cfg.Server.Run {
			a.createServer()
		}
		a.createEngine()

		a.baseActor.Start()
		if a.engine != nil {
			a.engine.Start()
		}
		if a.server != nil {
			a.server.Start()
		}

		a.started = true
	}
}

func (a *supervisor) Stop() {
	a.baseActor.Stop()
}

func (a *supervisor) AddSchema(schema workflow.Schema) {
	a.engine.AddSchema(schema)
}

func (a *supervisor) createDatabase() {
	var err error
	dbCfg := a.cfg.Database
	a.db, err = database.NewClient(dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Pass, dbCfg.TLS)
	if err != nil {
		audit.Error().Err(err).Msg("Failed to create database client")
	}

	if err = a.db.Ping(); err != nil {
		audit.Error().Err(err).Msg("Failed to ping database")
	}
}

func (a *supervisor) createServer() {
	a.server = server.NewServer(a.cfg.Server, a.db)
}

func (a *supervisor) createEngine() {
	a.engine = workflow.NewEngine(a)
}

func (a *supervisor) createProviderRegistry() {
	a.providerRegistry = providers.NewRegistry()
}

func (a *supervisor) SendMessage(ctx actor.Context, msg actormodel.Message) {
	a.SendMessageTo(actormodel.AppSupervisor, ctx, msg)
}

func (a *supervisor) SendMessageTo(receiver actormodel.MessageReceiver, ctx actor.Context, msg actormodel.Message) {
	switch receiver {
	case actormodel.AppSupervisor:
		err := a.mailbox.Send(ctx, msg)
		if err != nil {
			audit.Error().Err(err).Msg("Failed to send message")
		}
	case actormodel.HTTPServer:
		a.server.SendMessage(ctx, msg)
	case actormodel.WorkflowEngine:
		a.engine.SendMessage(ctx, msg)
	}
}
