// Package cli provides the root command for the FUSE Workflow Engine CLI
package cli

import (
	"github.com/rs/zerolog"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var loglevel string

// Root command for FUSE Workflow Engine CLI
var rootCmd = &cobra.Command{
	Use:           "fuse",
	Short:         "FUSE Workflow Engine application server",
	SilenceErrors: false,
	SilenceUsage:  false,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level, err := zerolog.ParseLevel(strings.ToLower(loglevel))
		if err != nil {
			log.Error().Msgf("Invalid log level: %s", loglevel)
			level = zerolog.InfoLevel // Default fallback
		}
		zerolog.SetGlobalLevel(level)
	},
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
	InitLogger()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Initialize the root command
func init() {
	rootCmd.PersistentFlags().StringVarP(
		&loglevel,
		"loglevel",
		"l",
		"info",
		"Log level (debug, info, warn, error)",
	)
	rootCmd.AddCommand(workflowCmd)
}
