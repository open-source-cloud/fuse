package server

import (
	"github.com/caarlos0/env/v11"
)

const (
	ArangoDBDriver = "arangodb"
)

type (
	Config struct {
		Name     string `env:"APP_NAME"`
		Database DatabaseConfig
	}
	DatabaseConfig struct {
		Driver string `env:"DB_DRIVER"`
		Host   string `env:"DB_HOST"`
		Port   string `env:"DB_PORT"`
		User   string `env:"DB_USER"`
		Pass   string `env:"DB_PASS"`
		Name   string `env:"DB_NAME"`
		TLS    bool   `env:"DB_TLS" `
	}
)

func NewConfig() (*Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) Validate() error {
	return nil
}
