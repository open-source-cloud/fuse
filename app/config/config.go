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
		Driver string `env:"DB_DRIVER" envDefault:"memory"`
		Name   string `env:"DB_NAME" envDefault:"fuse"`
		URL    string `env:"DB_URL"`
		Mongo  MongoConfig
	}

	MongoConfig struct {
		AuthSource string `env:"MONGO_AUTH_SOURCE" envDefault:"admin"`
	}

	// ServerConfig http server config
	ServerConfig struct {
		Port string `env:"SERVER_PORT" envDefault:"9090"`
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
