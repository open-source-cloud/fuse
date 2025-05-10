package server

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/server/handlers"
	"github.com/rs/zerolog/log"
	"time"
)

func New(config *config.Config, graphRepo repos.GraphRepo, messageChan chan<- any) *fiber.App {
	app := fiber.New(fiber.Config{
		Immutable:     true,
		StrictRouting: true,
	})
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency", time.Since(start)).
			Msg("request handled")
		return err
	})

	app.Put("/api/workflow/schema", handlers.NewUpsertWorkflowSchemaHandler(graphRepo).Handle)
	app.Post("/api/workflow", handlers.NewTriggerWorkflowHandler(messageChan).Handle)

	go func() {
		err := app.Listen(fmt.Sprintf(":%s", config.Server.Port), fiber.ListenConfig{
			DisableStartupMessage: true,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to start server")
		}
	}()

	log.Info().
		Str("version", fiber.Version).
		Str("address", fmt.Sprintf("localhost:%s", config.Server.Port)).
		Msg("Fiber server starting")
	return app
}
