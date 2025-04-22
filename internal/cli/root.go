// Package cli provides the root command for the FUSE Workflow Engine CLI
package cli

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Root command for FUSE Workflow Engine CLI
var rootCmd = &cobra.Command{
	Use:           "fuse",
	Short:         "FUSE Workflow Engine application server",
	SilenceErrors: false,
	SilenceUsage:  false,
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Help()
		if err != nil {
			log.Error().Msgf("Failed to print help: %v", err)
			os.Exit(1)
		}
	},
}

// Execute the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Initialize the root command
func init() {
	rootCmd.AddCommand(workflowCmd)
}
