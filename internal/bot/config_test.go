package bot

import (
	"log/slog"
	"testing"
)

func TestLoadConfig_WithValidToken(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token-123")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DiscordToken != "test-token-123" {
		t.Errorf("expected token %q, got %q", "test-token-123", cfg.DiscordToken)
	}
}

func TestLoadConfig_WithEmptyToken(t *testing.T) {
	// Clear the environment variable
	t.Setenv("DISCORD_TOKEN", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing token, got nil")
	}
}

func TestLoadConfig_LogLevelDefault(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("expected log level %v, got %v", slog.LevelInfo, cfg.LogLevel)
	}
}

func TestLoadConfig_LogLevelFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"uppercase", "DEBUG", slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DISCORD_TOKEN", "test-token")
			t.Setenv("LOG_LEVEL", tt.envValue)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.LogLevel != tt.want {
				t.Errorf("expected log level %v, got %v", tt.want, cfg.LogLevel)
			}
		})
	}
}

func TestLoadConfig_LogLevelInvalid(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	t.Setenv("LOG_LEVEL", "invalid")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for invalid log level, got nil")
	}
}
