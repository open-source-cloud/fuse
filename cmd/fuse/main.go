// FUSE Workflow Engine application server
package main

import (
	"time"

	"github.com/mattn/go-colorable"
	"github.com/open-source-cloud/fuse/internal/cli"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// FUSE Workflow Engine application cli entrypoint
func main() {
	// Initialize logging
	zerolog.TimeFieldFormat = time.TimeOnly
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{
		Out:        colorable.NewColorableStdout(),
		TimeFormat: time.TimeOnly,
	}).With().Caller().Logger()

	cli.Execute()
}
