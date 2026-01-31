package music_player

// Config holds the music player module configuration.
type Config struct {
	LavalinkAddress  string `env:"LAVALINK_ADDRESS,notEmpty"`
	LavalinkPassword string `env:"LAVALINK_PASSWORD,notEmpty"`
}
