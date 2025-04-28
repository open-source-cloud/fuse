package cli

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/handlers"
	"github.com/spf13/cobra"
)

var port string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the FUSE Workflow Engine application server",
	Args:  cobra.NoArgs,
	RunE:  serverRunner,
}

func init() {
	serverCmd.Flags().StringVarP(&port, "port", "p", "4567", "Port to listen on for HTTP requests")
}

func serverRunner(_ *cobra.Command, _ []string) error {
	healthHandler := handlers.NewHealthCheckHandler()

	app := fiber.New(fiber.Config{
		Immutable:     true,
		StrictRouting: true,
	})

	app.Get("/health-check", healthHandler.Handle)

	return app.Listen(fmt.Sprintf(":%s", port))
}
