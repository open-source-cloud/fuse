// Package cli provides the root command for the FUSE Workflow Engine CLI
package cli

import (
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var loglevel string
var observer bool
var port string
var nocolor bool

// Run runs the CLI
func Run() {
	err := newRoot().Execute()
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute root command")
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "fuse",
		Short:         "FUSE Workflow Engine application server",
		SilenceErrors: false,
		SilenceUsage:  false,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			setupGlobalConfig()
		},
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			if err != nil {
				log.Error().Msgf("Failed to print help: %v", err)
				os.Exit(1)
			}
		},
	}

	setupRootFlags(rootCmd)

	rootCmd.AddCommand(newServerCommand())
	rootCmd.AddCommand(newWorkflowCommand())
	rootCmd.AddCommand(newMermaidCommand())

	return rootCmd
}

func setupGlobalConfig() {
	cfg := config.Instance()

	level, err := zerolog.ParseLevel(strings.ToLower(loglevel))
	if err != nil {
		level = zerolog.InfoLevel // Default fallback
	}
	zerolog.SetGlobalLevel(level)
	cfg.Params.LogLevel = loglevel
	cfg.Params.ActorObserver = observer
	cfg.Server.Port = port
	color.NoColor = color.NoColor || nocolor
}

// Initialize the root command
func setupRootFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().StringVarP(
		&loglevel,
		"loglevel",
		"l",
		"info",
		"Log level (debug, info, warn, error)",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&observer,
		"observer",
		"o",
		false,
		"Run the actor observer app",
	)
	rootCmd.PersistentFlags().StringVarP(
		&port,
		"port",
		"p",
		"9090",
		"Port to listen on for HTTP requests",
	)
	rootCmd.PersistentFlags().BoolVar(
		&nocolor,
		"no-color",
		false,
		"Disable color output",
	)
}
