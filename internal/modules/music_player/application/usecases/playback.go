package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PlaybackService handles playback operations.
type PlaybackService struct {
	playerStates  domain.PlayerStateRepository
	audioPlayer   ports.AudioPlayer
	publisher     ports.EventPublisher
	notifier      ports.NotificationSender
	trackProvider ports.TrackProvider
	voiceState    ports.VoiceStateProvider
}

// NewPlaybackService creates a new PlaybackService.
func NewPlaybackService(
	playerStates domain.PlayerStateRepository,
	audioPlayer ports.AudioPlayer,
	publisher ports.EventPublisher,
	notifier ports.NotificationSender,
	trackProvider ports.TrackProvider,
	voiceState ports.VoiceStateProvider,
) *PlaybackService {
	return &PlaybackService{
		playerStates:  playerStates,
		audioPlayer:   audioPlayer,
		publisher:     publisher,
		notifier:      notifier,
		trackProvider: trackProvider,
		voiceState:    voiceState,
	}
}

// PauseInput contains the input for the Pause use case.
type PauseInput struct {
	GuildID snowflake.ID
}

// Pause pauses the current playback.
func (p *PlaybackService) Pause(ctx context.Context, input PauseInput) error {
	state, err := p.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return ErrNotConnected
	}

	if !state.IsPlaybackActive() {
		return ErrNotPlaying
	}
	if state.IsPaused() {
		return ErrAlreadyPaused
	}

	if err := p.audioPlayer.Pause(ctx, input.GuildID); err != nil {
		return err
	}

	state.SetPaused(true)

	return p.playerStates.Save(ctx, state)
}

// ResumeInput contains the input for the Resume use case.
type ResumeInput struct {
	GuildID snowflake.ID
}

// Resume resumes the paused playback.
func (p *PlaybackService) Resume(ctx context.Context, input ResumeInput) error {
	state, err := p.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return ErrNotConnected
	}

	if !state.IsPlaybackActive() {
		return ErrNotPlaying
	}
	if !state.IsPaused() {
		return ErrNotPaused
	}

	if err := p.audioPlayer.Resume(ctx, input.GuildID); err != nil {
		return err
	}

	state.SetPaused(false)

	return p.playerStates.Save(ctx, state)
}

// SkipInput contains the input for the Skip use case.
type SkipInput struct {
	GuildID snowflake.ID
}

// SkipOutput contains the result of the Skip use case.
type SkipOutput struct {
	SkippedTrackID string
	NextTrackID    *string // nil if queue is empty
}

// Skip skips the current track and plays the next one from the queue.
func (p *PlaybackService) Skip(ctx context.Context, input SkipInput) (*SkipOutput, error) {
	state, err := p.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	skipped := state.Current()
	if skipped == nil {
		return nil, ErrNotPlaying
	}

	loopmode := state.GetLoopMode()
	if loopmode == domain.LoopModeTrack {
		loopmode = domain.LoopModeNone
	}
	next := state.Advance(loopmode)

	var nextID *string
	if next != nil {
		nextID = (*string)(&next.TrackID)
	} else {
		state.SetPlaybackActive(false)
	}

	if err := p.playerStates.Save(ctx, state); err != nil {
		return nil, err
	}

	// Publish event to trigger playback and notification via standard handlers
	if err := p.publisher.Publish(domain.NewCurrentTrackChangedEvent(input.GuildID)); err != nil {
		return nil, err
	}

	return &SkipOutput{
		SkippedTrackID: skipped.TrackID.String(),
		NextTrackID:    nextID,
	}, nil
}

// SetLoopModeInput contains the input for the SetLoopMode use case.
type SetLoopModeInput struct {
	GuildID snowflake.ID
	Mode    string // "none", "track", "queue"
}

// SetLoopMode sets the loop mode for the guild's player.
func (p *PlaybackService) SetLoopMode(ctx context.Context, input SetLoopModeInput) error {
	state, err := p.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return ErrNotConnected
	}

	state.SetLoopMode(domain.ParseLoopMode(input.Mode))

	return p.playerStates.Save(ctx, state)
}

// CycleLoopModeInput contains the input for the CycleLoopMode use case.
type CycleLoopModeInput struct {
	GuildID snowflake.ID
}

// CycleLoopModeOutput contains the result of the CycleLoopMode use case.
type CycleLoopModeOutput struct {
	NewMode string // "none", "track", "queue"
}

// CycleLoopMode cycles through loop modes: None -> Track -> Queue -> None.
func (p *PlaybackService) CycleLoopMode(
	ctx context.Context,
	input CycleLoopModeInput,
) (*CycleLoopModeOutput, error) {
	state, err := p.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	newMode := state.CycleLoopMode()

	if err := p.playerStates.Save(ctx, state); err != nil {
		return nil, err
	}

	return &CycleLoopModeOutput{
		NewMode: newMode.String(),
	}, nil
}
