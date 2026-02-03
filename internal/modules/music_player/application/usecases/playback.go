package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PauseInput contains the input for the Pause use case.
type PauseInput struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// ResumeInput contains the input for the Resume use case.
type ResumeInput struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// SkipInput contains the input for the Skip use case.
type SkipInput struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// SkipOutput contains the result of the Skip use case.
type SkipOutput struct {
	SkippedTrack *domain.Track
	NextTrack    *domain.Track // nil if queue is empty
}

// PlaybackService handles playback operations.
type PlaybackService struct {
	repo        domain.PlayerStateRepository
	audioPlayer ports.AudioPlayer
	voiceState  ports.VoiceStateProvider
	publisher   ports.EventPublisher
}

// NewPlaybackService creates a new PlaybackService.
func NewPlaybackService(
	repo domain.PlayerStateRepository,
	audioPlayer ports.AudioPlayer,
	voiceState ports.VoiceStateProvider,
	publisher ports.EventPublisher,
) *PlaybackService {
	return &PlaybackService{
		repo:        repo,
		audioPlayer: audioPlayer,
		voiceState:  voiceState,
		publisher:   publisher,
	}
}

// Pause pauses the current playback.
func (p *PlaybackService) Pause(ctx context.Context, input PauseInput) error {
	state := p.repo.Get(input.GuildID)
	if state == nil {
		return ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	if !state.IsPlaying() {
		if state.IsPaused() {
			return ErrAlreadyPaused
		}
		return ErrNotPlaying
	}

	if err := p.audioPlayer.Pause(ctx, input.GuildID); err != nil {
		return err
	}

	state.SetPaused()

	return nil
}

// Resume resumes the paused playback.
func (p *PlaybackService) Resume(ctx context.Context, input ResumeInput) error {
	state := p.repo.Get(input.GuildID)
	if state == nil {
		return ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	if !state.IsPaused() {
		if state.IsPlaying() {
			return ErrNotPaused
		}
		return ErrNotPlaying
	}

	if err := p.audioPlayer.Resume(ctx, input.GuildID); err != nil {
		return err
	}

	state.SetResumed()

	return nil
}

// Skip skips the current track and plays the next one from the queue.
func (p *PlaybackService) Skip(ctx context.Context, input SkipInput) (*SkipOutput, error) {
	state := p.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	if !state.HasTrack() {
		return nil, ErrNotPlaying
	}

	skippedTrack := state.CurrentTrack()

	// Publish event to delete the "Now Playing" message
	nowPlayingMsg := state.GetNowPlayingMessage()
	if nowPlayingMsg != nil && p.publisher != nil {
		p.publisher.PublishPlaybackFinished(ports.PlaybackFinishedEvent{
			GuildID:               input.GuildID,
			NotificationChannelID: nowPlayingMsg.ChannelID,
			LastMessageID:         &nowPlayingMsg.MessageID,
		})
	}

	// Remove current track from queue
	state.SetStopped()

	// Check if there are more tracks
	if state.Queue.IsEmpty() {
		// Stop playback if no more tracks
		if err := p.audioPlayer.Stop(ctx, input.GuildID); err != nil {
			return nil, err
		}
		return &SkipOutput{
			SkippedTrack: skippedTrack,
			NextTrack:    nil,
		}, nil
	}

	// Play next track (which is now at Queue[0])
	nextTrack, err := p.PlayNext(ctx, input.GuildID)
	if err != nil {
		return nil, err
	}

	return &SkipOutput{
		SkippedTrack: skippedTrack,
		NextTrack:    nextTrack,
	}, nil
}

// PlayNext plays the track at Queue[0].
// Caller is responsible for removing the finished track before calling this.
// Returns the track that started playing, or nil if the queue is empty.
// Returns error if not connected or audio player fails.
func (p *PlaybackService) PlayNext(
	ctx context.Context,
	guildID snowflake.ID,
) (*domain.Track, error) {
	state := p.repo.Get(guildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Get track at Queue[0] (next track to play)
	nextTrack := state.Queue.Peek()
	if nextTrack == nil {
		// No tracks
		return nil, nil
	}

	// Play via audio player
	if err := p.audioPlayer.Play(ctx, guildID, nextTrack); err != nil {
		return nil, err
	}

	state.SetResumed() // Clear paused flag, track is already at Queue[0]

	// Publish event for "Now Playing" notification (async)
	if p.publisher != nil {
		p.publisher.PublishPlaybackStarted(ports.PlaybackStartedEvent{
			GuildID:               guildID,
			Track:                 nextTrack,
			NotificationChannelID: state.NotificationChannelID,
		})
	}

	return nextTrack, nil
}
