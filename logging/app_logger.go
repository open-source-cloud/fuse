package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

var fieldsOrdering = []string{
	"workflow",
	"node",
	"input",
	"output",
}

// NewAppLogger initializes the logging library
func NewAppLogger() zerolog.Logger {
	// Initialize logging
	zerolog.TimeFieldFormat = time.TimeOnly
	logger := newLogger(defaultFormatCaller).With().Caller().Logger()

	log.Logger = logger
	return logger
}

func newLogger(formatCaller zerolog.Formatter) zerolog.Logger {
	// Initialize logging
	zerolog.TimeFieldFormat = time.TimeOnly
	logger := log.Logger.Output(zerolog.ConsoleWriter{
		Out:                 os.Stdout,
		TimeFormat:          time.TimeOnly,
		FieldsOrder:         fieldsOrdering,
		FormatCaller:        formatCaller,
		FormatFieldName:     formatFieldName,
		FormatFieldValue:    formatFieldValue,
		FormatErrFieldValue: formatErrFieldValue,
	})

	return logger
}
