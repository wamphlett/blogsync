package cfg

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

// Config is the primary config for the application.
type Config struct {
	Blog    *Blog
	Repo    *Repo
	Git     *Git
	Logging *Logging
	Otel    *Otel
}

// New creates a new Config, populated from environment variables.
func New() (*Config, error) {
	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
