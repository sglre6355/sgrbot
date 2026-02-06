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

// SetLoopModeInput contains the input for the SetLoopMode use case.
type SetLoopModeInput struct {
	GuildID               snowflake.ID
	Mode                  string       // "none", "track", "queue"
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// CycleLoopModeInput contains the input for the CycleLoopMode use case.
type CycleLoopModeInput struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// CycleLoopModeOutput contains the result of the CycleLoopMode use case.
type CycleLoopModeOutput struct {
	NewMode string // "none", "track", "queue"
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
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	if state.IsIdle() {
		return ErrNotPlaying
	}
	if state.IsPaused() {
		return ErrAlreadyPaused
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
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	if state.IsIdle() {
		return ErrNotPlaying
	}
	if !state.IsPaused() {
		return ErrNotPaused
	}

	if err := p.audioPlayer.Resume(ctx, input.GuildID); err != nil {
		return err
	}

	state.SetResumed()

	return nil
}

// Skip skips the current track and plays the next one from the queue.
// Skip always advances to the next track, regardless of loop mode.
func (p *PlaybackService) Skip(ctx context.Context, input SkipInput) (*SkipOutput, error) {
	state := p.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	if !state.HasTrack() {
		return nil, ErrNotPlaying
	}

	skippedTrack := state.CurrentTrack()

	// Publish event to delete the "Now Playing" message
	nowPlayingMsg := state.GetNowPlayingMessage()
	if nowPlayingMsg != nil && p.publisher != nil {
		p.publisher.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
			GuildID:               input.GuildID,
			NotificationChannelID: nowPlayingMsg.ChannelID,
			LastMessageID:         &nowPlayingMsg.MessageID,
		})
	}

	// Advance to next track, using LoopModeNone to ensure we move forward
	// (skip should not repeat the same track even in track loop mode)
	state.Queue.Advance(domain.LoopModeNone)

	// Check if we've reached the end of the queue
	if state.Queue.IsIdle() {
		// If queue loop mode is enabled, wrap to the beginning
		if state.LoopMode() == domain.LoopModeQueue {
			state.Queue.Seek(0)
		} else {
			// Stop playback if no more tracks and not looping queue
			if err := p.audioPlayer.Stop(ctx, input.GuildID); err != nil {
				return nil, err
			}
			state.StopPlayback()
			return &SkipOutput{
				SkippedTrack: skippedTrack,
				NextTrack:    nil,
			}, nil
		}
	}

	// Play the next track (now the current track after advance)
	nextTrack, err := p.PlayNext(ctx, input.GuildID)
	if err != nil {
		return nil, err
	}

	return &SkipOutput{
		SkippedTrack: skippedTrack,
		NextTrack:    nextTrack,
	}, nil
}

// PlayNext plays the current track from the queue.
// If the queue is idle (not started or past end), it starts from index 0.
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

	// Get current track, starting playback only if queue hasn't started yet
	var nextTrack *domain.Track
	if state.Queue.CurrentIndex() < 0 {
		// Queue not started - start from beginning
		nextTrack = state.Queue.Start()
	} else {
		// Queue already started - get current track (may be nil if finished)
		nextTrack = state.Queue.Current()
	}

	if nextTrack == nil {
		// No tracks
		return nil, nil
	}

	// Play via audio player
	if err := p.audioPlayer.Play(ctx, guildID, nextTrack); err != nil {
		// Mark playback as inactive but preserve queue position
		state.StopPlayback()
		return nil, err
	}

	state.StartPlayback()

	// Publish event for "Now Playing" notification (async)
	if p.publisher != nil {
		p.publisher.PublishPlaybackStarted(domain.PlaybackStartedEvent{
			GuildID:               guildID,
			Track:                 nextTrack,
			NotificationChannelID: state.GetNotificationChannelID(),
		})
	}

	return nextTrack, nil
}

// SetLoopMode sets the loop mode for the guild's player.
func (p *PlaybackService) SetLoopMode(ctx context.Context, input SetLoopModeInput) error {
	state := p.repo.Get(input.GuildID)
	if state == nil {
		return ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	state.SetLoopMode(parseLoopMode(input.Mode))

	return nil
}

// parseLoopMode converts a string to domain.LoopMode.
func parseLoopMode(s string) domain.LoopMode {
	switch s {
	case "track":
		return domain.LoopModeTrack
	case "queue":
		return domain.LoopModeQueue
	default:
		return domain.LoopModeNone
	}
}

// CycleLoopMode cycles through loop modes: None -> Track -> Queue -> None.
func (p *PlaybackService) CycleLoopMode(
	ctx context.Context,
	input CycleLoopModeInput,
) (*CycleLoopModeOutput, error) {
	state := p.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	newMode := state.CycleLoopMode()

	return &CycleLoopModeOutput{
		NewMode: newMode.String(),
	}, nil
}
