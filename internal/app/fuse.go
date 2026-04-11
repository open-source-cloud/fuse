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
	"ergo.services/registrar/etcd"
	"github.com/open-source-cloud/fuse/internal/actors"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/rs/zerolog/log"
)

// NewApp creates a new FUSE application, in the context of the FX dependency injection engine and the Ergo framework
func NewApp(
	cfg *config.Config,
	workflowSup *actors.WorkflowSupervisorFactory,
	serverSup *actors.MuxServerSupFactory,
	schemaReplicationSup *actors.SchemaReplicationActorFactory,
	claimActor *actors.WorkflowClaimActorFactory,
	pgListenerActor *actors.PgListenerActorFactory,
) (gen.Node, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var options gen.NodeOptions
	options.ShutdownTimeout = cfg.Params.ShutdownTimeout

	apps := make([]gen.ApplicationBehavior, 0, 2)
	apps = append(apps, &Fuse{
		config:               cfg,
		workflowSup:          workflowSup,
		serverSup:            serverSup,
		schemaReplicationSup: schemaReplicationSup,
		claimActor:           claimActor,
		pgListenerActor:      pgListenerActor,
	})
	if cfg.Params.ActorObserver {
		apps = append(apps, observer.CreateApp(observer.Options{}))
	}
	options.Applications = apps

	// disable default logger to get rid of multiple logging to the os.Stdout
	options.Log.DefaultLogger.Disable = true
	options.Log.Level = parseLogLevel(cfg.Params.LogLevel)

	// add logger to the node
	logger, err := logging.ErgoLogger()
	if err != nil {
		panic(err)
	}
	options.Log.Loggers = append(options.Log.Loggers, gen.Logger{Name: "zerolog", Logger: logger})

	nodeName := buildNodeName(cfg)

	// Configure networking for cluster mode
	if cfg.Cluster.Enabled {
		options.Network.Mode = gen.NetworkModeEnabled
		options.Network.Cookie = cfg.Cluster.Cookie
		options.Network.Acceptors = []gen.AcceptorOptions{
			{
				Host: "0.0.0.0",
				Port: cfg.Cluster.AcceptorPort,
			},
		}
		log.Info().Str("node", string(nodeName)).Uint16("port", cfg.Cluster.AcceptorPort).Msg("starting in cluster mode")
		if cfg.Cluster.DiscoveryModeNormalized() == config.ClusterDiscoveryModeEtcd {
			reg, err := newEtcdRegistrar(cfg)
			if err != nil {
				return nil, err
			}
			options.Network.Registrar = reg
			log.Info().Str("cluster", cfg.Cluster.EtcdCluster).Msg("etcd registrar enabled for cluster discovery")
		}
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

func newEtcdRegistrar(cfg *config.Config) (gen.Registrar, error) {
	opts := etcd.Options{
		Cluster:            cfg.Cluster.EtcdCluster,
		Endpoints:          cfg.Cluster.EtcdEndpointsList(),
		Username:           cfg.Cluster.EtcdUsername,
		Password:           cfg.Cluster.EtcdPassword,
		InsecureSkipVerify: cfg.Cluster.EtcdInsecureSkipVerify,
	}
	if cfg.Cluster.EtcdLeaseTTL > 0 {
		opts.LeaseTTL = cfg.Cluster.EtcdLeaseTTL
	}
	return etcd.Create(opts)
}

// Fuse the FUSE application
type Fuse struct {
	config               *config.Config
	workflowSup          *actors.WorkflowSupervisorFactory
	serverSup            *actors.MuxServerSupFactory
	schemaReplicationSup *actors.SchemaReplicationActorFactory
	claimActor           *actors.WorkflowClaimActorFactory
	pgListenerActor      *actors.PgListenerActorFactory
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
		if app.pgListenerActor.Factory != nil {
			group = append(group, gen.ApplicationMemberSpec{
				Name:    actornames.PgListenerActorName,
				Factory: app.pgListenerActor.Factory,
			})
		}
	}

	return gen.ApplicationSpec{
		Name:        actornames.FuseApplicationName,
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
