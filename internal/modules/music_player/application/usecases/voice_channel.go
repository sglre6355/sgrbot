package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/events"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// JoinInput contains the input for the Join use case.
type JoinInput struct {
	GuildID               snowflake.ID
	UserID                snowflake.ID
	NotificationChannelID snowflake.ID
	VoiceChannelID        snowflake.ID // Optional: specific channel to join (0 means use user's channel)
}

// JoinOutput contains the result of the Join use case.
type JoinOutput struct {
	VoiceChannelID snowflake.ID
}

// LeaveInput contains the input for the Leave use case.
type LeaveInput struct {
	GuildID snowflake.ID
}

// BotVoiceStateChangeInput contains the input for handling bot voice state changes.
type BotVoiceStateChangeInput struct {
	GuildID      snowflake.ID
	NewChannelID *snowflake.ID // nil means disconnected
}

// VoiceChannelService handles voice channel operations.
type VoiceChannelService struct {
	repo            domain.PlayerStateRepository
	voiceConnection ports.VoiceConnection
	voiceState      ports.VoiceStateProvider
	bus             *events.Bus
}

// NewVoiceChannelService creates a new VoiceChannelService.
func NewVoiceChannelService(
	repo domain.PlayerStateRepository,
	voiceConnection ports.VoiceConnection,
	voiceState ports.VoiceStateProvider,
	bus *events.Bus,
) *VoiceChannelService {
	return &VoiceChannelService{
		repo:            repo,
		voiceConnection: voiceConnection,
		voiceState:      voiceState,
		bus:             bus,
	}
}

// Join joins the bot to a voice channel.
func (v *VoiceChannelService) Join(ctx context.Context, input JoinInput) (*JoinOutput, error) {
	existingState := v.repo.Get(input.GuildID)

	// Determine which channel to join
	voiceChannelID := input.VoiceChannelID
	if voiceChannelID == 0 {
		// Get user's current voice channel
		userChannel, err := v.voiceState.GetUserVoiceChannel(input.GuildID, input.UserID)
		if err != nil {
			return nil, err
		}
		if userChannel == 0 {
			return nil, ErrUserNotInVoice
		}
		voiceChannelID = userChannel
	}

	// Check if already connected to the same channel - just update notification channel
	if existingState != nil && existingState.VoiceChannelID == voiceChannelID {
		existingState.SetNotificationChannel(input.NotificationChannelID)
		return &JoinOutput{VoiceChannelID: voiceChannelID}, nil
	}

	// Join the channel
	if err := v.voiceConnection.JoinChannel(ctx, input.GuildID, voiceChannelID); err != nil {
		return nil, err
	}

	if existingState != nil {
		// Moving channels - preserve queue, update channel IDs
		existingState.SetVoiceChannel(voiceChannelID)
		existingState.SetNotificationChannel(input.NotificationChannelID)
	} else {
		// Fresh connection - create new state
		state := domain.NewPlayerState(input.GuildID, voiceChannelID, input.NotificationChannelID)
		v.repo.Save(state)
	}

	return &JoinOutput{VoiceChannelID: voiceChannelID}, nil
}

// HandleBotVoiceStateChange handles external voice state changes (bot moved or disconnected).
// This should be called when the bot's voice state changes due to external factors
// (e.g., being moved by a user or disconnected by Discord).
func (v *VoiceChannelService) HandleBotVoiceStateChange(input BotVoiceStateChangeInput) {
	state := v.repo.Get(input.GuildID)
	if state == nil {
		// No player state exists, nothing to do
		return
	}

	if input.NewChannelID == nil {
		// Bot was disconnected from voice
		// Publish event to delete the "Now Playing" message before we lose the state
		nowPlayingMsgID := state.GetNowPlayingMessageID()
		if nowPlayingMsgID != nil && v.bus != nil {
			v.bus.Publish(events.PlaybackFinishedEvent{
				GuildID:               input.GuildID,
				NotificationChannelID: state.NotificationChannelID,
				LastMessageID:         nowPlayingMsgID,
			})
		}

		// Delete player state
		v.repo.Delete(input.GuildID)
		return
	}

	// Bot was moved to a different channel
	if *input.NewChannelID != state.GetVoiceChannelID() {
		state.SetVoiceChannel(*input.NewChannelID)
	}
}

// Leave leaves the voice channel and deletes the player state.
func (v *VoiceChannelService) Leave(ctx context.Context, input LeaveInput) error {
	state := v.repo.Get(input.GuildID)
	if state == nil {
		return ErrNotConnected
	}

	// Publish event to delete the "Now Playing" message before we lose the state
	nowPlayingMsgID := state.GetNowPlayingMessageID()
	if nowPlayingMsgID != nil && v.bus != nil {
		v.bus.Publish(events.PlaybackFinishedEvent{
			GuildID:               input.GuildID,
			NotificationChannelID: state.NotificationChannelID,
			LastMessageID:         nowPlayingMsgID,
		})
	}

	// Leave the channel
	if err := v.voiceConnection.LeaveChannel(ctx, input.GuildID); err != nil {
		return err
	}

	// Delete the entire player state
	v.repo.Delete(input.GuildID)

	return nil
}
