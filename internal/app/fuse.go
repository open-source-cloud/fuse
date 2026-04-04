// Package app FUSE Application package
package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"

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
	schemaReplicationSup *actors.SchemaReplicationActorFactory,
	claimActor *actors.WorkflowClaimActorFactory,
) (gen.Node, error) {
	var options gen.NodeOptions
	options.ShutdownTimeout = config.Params.ShutdownTimeout

	apps := make([]gen.ApplicationBehavior, 0, 2)
	apps = append(apps, &Fuse{
		config:               config,
		workflowSup:          workflowSup,
		serverSup:            serverSup,
		schemaReplicationSup: schemaReplicationSup,
		claimActor:           claimActor,
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

	nodeName := buildNodeName(config)

	// Configure networking for cluster mode
	if config.Cluster.Enabled {
		options.Network.Mode = gen.NetworkModeEnabled
		options.Network.Cookie = config.Cluster.Cookie
		options.Network.Acceptors = []gen.AcceptorOptions{
			{
				Host: "0.0.0.0",
				Port: config.Cluster.AcceptorPort,
			},
		}
		log.Info().Str("node", string(nodeName)).Uint16("port", config.Cluster.AcceptorPort).Msg("starting in cluster mode")
	} else {
		options.Network.Mode = gen.NetworkModeDisabled
		log.Info().Str("node", string(nodeName)).Msg("starting in standalone mode")
	}

	node, err := ergo.StartNode(nodeName, options)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func buildNodeName(cfg *config.Config) gen.Atom {
	if cfg.Cluster.Enabled {
		nodeName := cfg.Cluster.NodeName
		if nodeName != "" && strings.Contains(nodeName, "@") {
			return gen.Atom(nodeName)
		}
		podName := os.Getenv("POD_NAME")
		if cfg.Cluster.HeadlessServiceFQDN != "" && podName != "" {
			host := fmt.Sprintf("%s.%s", podName, cfg.Cluster.HeadlessServiceFQDN)
			nodeName = fmt.Sprintf("fuse-%s@%s", podName, host)
		} else {
			podIP := os.Getenv("POD_IP")
			if podName != "" && podIP != "" {
				nodeName = fmt.Sprintf("fuse-%s@%s", podName, podIP)
			} else {
				hostname, _ := os.Hostname()
				nodeName = fmt.Sprintf("fuse@%s", hostname)
			}
		}
		return gen.Atom(nodeName)
	}
	return "fuse@localhost"
}

// Fuse the FUSE application
type Fuse struct {
	config               *config.Config
	workflowSup          *actors.WorkflowSupervisorFactory
	serverSup            *actors.MuxServerSupFactory
	schemaReplicationSup *actors.SchemaReplicationActorFactory
	claimActor           *actors.WorkflowClaimActorFactory
	node                 gen.Node
}

// Load invoked on loading application using the method ApplicationLoad of gen.Node interface.
func (app *Fuse) Load(node gen.Node, _ ...any) (gen.ApplicationSpec, error) {
	app.node = node
	group := []gen.ApplicationMemberSpec{
		{
			Name:    actornames.WorkflowSupervisorName,
			Factory: app.workflowSup.Factory,
		},
		{
			Name:    actornames.MuxServerSupName,
			Factory: app.serverSup.Factory,
		},
		{
			Name:    actornames.SchemaReplicationActorName,
			Factory: app.schemaReplicationSup.Factory,
		},
	}
	if app.config.HA.Enabled {
		group = append(group, gen.ApplicationMemberSpec{
			Name:    actornames.WorkflowClaimActorName,
			Factory: app.claimActor.Factory,
		})
	}

	return gen.ApplicationSpec{
		Name:        "fuse_app",
		Description: "FUSE application",
		Group:       group,
		Mode:        gen.ApplicationModeTemporary,
		Tags:        []gen.Atom{"v0.1.0"},
		LogLevel:    parseLogLevel(app.config.Params.LogLevel),
	}, nil
}

// Start invoked once the application started
func (app *Fuse) Start(_ gen.ApplicationMode) {
	// Trigger workflow recovery for any in-progress workflows from a previous run
	recoverMsg := messaging.Message{Type: messaging.RecoverWorkflows}
	if err := app.node.Send(gen.Atom(actornames.WorkflowSupervisorName), recoverMsg); err != nil {
		log.Error().Err(err).Msg("failed to send workflow recovery message")
	}
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
