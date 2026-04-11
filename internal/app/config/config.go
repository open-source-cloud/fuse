// Package config App config
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// DBDriverPostgres is the driver value for PostgreSQL persistence.
const DBDriverPostgres = "postgres"

// Cluster discovery modes (CLUSTER_DISCOVERY_MODE).
const (
	ClusterDiscoveryModeStatic = "static"
	ClusterDiscoveryModeEtcd   = "etcd"
)

var config *Config

type (
	// Config represents the application configuration.
	Config struct {
		Name        string `env:"APP_NAME"`
		Params      ParamsConfig
		Server      ServerConfig
		Cluster     ClusterConfig
		Database    DatabaseConfig
		ObjectStore ObjectStoreConfig
		HA          HAConfig
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
		// DiscoveryMode is static (Helm-generated CLUSTER_PEER_NODES) or etcd (BYO registrar for dynamic peers / autoscale).
		DiscoveryMode string `env:"CLUSTER_DISCOVERY_MODE" envDefault:"static"`
		// EtcdEndpointsCSV is a comma-separated list of etcd client endpoints (e.g. http://etcd:2379).
		EtcdEndpointsCSV string `env:"CLUSTER_ETCD_ENDPOINTS"`
		EtcdCluster      string `env:"CLUSTER_ETCD_CLUSTER" envDefault:"default"`
		EtcdUsername     string `env:"CLUSTER_ETCD_USERNAME"`
		EtcdPassword     string `env:"CLUSTER_ETCD_PASSWORD"`
		// EtcdInsecureSkipVerify skips TLS certificate verification when connecting to etcd.
		EtcdInsecureSkipVerify bool `env:"CLUSTER_ETCD_TLS_INSECURE_SKIP_VERIFY" envDefault:"false"`
		// EtcdLeaseTTL is the etcd lease TTL in seconds for node registration (0 = registrar default).
		EtcdLeaseTTL int64 `env:"CLUSTER_ETCD_LEASE_TTL_SEC" envDefault:"0"`
	}

	// DatabaseConfig configuration for the persistence backend
	DatabaseConfig struct {
		Driver          string        `env:"DB_DRIVER" envDefault:"memory"`
		PostgresDSN     string        `env:"DB_POSTGRES_DSN"`
		MaxOpenConns    int32         `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
		MaxIdleConns    int32         `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
		ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m"`
	}

	// ObjectStoreConfig configuration for the pluggable object storage backend
	ObjectStoreConfig struct {
		Driver      string `env:"OBJECT_STORE_DRIVER" envDefault:"memory"`
		KeyPrefix   string `env:"OBJECT_STORE_KEY_PREFIX"`
		FSBasePath  string `env:"OBJECT_STORE_FS_BASE_PATH" envDefault:"./data/fuse"`
		S3Bucket    string `env:"OBJECT_STORE_S3_BUCKET" envDefault:"fuse-data"`
		S3Endpoint  string `env:"OBJECT_STORE_S3_ENDPOINT"`
		S3Region    string `env:"OBJECT_STORE_S3_REGION" envDefault:"us-east-1"`
		S3AccessKey string `env:"OBJECT_STORE_S3_ACCESS_KEY"`
		S3SecretKey string `env:"OBJECT_STORE_S3_SECRET_KEY"`
		S3UseSSL    bool   `env:"OBJECT_STORE_S3_USE_SSL" envDefault:"false"`
	}

	// HAConfig configuration for high availability mode
	HAConfig struct {
		Enabled            bool          `env:"HA_ENABLED" envDefault:"false"`
		NodeID             string        `env:"HA_NODE_ID"`
		HeartbeatInterval  time.Duration `env:"HA_HEARTBEAT_INTERVAL" envDefault:"10s"`
		ClaimSweepInterval time.Duration `env:"HA_CLAIM_SWEEP_INTERVAL" envDefault:"5s"`
		LeaseTimeout       time.Duration `env:"HA_LEASE_TIMEOUT" envDefault:"30s"`
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
	if c.Cluster.Enabled && c.Cluster.DiscoveryModeNormalized() == ClusterDiscoveryModeEtcd {
		if len(c.Cluster.EtcdEndpointsList()) == 0 {
			return fmt.Errorf("CLUSTER_ETCD_ENDPOINTS is required when CLUSTER_ENABLED=true and CLUSTER_DISCOVERY_MODE=etcd")
		}
	}
	return nil
}

// DiscoveryModeNormalized returns static or etcd (default static).
func (c *ClusterConfig) DiscoveryModeNormalized() string {
	s := strings.ToLower(strings.TrimSpace(c.DiscoveryMode))
	if s == "" {
		return ClusterDiscoveryModeStatic
	}
	return s
}

// EtcdEndpointsList returns CLUSTER_ETCD_ENDPOINTS as trimmed non-empty entries.
func (c *ClusterConfig) EtcdEndpointsList() []string {
	if c.EtcdEndpointsCSV == "" {
		return nil
	}
	parts := strings.Split(c.EtcdEndpointsCSV, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
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
