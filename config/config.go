package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the application-wide configuration
type Config struct {
	PDSURL       string        `envconfig:"PDS_URL" default:"https://bsky.social"`
	Collection   string        `envconfig:"COLLECTION" default:"app.bsky.feed.post"`
	QuotesFile   string        `envconfig:"QUOTES_FILE" default:"quotes.json"`
	AccessJWT    string        `envconfig:"ACCESS_JWT" required:"true"`
	RefreshJWT   string        `envconfig:"REFRESH_JWT" required:"true"`
	DID          string        `envconfig:"DID" required:"true"`
	PostInterval time.Duration `envconfig:"POST_INTERVAL" default:"1h"`
	HTTPTimeout  time.Duration `envconfig:"HTTP_TIMEOUT" default:"10s"`
}

// New creates a new configuration instance.
// It automatically loads settings from environment variables and returns an error
// if any required fields are missing
func New() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}
	return &cfg, nil
}
