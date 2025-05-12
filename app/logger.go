package app

import (
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NewLogger initializes logging library
func NewLogger() zerolog.Logger {
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}

	// Initialize logging
	zerolog.TimeFieldFormat = time.TimeOnly
	logger := log.Logger.Output(zerolog.ConsoleWriter{
		Out:        colorable.NewColorableStdout(),
		TimeFormat: time.TimeOnly,
		FieldsOrder: []string{
			"workflow",
			"node",
			"input",
			"output",
		},
		FormatCaller: func(i any) string {
			fullPath, ok := i.(string)
			if !ok {
				return ""
			}
			relPath := fullPath
			if strings.Contains(fullPath, "@") {
				relPath = func() string {
					parts := strings.Split(filepath.ToSlash(fullPath), "/")
					for i, part := range parts {
						if strings.Contains(part, "@") {
							slice := parts[i:]
							slice[0] = "@" + strings.Split(part, "@")[0]
							return strings.Join(slice, string(filepath.Separator))
						}
					}
					// If not found, return the original path
					return fullPath
				}()
			} else if rel, err := filepath.Rel(projectRoot, fullPath); err == nil {
				relPath = rel
			}

			return "\x1b[90m" + relPath + "\x1b[0m" // Cyan color
		},
		FormatFieldName: func(i any) string {
			return "\x1b[94m" + i.(string) + "=\x1b[0m"
		},
	}).With().Caller().Logger()

	log.Logger = logger
	return logger
}

type ergoLogger struct{}

func ErgoLogger() (gen.LoggerBehavior, error) {
	return &ergoLogger{}, nil
}

func (l *ergoLogger) Log(message gen.MessageLog) {
	var source string
	var event *zerolog.Event

	switch message.Level {
	case gen.LogLevelInfo:
		event = log.Info()
	case gen.LogLevelWarning:
		event = log.Warn()
	case gen.LogLevelError:
		event = log.Error()
	case gen.LogLevelPanic:
		event = log.Panic()
	case gen.LogLevelDebug:
		event = log.Debug()
	case gen.LogLevelTrace:
		event = log.Trace()

	default:
		event = log.Info()
	}
	event = event.CallerSkipFrame(1)

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
		source = fmt.Sprintf("%s", color.CyanString("%s", src.Meta))
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
