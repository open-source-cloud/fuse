package logging

import (
	"bytes"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogMessage(format string, args ...any) gen.MessageLog {
	return gen.MessageLog{
		Time:   time.Now(),
		Level:  gen.LogLevelInfo,
		Source: gen.MessageLogNode{Node: "test@localhost"},
		Format: format,
		Args:   args,
	}
}

func TestNewAppLogger_JSONFormat(t *testing.T) {
	cfg := &config.Config{}
	cfg.Params.LogFormat = "json"

	logger := NewAppLogger(cfg)

	var buf bytes.Buffer
	testLogger := logger.Output(&buf)

	testLogger.Info().Str("key", "value").Msg("test message")

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err, "JSON format should produce valid JSON")
	assert.Equal(t, "test message", parsed["message"])
	assert.Equal(t, "value", parsed["key"])
	assert.Equal(t, "info", parsed["level"])
}

func TestNewAppLogger_ConsoleFormat(t *testing.T) {
	cfg := &config.Config{}
	cfg.Params.LogFormat = "console"

	logger := NewAppLogger(cfg)

	var buf bytes.Buffer
	testLogger := logger.Output(zerolog.ConsoleWriter{
		Out:         &buf,
		NoColor:     true,
		FieldsOrder: fieldsOrdering,
	})

	testLogger.Info().Str("key", "value").Msg("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=")

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	assert.Error(t, err, "console format should NOT produce valid JSON")
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "json lowercase", input: "json", expected: LogFormatJSON},
		{name: "json uppercase", input: "JSON", expected: LogFormatJSON},
		{name: "console lowercase", input: "console", expected: LogFormatConsole},
		{name: "console uppercase", input: "CONSOLE", expected: LogFormatConsole},
		{name: "console mixed case", input: "Console", expected: LogFormatConsole},
		{name: "empty defaults to json", input: "", expected: LogFormatJSON},
		{name: "unknown defaults to json", input: "pretty", expected: LogFormatJSON},
		{name: "whitespace trimmed", input: "  console  ", expected: LogFormatConsole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsJSONFormat(t *testing.T) {
	original := logFormat
	defer func() { logFormat = original }()

	logFormat = LogFormatJSON
	assert.True(t, IsJSONFormat())

	logFormat = LogFormatConsole
	assert.False(t, IsJSONFormat())
}

func TestErgoLogger_JSONMode_NoANSI(t *testing.T) {
	original := logFormat
	defer func() { logFormat = original }()
	logFormat = LogFormatJSON

	var buf bytes.Buffer
	el := &ergoLogger{
		logger: zerolog.New(&buf),
	}

	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	el.Log(newTestLogMessage("test ergo message %s", "hello"))

	output := buf.String()
	require.NotEmpty(t, output)
	assert.False(t, ansiRegex.MatchString(output), "JSON mode must not contain ANSI escape codes, got: %s", output)

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err, "ergo JSON output should be valid JSON")
}

func TestErgoLogger_JSONMode_ProcessSource(t *testing.T) {
	original := logFormat
	defer func() { logFormat = original }()
	logFormat = LogFormatJSON

	var buf bytes.Buffer
	el := &ergoLogger{
		logger: zerolog.New(&buf),
	}

	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	msg := gen.MessageLog{
		Time:  time.Now(),
		Level: gen.LogLevelInfo,
		Source: gen.MessageLogProcess{
			Node:     "test@localhost",
			PID:      gen.PID{},
			Name:     "worker",
			Behavior: "TestBehavior",
		},
		Format: "process started %s",
		Args:   []any{"ok"},
	}

	el.Log(msg)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.False(t, ansiRegex.MatchString(output), "JSON mode must not contain ANSI escape codes, got: %s", output)

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err, "ergo JSON output should be valid JSON")
}
