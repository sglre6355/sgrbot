package bot

import (
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
