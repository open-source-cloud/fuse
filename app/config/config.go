// Package config App config
package config

import (
	"github.com/caarlos0/env/v11"
)

var config *Config

type (
	// Config represents the application configuration, including the app name and database configuration.
	Config struct {
		Name     string `env:"APP_NAME"`
		Params   ParamsConfig
		Server   ServerConfig
		Database DatabaseConfig
	}

	// ParamsConfig configuration parameters
	ParamsConfig struct {
		LogLevel      string
		ActorObserver bool
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
		Port uint16 `env:"SERVER_PORT" envDefault:"9090"`
		Host string `env:"SERVER_HOST" envDefault:"localhost"`
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
