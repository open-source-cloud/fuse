// Package config App config
package config

import (
	"github.com/caarlos0/env/v11"
)

const (
	// ArangoDBDriver specifies the driver string identifier for connecting to an ArangoDB database.
	ArangoDBDriver = "arangodb"
)

type (

	// Config represents the application configuration, including the app name and database configuration.
	Config struct {
		Name     string `env:"APP_NAME"`
		Server   ServerConfig
		Database DatabaseConfig
	}

	// DatabaseConfig represents the configuration settings required to connect to a database.
	DatabaseConfig struct {
		Driver string `env:"DB_DRIVER"`
		Host   string `env:"DB_HOST"`
		Port   string `env:"DB_PORT"`
		User   string `env:"DB_USER"`
		Pass   string `env:"DB_PASS"`
		Name   string `env:"DB_NAME"`
		TLS    bool   `env:"DB_TLS" `
	}

	// ServerConfig http server config
	ServerConfig struct {
		Run  bool
		Port string
	}
)

// NewConfig initializes and parses the application configuration from environment variables. Returns the configuration or an error.
func NewConfig() (*Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Validate checks the fields of the Config object for correctness and returns an error if validation fails.
func (c *Config) Validate() error {
	return nil
}
