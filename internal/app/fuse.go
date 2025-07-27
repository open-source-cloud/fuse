// Package app FUSE Application package
package app

import (
	"strings"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"

	"ergo.services/application/observer"
	"ergo.services/ergo"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/rs/zerolog/log"
)

// NewApp creates a new FUSE application, in the context of the FX dependency injection engine and the Ergo framework
func NewApp(
	config *config.Config,
	workflowSup *actors.WorkflowSupervisorFactory,
	serverSup *actors.MuxServerSupFactory,
) (gen.Node, error) {
	var options gen.NodeOptions

	apps := make([]gen.ApplicationBehavior, 0, 2)
	apps = append(apps, &Fuse{
		config:      config,
		workflowSup: workflowSup,
		serverSup:   serverSup,
	})
	if config.Params.ActorObserver {
		apps = append(apps, observer.CreateApp(observer.Options{}))
	}
	options.Applications = apps

	// disable default logger to get rid of multiple logging to the os.Stdout
	options.Log.DefaultLogger.Disable = true
	options.Log.Level = parseLogLevel(config.Params.LogLevel)

	// add logger to the node
	logger, err := logging.ErgoLogger()
	if err != nil {
		panic(err)
	}
	options.Log.Loggers = append(options.Log.Loggers, gen.Logger{Name: "zerolog", Logger: logger})

	node, err := ergo.StartNode("fuse@localhost", options)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// Fuse the FUSE application
type Fuse struct {
	config      *config.Config
	workflowSup *actors.WorkflowSupervisorFactory
	serverSup   *actors.MuxServerSupFactory
}

// Load invoked on loading application using the method ApplicationLoad of gen.Node interface.
func (app *Fuse) Load(_ gen.Node, _ ...any) (gen.ApplicationSpec, error) {
	return gen.ApplicationSpec{
		Name:        "fuse_app",
		Description: "FUSE application",
		Group: []gen.ApplicationMemberSpec{
			{
				Name:    actornames.WorkflowSupervisorName,
				Factory: app.workflowSup.Factory,
			},
			{
				Name:    actornames.MuxServerSupName,
				Factory: app.serverSup.Factory,
			},
		},
		Mode:     gen.ApplicationModeTemporary,
		LogLevel: parseLogLevel(app.config.Params.LogLevel),
	}, nil
}

// Start invoked once the application started
func (app *Fuse) Start(_ gen.ApplicationMode) {
	// any start code for the actor model application start goes here...
}

// Terminate invoked once the application stopped
func (app *Fuse) Terminate(_ error) {
	log.Info().Msg("FUSE application terminated")
}

func parseLogLevel(s string) gen.LogLevel {
	switch strings.ToLower(s) {
	case "trace":
		return gen.LogLevelTrace
	case "debug":
		return gen.LogLevelDebug
	case "info":
		return gen.LogLevelInfo
	case "warning":
		return gen.LogLevelWarning
	case "error":
		return gen.LogLevelError
	case "panic":
		return gen.LogLevelPanic
	case "disabled":
		return gen.LogLevelDisabled
	case "system":
		return gen.LogLevelSystem
	default:
		return gen.LogLevelDefault
	}
}
