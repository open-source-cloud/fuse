// Package cli provides the root command for the FUSE Workflow Engine CLI
package cli

import (
	"github.com/open-source-cloud/fuse/app/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var loglevel string
var observer bool
var port string
var cfg *config.Config

// Root command for FUSE Workflow Engine CLI
var rootCmd = &cobra.Command{
	Use:           "fuse",
	Short:         "FUSE Workflow Engine application server",
	SilenceErrors: false,
	SilenceUsage:  false,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		level, err := zerolog.ParseLevel(strings.ToLower(loglevel))
		if err != nil {
			level = zerolog.InfoLevel // Default fallback
		}
		zerolog.SetGlobalLevel(level)
		cfg.Params.LogLevel = loglevel
		cfg.Params.ActorObserver = observer
		cfg.Server.Port = port
	},
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Help()
		if err != nil {
			log.Error().Msgf("Failed to print help: %v", err)
			os.Exit(1)
		}
	},
}

type Cli struct{}

func New(config *config.Config) *Cli {
	cfg = config
	err := newRoot().Execute()
	if err != nil {
		return nil
	}
	return &Cli{}
}

func newRoot() *cobra.Command {
	return rootCmd
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
	rootCmd.PersistentFlags().BoolVarP(
		&observer,
		"observer",
		"o",
		false,
		"Run the actor observer app",
	)
	serverCmd.PersistentFlags().StringVarP(
		&port,
		"port",
		"p",
		"9090",
		"Port to listen on for HTTP requests",
	)
	rootCmd.AddCommand(workflowCmd)
	rootCmd.AddCommand(serverCmd)
}
