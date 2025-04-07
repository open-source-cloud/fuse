package debug

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

const (
	levelInfo  = "info"
	levelDebug = "debug"
	levelWarn  = "warn"
	levelError = "error"
)

type LogNode struct{}

func (n *LogNode) ID() string {
	return fmt.Sprintf("%s/log", debugProviderID)
}

func (n *LogNode) InputSchema() *workflow.DataSchema {
	return &workflow.DataSchema{
		Fields: []workflow.FieldSchema{
			{
				FieldName:   "level",
				Type:        "string",
				Required:    true,
				Validations: []string{fmt.Sprintf("in=%s,%s,%s,%s", levelInfo, levelDebug, levelWarn, levelError)},
				Description: "Log level",
				Default:     levelInfo,
			},
			{
				FieldName:   "msgFormat",
				Type:        "string",
				Required:    true,
				Validations: nil,
				Description: "Log message format",
				Default:     nil,
			},
		},
	}
}

func (n *LogNode) OutputSchemas(_ string) *workflow.DataSchema {
	return nil
}

func (n *LogNode) Execute(input map[string]any) (interface{}, error) {
	// Extract and validate input parameters
	level, ok := input["level"]
	if !ok {
		return nil, fmt.Errorf("missing required field: level")
	}
	message, ok := input["msgFormat"]
	if !ok {
		return nil, fmt.Errorf("missing required field: msgFormat")
	}
	msg := message.(string)

	switch level {
	case "info":
		log.Info().Msg(msg)
	case "warn":
		log.Warn().Msg(msg)
	case "debug":
		log.Debug().Msg(msg)
	case "error":
		log.Error().Msg(msg)
	default:
		log.Info().Msg(msg)
	}

	// Return nil output and nil error since no output is produced
	return nil, nil
}
