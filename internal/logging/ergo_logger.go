package logging

import (
	"fmt"
	"strings"

	"ergo.services/ergo/gen"
	"github.com/fatih/color"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog"
)

type (
	ergoLogger struct {
		logger zerolog.Logger
	}
)

// ErgoLogger creates the logger adapter for Ergo framework
func ErgoLogger() (gen.LoggerBehavior, error) {
	return &ergoLogger{
		logger: newLogger(ergoFormatCaller).With().Logger(),
	}, nil
}

func (l *ergoLogger) Log(message gen.MessageLog) {
	event := l.eventForLevel(message.Level)
	source := ergoSource(message.Source, IsJSONFormat())
	args := ergoArgs(message.Args, IsJSONFormat())

	format := strings.ReplaceAll(message.Format, "%d", "%s")
	msg := fmt.Sprintf(format, args...)
	event.Msgf("%s %s", source, msg)
}

func (l *ergoLogger) eventForLevel(level gen.LogLevel) *zerolog.Event {
	switch level {
	case gen.LogLevelInfo:
		return l.logger.Info()
	case gen.LogLevelWarning:
		return l.logger.Warn()
	case gen.LogLevelError:
		return l.logger.Error()
	case gen.LogLevelPanic:
		// Ergo uses panic-level log lines when reporting actor failures; zerolog.Panic() re-panics
		// after logging, which can crash the node adapter. Record at error instead.
		return l.logger.Error()
	case gen.LogLevelDebug:
		return l.logger.Debug()
	case gen.LogLevelTrace:
		return l.logger.Trace()
	default:
		return l.logger.Info()
	}
}

func ergoSource(src any, jsonMode bool) string {
	switch s := src.(type) {
	case gen.MessageLogNode:
		if jsonMode {
			return s.Node.CRC32()
		}
		return color.GreenString(s.Node.CRC32())

	case gen.MessageLogNetwork:
		if jsonMode {
			return fmt.Sprintf("%s-%s", s.Node.CRC32(), s.Peer.CRC32())
		}
		return color.GreenString("%s-%s", s.Node.CRC32(), s.Peer.CRC32())

	case gen.MessageLogProcess:
		return ergoProcessSource(s, jsonMode)

	case gen.MessageLogMeta:
		if jsonMode {
			return s.Meta.String()
		}
		return color.CyanString("%s", s.Meta)

	default:
		return fmt.Sprintf("unknown-source(%T)", src)
	}
}

func ergoProcessSource(src gen.MessageLogProcess, jsonMode bool) string {
	if jsonMode {
		var tag string
		if src.Name.String() == "''" {
			tag = src.Behavior
		} else {
			tag = src.Name.String()
		}
		return fmt.Sprintf("%s%s", src.PID, tag)
	}

	var tag string
	if src.Name.String() == "''" {
		tag = color.MagentaString(src.Behavior)
	} else {
		tag = color.GreenString(src.Name.String())
	}
	return fmt.Sprintf("%s%s", color.BlueString("%s", src.PID), tag)
}

func ergoArgs(raw []any, jsonMode bool) []any {
	args := make([]any, 0, len(raw))
	for _, a := range raw {
		if jsonMode {
			args = append(args, ergoArgJSON(a))
		} else {
			args = append(args, ergoArgConsole(a))
		}
	}
	return args
}

func ergoArgJSON(a any) any {
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", a)
}

func ergoArgConsole(a any) any {
	switch a.(type) {
	case gen.PID:
		return color.BlueString("%s", a)
	case gen.ProcessID:
		return color.BlueString("%s", a)
	case gen.Atom:
		return color.GreenString("%s", a)
	case gen.Ref:
		return color.CyanString("%s", a)
	case gen.Alias:
		return color.CyanString("%s", a)
	case gen.Event:
		return color.CyanString("%s", a)
	case gen.Version:
		return a
	case workflow.ID:
		return color.CyanString("%s", a)
	case error:
		return color.RedString("%s", a)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return color.YellowString("%d", a)
	default:
		if a == nil {
			return "<nil>"
		}
		return color.YellowString("%s", a)
	}
}

func (l *ergoLogger) Terminate() {
	// terminate is called when an Ergo Logger actor gets terminated
}
