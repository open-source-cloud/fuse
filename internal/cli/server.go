package cli

import (
	"context"
	"github.com/open-source-cloud/fuse/internal/actormodel"
	"github.com/open-source-cloud/fuse/internal/app"
	"github.com/open-source-cloud/fuse/internal/config"
	"github.com/open-source-cloud/fuse/internal/server/servermsg"
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
	serverCmd.Flags().StringVarP(&port, "port", "p", "", "Port to listen on for HTTP requests")
}

func serverRunner(_ *cobra.Command, _ []string) error {
	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}

	if err = cfg.Validate(); err != nil {
		return err
	}

	cfg.Server.Run = true
	cfg.Server.Port = port

	appSupervisor := app.NewSupervisor(cfg)
	appSupervisor.Start()
	appSupervisor.SendMessageTo(
		actormodel.HTTPServer,
		context.Background(),
		actormodel.NewMessage(servermsg.StartListening, nil),
	)

	quitOnCtrlC()
	return nil
}
