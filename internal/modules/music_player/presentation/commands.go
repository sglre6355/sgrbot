package presentation

import "github.com/bwmarrin/discordgo"

// Commands returns all slash commands for the music player module.
func Commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "join",
			Description: "Join a voice channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Voice channel to join (defaults to your current channel)",
					Required:    false,
					ChannelTypes: []discordgo.ChannelType{
						discordgo.ChannelTypeGuildVoice,
						discordgo.ChannelTypeGuildStageVoice,
					},
				},
			},
		},
		{
			Name:        "leave",
			Description: "Leave the voice channel",
		},
		{
			Name:        "play",
			Description: "Play a track from URL or search",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "query",
					Description:  "URL or search term",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "stop",
			Description: "Stop playback",
		},
		{
			Name:        "pause",
			Description: "Pause playback",
		},
		{
			Name:        "resume",
			Description: "Resume playback",
		},
		{
			Name:        "skip",
			Description: "Skip the current track",
		},
		{
			Name:        "queue",
			Description: "Manage the queue",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "Show the current queue",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "page",
							Description: "Page number",
							Required:    false,
							MinValue:    ptr[float64](1),
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a track from the queue",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionInteger,
							Name:         "position",
							Description:  "Position of the track to remove (1-indexed, as shown in queue list)",
							Required:     true,
							MinValue:     ptr[float64](1),
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "clear",
					Description: "Clear the queue",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "restart",
					Description: "Restart the queue from the beginning",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "shuffle",
					Description: "Shuffle the queue",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "seek",
					Description: "Jump to a specific position in the queue",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionInteger,
							Name:         "position",
							Description:  "Position to jump to (1-indexed, as shown in queue list)",
							Required:     true,
							MinValue:     ptr[float64](1),
							Autocomplete: true,
						},
					},
				},
			},
		},
		{
			Name:        "autoplay",
			Description: "Set auto-play (automatically play related tracks when the queue ends)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "enabled",
					Description: "Enable or disable auto-play",
					Required:    true,
				},
			},
		},
		{
			Name:        "loop",
			Description: "Set the loop mode",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "Loop mode to set",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Off", Value: "none"},
						{Name: "Track", Value: "track"},
						{Name: "Queue", Value: "queue"},
					},
				},
			},
		},
	}
}

// TODO: replace with built-in `new` function in Go 1.26
func ptr[T any](v T) *T {
	return &v
}
