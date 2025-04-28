package server

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/database"
	"github.com/open-source-cloud/fuse/internal/server/handlers"
)

type Server struct {
	app *fiber.App
	cfg *Config
	db  *database.ArangoClient
}

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

func (s *Server) registerHandlers() {
	healthHandler := handlers.NewHealthCheckHandler(s.db)
	s.app.Get("/health-check", healthHandler.Handle)
}

func (s *Server) Start(port string) error {
	return s.app.Listen(fmt.Sprintf(":%s", port))
}
