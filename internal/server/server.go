package server

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/actormodel"
	"github.com/open-source-cloud/fuse/internal/audit"
	"github.com/open-source-cloud/fuse/internal/config"
	"github.com/open-source-cloud/fuse/internal/database"
	"github.com/open-source-cloud/fuse/internal/server/handlers"
	"github.com/open-source-cloud/fuse/internal/server/servermsg"
	"github.com/vladopajic/go-actor/actor"
)

// Server represents the FUSE Workflow Engine application server.
type Server interface {
	actor.Actor
	actormodel.Messenger
}

type server struct {
	baseActor actor.Actor
	mailbox   actor.Mailbox[actormodel.Message]
	fiberApp  *fiber.App
	cfg       config.ServerConfig
	db        *database.ArangoClient
}

// NewServer creates and returns a new instance of Server
func NewServer(cfg config.ServerConfig, db *database.ArangoClient) Server {
	app := fiber.New(fiber.Config{
		Immutable:     true,
		StrictRouting: true,
	})

	sv := &server{
		fiberApp: app,
		mailbox:  actor.NewMailbox[actormodel.Message](),
		cfg:      cfg,
		db:       db,
	}

	sv.registerHandlers()

	sv.baseActor = actor.New(sv)
	return sv
}

func (s *server) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		_ = s.fiberApp.Shutdown()
		return actor.WorkerEnd

	case msg := <-s.mailbox.ReceiveC():
		audit.Info().ActorMessage(msg).Msg("received serverMessage")
		switch msg.Type() {
		case servermsg.StartListening:
			err := s.listen()
			if err != nil {
				audit.Error().ActorMessage(msg).Err(err).Msg("Failed to start listening")
				return actor.WorkerEnd
			}
			audit.Info().Msg("Listening on port " + s.cfg.Port)
			return actor.WorkerContinue

		default:
			audit.Warn().ActorMessage(msg).Msg("Unhandled server message")
			return actor.WorkerContinue
		}
	}
}

func (s *server) Start() {
	s.baseActor.Start()
}

func (s *server) Stop() {
	s.baseActor.Stop()
}

func (s *server) SendMessage(ctx actor.Context, msg actormodel.Message) {
	err := s.mailbox.Send(ctx, msg)
	if err != nil {
		audit.Error().ActorMessage(msg).Err(err).Msg("Failed to send message")
		return
	}
	s.mailbox.Start()
}

func (s *server) listen() error {
	return s.fiberApp.Listen(fmt.Sprintf(":%s", s.cfg.Port))
}

// registerHandlers registers the handlers for the server.
func (s *server) registerHandlers() {
	healthHandler := handlers.NewHealthCheckHandler(s.db)
	schemaHandler := handlers.NewSchemaHandler()

	s.fiberApp.Get("/health-check", healthHandler.Handle)

	s.fiberApp.Get("/v1/schemas", schemaHandler.Handle)
}
