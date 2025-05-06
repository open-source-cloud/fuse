package cli

import (
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the FUSE Workflow Engine application server",
	Args:  cobra.NoArgs,
	RunE:  serverRunner,
}

func init() {}

func serverRunner(_ *cobra.Command, _ []string) error {
	//cfg, err := config.NewConfig()
	//if err != nil {
	//	return err
	//}
	//
	//if err = cfg.Validate(); err != nil {
	//	return err
	//}
	//
	//cfg.Server.Run = true
	//cfg.Server.Port = port

	//appSupervisor := app.NewSupervisor(cfg)
	//appSupervisor.Start()
	//appSupervisor.SendMessageTo(
	//	actors.HTTPServer,
	//	context.Background(),
	//	actors.NewMessage(servermsg.StartListening, nil),
	//)
	//
	return nil
}
