package cli

import (
	"github.com/open-source-cloud/fuse/internal/database"
	"github.com/open-source-cloud/fuse/internal/server"
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
	config, err := server.NewConfig()
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	db, err := database.NewClient(config.Database.Host, config.Database.Port, config.Database.User, config.Database.Pass, config.Database.TLS)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	sv := server.NewServer(config, db)

	return sv.Start(port)
}
