package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestPlaybackService_Pause(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		input       PauseInput
		setupRepo   func(*mockRepository)
		setupPlayer func(*mockAudioPlayer)
		wantErr     error
	}{
		{
			name: "pause successfully",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("track-1"))
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
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantErr: ErrNotPlaying,
		},
		{
			name: "already paused",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("track-1"))
				state.SetPaused()
			},
			wantErr: ErrAlreadyPaused,
		},
		{
			name: "audio player error",
			input: PauseInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("track-1"))
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

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			if tt.setupPlayer != nil {
				tt.setupPlayer(player)
			}

			service := NewPlaybackService(repo, player, nil, nil)
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
			state := repo.Get(guildID)
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
		setupRepo   func(*mockRepository)
		setupPlayer func(*mockAudioPlayer)
		wantErr     error
	}{
		{
			name: "resume successfully",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("track-1"))
				state.SetPaused()
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
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantErr: ErrNotPlaying,
		},
		{
			name: "not paused - playing",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("track-1"))
			},
			wantErr: ErrNotPaused,
		},
		{
			name: "audio player error",
			input: ResumeInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("track-1"))
				state.SetPaused()
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

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			if tt.setupPlayer != nil {
				tt.setupPlayer(player)
			}

			service := NewPlaybackService(repo, player, nil, nil)
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
			state := repo.Get(guildID)
			if state.IsIdle() {
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
		name          string
		input         SkipInput
		setupRepo     func(*mockRepository)
		setupPlayer   func(*mockAudioPlayer)
		wantErr       error
		wantNextTrack bool
	}{
		{
			name: "skip to next track",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("next"))
			},
			wantNextTrack: true,
		},
		{
			name: "skip with empty queue - stop",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("current"))
			},
			wantNextTrack: false,
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
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
			},
			wantErr: ErrNotPlaying,
		},
		{
			name: "audio player play error",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("next"))
			},
			setupPlayer: func(m *mockAudioPlayer) {
				m.playErr = errors.New("play failed")
			},
			wantErr: errors.New("play failed"),
		},
		{
			name: "audio player stop error",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.SetPlaying(mockTrack("current"))
				// Empty queue
			},
			setupPlayer: func(m *mockAudioPlayer) {
				m.stopErr = errors.New("stop failed")
			},
			wantErr: errors.New("stop failed"),
		},
		{
			name: "skip at last track with queue loop wraps to first",
			input: SkipInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.Queue.Add(mockTrack("first"))
				state.Queue.Add(mockTrack("last"))
				state.Queue.Start()                      // Start at first
				state.Queue.Advance(domain.LoopModeNone) // Move to last
				state.StartPlayback()
				state.SetLoopMode(domain.LoopModeQueue)
			},
			wantNextTrack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			player := &mockAudioPlayer{}

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			if tt.setupPlayer != nil {
				tt.setupPlayer(player)
			}

			service := NewPlaybackService(repo, player, nil, nil)
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

			if output.SkippedTrack == nil {
				t.Error("expected SkippedTrack to be set")
			}

			if tt.wantNextTrack {
				if output.NextTrack == nil {
					t.Error("expected NextTrack to be set")
				}
				state := repo.Get(guildID)
				if state.IsIdle() {
					t.Error("expected playback to be active")
				}
			} else {
				if output.NextTrack != nil {
					t.Error("expected NextTrack to be nil")
				}
				state := repo.Get(guildID)
				if !state.IsIdle() {
					t.Error("expected status to be idle")
				}
			}
		})
	}
}

func TestTrackEndReason_ShouldAdvanceQueue(t *testing.T) {
	tests := []struct {
		reason   ports.TrackEndReason
		expected bool
	}{
		{ports.TrackEndFinished, true},
		{ports.TrackEndLoadFailed, true},
		{ports.TrackEndStopped, false},
		{ports.TrackEndReplaced, false},
		{ports.TrackEndCleanup, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			if got := tt.reason.ShouldAdvanceQueue(); got != tt.expected {
				t.Errorf("ShouldAdvanceQueue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlaybackService_PlayNext(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		setupRepo   func(*mockRepository)
		setupPlayer func(*mockAudioPlayer)
		wantErr     error
		wantTrack   bool
	}{
		{
			name: "play next track from queue",
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.Queue.Add(mockTrack("track-1"))
			},
			wantTrack: true,
		},
		{
			name: "returns nil when queue is empty",
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, textChannelID)
				// Empty queue
			},
			wantTrack: false,
		},
		{
			name:    "returns error when not connected",
			wantErr: ErrNotConnected,
		},
		{
			name: "audio player error propagates",
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, textChannelID)
				state.Queue.Add(mockTrack("track-1"))
			},
			setupPlayer: func(m *mockAudioPlayer) {
				m.playErr = errors.New("play failed")
			},
			wantErr: errors.New("play failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			player := &mockAudioPlayer{}

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			if tt.setupPlayer != nil {
				tt.setupPlayer(player)
			}

			service := NewPlaybackService(repo, player, nil, nil)
			track, err := service.PlayNext(context.Background(), guildID)

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

			if tt.wantTrack {
				if track == nil {
					t.Error("expected track to be set")
				}
				state := repo.Get(guildID)
				if state.IsIdle() {
					t.Error("expected playback to be active")
				}
			} else {
				if track != nil {
					t.Error("expected track to be nil")
				}
			}
		})
	}
}

func TestPlaybackService_PlayNext_PreservesQueuePositionOnPlayFailure(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	textChannelID := snowflake.ID(3)

	repo := newMockRepository()
	player := &mockAudioPlayer{
		playErr: errors.New("play failed"),
	}

	state := repo.createConnectedState(guildID, voiceChannelID, textChannelID)
	state.Queue.Add(mockTrack("track-1"))

	service := NewPlaybackService(repo, player, nil, nil)

	// Verify queue is idle before PlayNext
	if !state.Queue.IsIdle() {
		t.Error("expected queue to be idle before PlayNext")
	}

	// Call PlayNext - this will call Start() internally, setting index to 0
	_, err := service.PlayNext(context.Background(), guildID)
	if err == nil {
		t.Error("expected error from PlayNext")
		return
	}

	// Verify playback is marked as inactive but queue position is preserved
	if !state.IsIdle() {
		t.Error("expected playback to be inactive after Play failure")
	}
	// Queue position should be preserved (index 0, not -1)
	if state.Queue.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 (preserved), got %d", state.Queue.CurrentIndex())
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

			service := NewPlaybackService(repo, &mockAudioPlayer{}, nil, nil)
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

			state := repo.Get(guildID)
			if state.LoopMode() != tt.wantMode {
				t.Errorf("expected loop mode %v, got %v", tt.wantMode, state.LoopMode())
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

			service := NewPlaybackService(repo, &mockAudioPlayer{}, nil, nil)
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

			state := repo.Get(guildID)
			if state.LoopMode() != tt.wantStateDMode {
				t.Errorf("expected state loop mode %v, got %v", tt.wantStateDMode, state.LoopMode())
			}
		})
	}
}
