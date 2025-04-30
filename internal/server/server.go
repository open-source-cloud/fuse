package server

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/database"
	"github.com/open-source-cloud/fuse/internal/server/handlers"
)

// Server represents the FUSE Workflow Engine application server.
type Server struct {
	app *fiber.App
	cfg *Config
	db  *database.ArangoClient
}

// NewServer creates and returns a new instance of Server
func NewServer(cfg *Config, db *database.ArangoClient) *Server {
	app := fiber.New(fiber.Config{
		Immutable:     true,
		StrictRouting: true,
	})

	sv := &Server{
		app: app,
		cfg: cfg,
		db:  db,
	}

	sv.registerHandlers()

	return sv
}

// Start starts the http server and blocks until the server is stopped.
func (s *Server) Start(port string) error {
	return s.app.Listen(fmt.Sprintf(":%s", port))
}

// registerHandlers registers the handlers for the server.
func (s *Server) registerHandlers() {
	healthHandler := handlers.NewHealthCheckHandler(s.db)
	s.app.Get("/health-check", healthHandler.Handle)
}
