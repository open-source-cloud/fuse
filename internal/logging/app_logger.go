// Package logging helpers and adapters for application logging
package logging

import (
	"os"
	"strings"
	"time"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogFormatJSON selects structured JSON output (default, production-ready).
const LogFormatJSON = "json"

// LogFormatConsole selects human-readable colorized console output.
const LogFormatConsole = "console"

var fieldsOrdering = []string{
	"workflow",
	"node",
	"input",
	"output",
}

// logFormat holds the resolved format so other loggers in this package can read it.
var logFormat = LogFormatJSON

// IsJSONFormat reports whether the active log format is JSON.
func IsJSONFormat() bool {
	return logFormat == LogFormatJSON
}

// NewAppLogger initializes the logging library using the format from config.
func NewAppLogger(cfg *config.Config) zerolog.Logger {
	logFormat = resolveFormat(cfg.Params.LogFormat)

	logger := newLogger(defaultFormatCaller).With().Caller().Logger()
	log.Logger = logger
	return logger
}

func resolveFormat(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == LogFormatConsole {
		return LogFormatConsole
	}
	return LogFormatJSON
}

func newLogger(formatCaller zerolog.Formatter) zerolog.Logger {
	if logFormat == LogFormatJSON {
		zerolog.TimeFieldFormat = time.RFC3339
		return zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	zerolog.TimeFieldFormat = time.TimeOnly
	return log.Logger.Output(zerolog.ConsoleWriter{
		Out:                 os.Stdout,
		TimeFormat:          time.TimeOnly,
		FieldsOrder:         fieldsOrdering,
		FormatCaller:        formatCaller,
		FormatFieldName:     formatFieldName,
		FormatFieldValue:    formatFieldValue,
		FormatErrFieldValue: formatErrFieldValue,
	})
}
