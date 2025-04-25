package cli

import (
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"time"
)

// InitLogger initializes logging library
func InitLogger() {
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}
	// Initialize logging
	zerolog.TimeFieldFormat = time.TimeOnly
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
}
