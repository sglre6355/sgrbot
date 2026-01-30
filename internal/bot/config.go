package bot

import (
	"github.com/caarlos0/env/v11"
)

// Config holds the bot configuration loaded from environment variables.
type Config struct {
	DiscordToken string `env:"DISCORD_TOKEN,notEmpty"`
}

// LoadConfig loads configuration from environment variables.
// Returns an error if required fields are missing.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
