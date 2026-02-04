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
							MinValue:    floatPtr(1),
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
							MinValue:     floatPtr(1),
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "clear",
					Description: "Clear the queue",
				},
			},
		},
		{
			Name:        "loop",
			Description: "Set the loop mode (or cycle through modes if no option provided)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "Loop mode to set (omit to cycle through modes)",
					Required:    false,
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

func floatPtr(f float64) *float64 {
	return &f
}
