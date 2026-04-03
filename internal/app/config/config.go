// Package config App config
package config

import (
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var config *Config

type (
	// Config represents the application configuration.
	Config struct {
		Name    string `env:"APP_NAME"`
		Params  ParamsConfig
		Server  ServerConfig
		Cluster ClusterConfig
	}

	// ParamsConfig configuration parameters
	ParamsConfig struct {
		LogLevel        string
		ActorObserver   bool
		ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
	}

	// ServerConfig http server config
	ServerConfig struct {
		Host string `env:"HOST" envDefault:"0.0.0.0"`
		Port string `env:"PORT" envDefault:"9090"`
	}

	// ClusterConfig configuration for ergo distributed clustering
	ClusterConfig struct {
		Enabled             bool   `env:"CLUSTER_ENABLED" envDefault:"false"`
		NodeName            string `env:"CLUSTER_NODE_NAME"`
		Cookie              string `env:"CLUSTER_COOKIE" envDefault:"fuse-cluster-secret"`
		AcceptorPort        uint16 `env:"CLUSTER_ACCEPTOR_PORT" envDefault:"15000"`
		HeadlessServiceFQDN string `env:"CLUSTER_HEADLESS_SERVICE_FQDN"`
		PeerNodesCSV        string `env:"CLUSTER_PEER_NODES"`
	}
)

// Instance initializes and parses the application configuration from environment variables. Returns the configuration or an error.
func Instance() *Config {
	if config != nil {
		return config
	}
	config = &Config{}
	if err := env.Parse(config); err != nil {
		panic(err)
	}
	return config
}

// Validate checks the fields of the Config object for correctness and returns an error if validation fails.
func (c *Config) Validate() error {
	return nil
}

// PeerNodeNames returns CLUSTER_PEER_NODES split into non-empty trimmed entries (full ergo node names).
func (c *ClusterConfig) PeerNodeNames() []string {
	if c.PeerNodesCSV == "" {
		return nil
	}
	parts := strings.Split(c.PeerNodesCSV, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
