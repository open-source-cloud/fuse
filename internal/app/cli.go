package app

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fuse",
	Short: "FUSE Workflow Engine application server",
	Long: `FUSE Workflow Engine application server

	Entrypoint for FUSE Workflow Engine Server commands`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		//
	},
}

func init() {
	//
}

func RunServerCli() {
	runApplication()
}
