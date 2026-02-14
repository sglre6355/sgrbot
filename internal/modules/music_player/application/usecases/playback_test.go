package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestPlaybackService_Pause(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		input       PauseInput
		setupRepo   func(*mockRepository, *mockTrackProvider)
		setupPlayer func(*mockAudioPlayer)
		wantErr     error
	}{
		{
			name: "pause successfully",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("track-1"))
			},
		},
		{
			name: "not connected",
			input: PauseInput{
				GuildID: guildID,
			},
			wantErr: ErrNotConnected,
		},
		{
			name: "not playing - idle",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantErr: ErrNotPlaying,
		},
		{
			name: "already paused",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("track-1"))
				state.SetPaused(true)
			},
			wantErr: ErrAlreadyPaused,
		},
		{
			name: "audio player error",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("track-1"))
			},
			setupPlayer: func(m *mockAudioPlayer) {
				m.pauseErr = errors.New("pause failed")
			},
			wantErr: errors.New("pause failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			player := &mockAudioPlayer{}
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}
			if tt.setupPlayer != nil {
				tt.setupPlayer(player)
			}

			service := NewPlaybackService(repo, player, nil, nil, tp, nil)
			err := service.Pause(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify state was updated
			state, _ := repo.Get(context.Background(), guildID)
			if !state.IsPaused() {
				t.Error("expected status to be paused")
			}
		})
	}
}

func TestPlaybackService_Resume(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		input       ResumeInput
		setupRepo   func(*mockRepository, *mockTrackProvider)
		setupPlayer func(*mockAudioPlayer)
		wantErr     error
	}{
		{
			name: "resume successfully",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("track-1"))
				state.SetPaused(true)
			},
		},
		{
			name: "not connected",
			input: ResumeInput{
				GuildID: guildID,
			},
			wantErr: ErrNotConnected,
		},
		{
			name: "not paused - idle",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantErr: ErrNotPlaying,
		},
		{
			name: "not paused - playing",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("track-1"))
			},
			wantErr: ErrNotPaused,
		},
		{
			name: "audio player error",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("track-1"))
				state.SetPaused(true)
			},
			setupPlayer: func(m *mockAudioPlayer) {
				m.resumeErr = errors.New("resume failed")
			},
			wantErr: errors.New("resume failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			player := &mockAudioPlayer{}
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}
			if tt.setupPlayer != nil {
				tt.setupPlayer(player)
			}

			service := NewPlaybackService(repo, player, nil, nil, tp, nil)
			err := service.Resume(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify state was updated
			state, _ := repo.Get(context.Background(), guildID)
			if !state.IsPlaybackActive() {
				t.Error("expected playback to be active")
			}
		})
	}
}

func TestPlaybackService_Skip(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name       string
		input      SkipInput
		setupRepo  func(*mockRepository, *mockTrackProvider)
		wantErr    error
		wantNextID bool // whether NextTrackID should be non-nil
	}{
		{
			name: "skip to next track",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("next"))
				state.Append(domain.QueueEntry{TrackID: mockTrack("next").ID})
			},
			wantNextID: true,
		},
		{
			name: "skip with empty queue",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				setupPlaying(state, tp, mockTrack("current"))
			},
			wantNextID: false,
		},
		{
			name: "not connected",
			input: SkipInput{
				GuildID: guildID,
			},
			wantErr: ErrNotConnected,
		},
		{
			name: "not playing",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantErr: ErrNotPlaying,
		},
		{
			name: "skip at last track with queue loop wraps to first",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				tp.Store(mockTrack("first"))
				tp.Store(mockTrack("last"))
				state.Append(domain.QueueEntry{TrackID: mockTrack("first").ID})
				state.Append(domain.QueueEntry{TrackID: mockTrack("last").ID})
				state.Advance(domain.LoopModeNone) // Move to last
				state.SetPlaybackActive(true)
				state.SetLoopMode(domain.LoopModeQueue)
			},
			wantNextID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tp := newMockTrackProvider()
			publisher := &mockEventPublisher{}

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewPlaybackService(repo, &mockAudioPlayer{}, publisher, nil, tp, nil)
			output, err := service.Skip(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.SkippedTrackID == "" {
				t.Error("expected SkippedTrackID to be set")
			}

			if tt.wantNextID {
				if output.NextTrackID == nil {
					t.Error("expected NextTrackID to be set")
				}
			} else {
				if output.NextTrackID != nil {
					t.Error("expected NextTrackID to be nil")
				}
			}

			// Skip always publishes CurrentTrackChangedEvent
			if len(publisher.events) != 1 {
				t.Errorf("expected 1 event, got %d", len(publisher.events))
			} else if _, ok := publisher.events[0].(domain.CurrentTrackChangedEvent); !ok {
				t.Errorf("expected CurrentTrackChangedEvent, got %T", publisher.events[0])
			}
		})
	}
}

func TestPlaybackService_SetLoopMode(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name      string
		input     SetLoopModeInput
		setupRepo func(*mockRepository)
		wantErr   error
		wantMode  domain.LoopMode
	}{
		{
			name: "set loop mode to track",
			input: SetLoopModeInput{
				GuildID: guildID,
				Mode:    "track",
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantMode: domain.LoopModeTrack,
		},
		{
			name: "set loop mode to queue",
			input: SetLoopModeInput{
				GuildID: guildID,
				Mode:    "queue",
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantMode: domain.LoopModeQueue,
		},
		{
			name: "set loop mode to none",
			input: SetLoopModeInput{
				GuildID: guildID,
				Mode:    "none",
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetLoopMode(domain.LoopModeTrack) // Start with track mode
			},
			wantMode: domain.LoopModeNone,
		},
		{
			name: "not connected",
			input: SetLoopModeInput{
				GuildID: guildID,
				Mode:    "track",
			},
			wantErr: ErrNotConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			service := NewPlaybackService(
				repo,
				&mockAudioPlayer{},
				nil,
				nil,
				newMockTrackProvider(),
				nil,
			)
			err := service.SetLoopMode(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			state, _ := repo.Get(context.Background(), guildID)
			if state.GetLoopMode() != tt.wantMode {
				t.Errorf("expected loop mode %v, got %v", tt.wantMode, state.GetLoopMode())
			}
		})
	}
}

func TestPlaybackService_CycleLoopMode(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name           string
		input          CycleLoopModeInput
		setupRepo      func(*mockRepository)
		wantErr        error
		wantModeStr    string          // expected output string
		wantStateDMode domain.LoopMode // expected domain mode in state
	}{
		{
			name: "cycle from none to track",
			input: CycleLoopModeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantModeStr:    "track",
			wantStateDMode: domain.LoopModeTrack,
		},
		{
			name: "cycle from track to queue",
			input: CycleLoopModeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetLoopMode(domain.LoopModeTrack)
			},
			wantModeStr:    "queue",
			wantStateDMode: domain.LoopModeQueue,
		},
		{
			name: "cycle from queue to none",
			input: CycleLoopModeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetLoopMode(domain.LoopModeQueue)
			},
			wantModeStr:    "none",
			wantStateDMode: domain.LoopModeNone,
		},
		{
			name: "not connected",
			input: CycleLoopModeInput{
				GuildID: guildID,
			},
			wantErr: ErrNotConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			service := NewPlaybackService(
				repo,
				&mockAudioPlayer{},
				nil,
				nil,
				newMockTrackProvider(),
				nil,
			)
			output, err := service.CycleLoopMode(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.NewMode != tt.wantModeStr {
				t.Errorf("expected output mode %v, got %v", tt.wantModeStr, output.NewMode)
			}

			state, _ := repo.Get(context.Background(), guildID)
			if state.GetLoopMode() != tt.wantStateDMode {
				t.Errorf(
					"expected state loop mode %v, got %v",
					tt.wantStateDMode,
					state.GetLoopMode(),
				)
			}
		})
	}
}
