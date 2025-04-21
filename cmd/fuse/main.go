// FUSE Workflow Engine application server
package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/open-source-cloud/fuse/internal/cli"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// FUSE Workflow Engine application cli entrypoint
func main() {
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}
	// Initialize logging
	zerolog.TimeFieldFormat = time.TimeOnly
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{
		Out:        colorable.NewColorableStdout(),
		TimeFormat: time.TimeOnly,
		FieldsOrder: []string{
			"workflow",
			"node",
			"nodeId",
			"input",
			"output",
			"msg",
			"state",
		},
		FormatCaller: func(i any) string {
			fullPath, ok := i.(string)
			if !ok {
				return ""
			}
			relPath := fullPath
			if projectRoot != "" {
				if rel, err := filepath.Rel(projectRoot, fullPath); err == nil {
					relPath = rel
				}
			}
			return "\x1b[90m" + relPath + "\x1b[0m" // Cyan color
		},
		FormatFieldName: func(i any) string {
			return "\x1b[94m" + i.(string) + "=\x1b[0m"
		},
	}).With().Caller().Logger()

	cli.Execute()
}
