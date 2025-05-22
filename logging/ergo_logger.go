package logging

import (
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/fatih/color"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog"
	"strings"
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
	var source string
	var event *zerolog.Event

	switch message.Level {
	case gen.LogLevelInfo:
		event = l.logger.Info()
	case gen.LogLevelWarning:
		event = l.logger.Warn()
	case gen.LogLevelError:
		event = l.logger.Error()
	case gen.LogLevelPanic:
		event = l.logger.Panic()
	case gen.LogLevelDebug:
		event = l.logger.Debug()
	case gen.LogLevelTrace:
		event = l.logger.Trace()

	default:
		event = l.logger.Info()
	}

	switch src := message.Source.(type) {
	case gen.MessageLogNode:
		source = color.GreenString(src.Node.CRC32())
	case gen.MessageLogNetwork:
		source = color.GreenString("%s-%s", src.Node.CRC32(), src.Peer.CRC32())
	case gen.MessageLogProcess:
		var tag string
		if src.Name.String() == "''" {
			tag = color.MagentaString(src.Behavior)
		} else {
			tag = color.GreenString(src.Name.String())
		}
		source = fmt.Sprintf("%s%s", color.BlueString("%s", src.PID), tag)
	case gen.MessageLogMeta:
		source = color.CyanString("%s", src.Meta)
	default:
		panic(fmt.Sprintf("unknown log source type: %#v", message.Source))
	}

	// we shouldn't modify message.Args (might be used in the other loggers)
	args := make([]any, 0, len(message.Args))
	for i := range message.Args {
		switch message.Args[i].(type) {
		case gen.PID:
			args = append(args, color.BlueString("%s", message.Args[i]))
		case gen.ProcessID:
			args = append(args, color.BlueString("%s", message.Args[i]))
		case gen.Atom:
			args = append(args, color.GreenString("%s", message.Args[i]))
		case gen.Ref:
			args = append(args, color.CyanString("%s", message.Args[i]))
		case gen.Alias:
			args = append(args, color.CyanString("%s", message.Args[i]))
		case gen.Event:
			args = append(args, color.CyanString("%s", message.Args[i]))
		case gen.Version:
			args = append(args, message.Args[i])
		case workflow.ID:
			args = append(args, color.CyanString("%s", message.Args[i]))
		case error:
			args = append(args, color.RedString("%s", message.Args[i]))
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			args = append(args, color.YellowString("%d", message.Args[i]))

		default:
			args = append(args, color.YellowString("%s", message.Args[i]))
		}
	}

	format := strings.ReplaceAll(message.Format, "%d", "%s")
	event.Msgf("%s %s", source, fmt.Sprintf(format, args...))
}

func (l *ergoLogger) Terminate() {}
