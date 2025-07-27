package cli

import (
	"github.com/open-source-cloud/fuse/internal/app/di"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func newServerCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the FUSE Workflow Engine application server",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			di.Run(fx.Options(
				di.AllModules,
			))
		},
	}
	return serverCmd
}
