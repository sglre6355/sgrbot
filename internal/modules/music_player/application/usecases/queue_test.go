package usecases

import (
	"context"
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func TestQueueService_List(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name           string
		input          QueueListInput
		setupRepo      func(*mockRepository)
		wantTotalTrack int // count of QUEUED tracks (excluding current)
		wantPageTracks int // tracks on current page
		wantPage       int
		wantTotalPages int
		wantCurrent    bool // expect current track to be set
	}{
		{
			name: "empty queue",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantTotalTrack: 0,
			wantPageTracks: 0,
			wantPage:       1,
			wantTotalPages: 1,
			wantCurrent:    false,
		},
		{
			name: "single page - only queued tracks (no current)",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks - first becomes "current", rest are queued
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
			},
			wantTotalTrack: 4, // 5 total - 1 current = 4 queued
			wantPageTracks: 4,
			wantPage:       1,
			wantTotalPages: 1,
			wantCurrent:    true,
		},
		{
			name: "multiple pages - first page",
			input: QueueListInput{
				GuildID:  guildID,
				Page:     1,
				PageSize: 3,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 8 tracks - first becomes "current", 7 are queued
				for i := range 8 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
			},
			wantTotalTrack: 7, // 8 total - 1 current = 7 queued
			wantPageTracks: 3,
			wantPage:       1,
			wantTotalPages: 3,
			wantCurrent:    true,
		},
		{
			name: "multiple pages - last page",
			input: QueueListInput{
				GuildID:  guildID,
				Page:     3,
				PageSize: 3,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 8 tracks - first becomes "current", 7 are queued
				for i := range 8 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
			},
			wantTotalTrack: 7,
			wantPageTracks: 1, // 7 queued, page 3 with size 3 = 1 track
			wantPage:       3,
			wantTotalPages: 3,
			wantCurrent:    true,
		},
		{
			name: "page out of range - clamp to last",
			input: QueueListInput{
				GuildID:  guildID,
				Page:     10,
				PageSize: 3,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks - first becomes "current", 4 are queued
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
			},
			wantTotalTrack: 4,
			wantPageTracks: 1, // 4 queued, page 2 (clamped) with size 3 = 1 track
			wantPage:       2,
			wantTotalPages: 2,
			wantCurrent:    true,
		},
		{
			name: "with current track playing and queue",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("queued"))
			},
			wantTotalTrack: 1, // 1 queued track
			wantPageTracks: 1,
			wantPage:       1,
			wantTotalPages: 1,
			wantCurrent:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			service := NewQueueService(repo, nil)
			output, err := service.List(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.TotalTracks != tt.wantTotalTrack {
				t.Errorf("TotalTracks = %d, want %d", output.TotalTracks, tt.wantTotalTrack)
			}
			if len(output.Tracks) != tt.wantPageTracks {
				t.Errorf("len(Tracks) = %d, want %d", len(output.Tracks), tt.wantPageTracks)
			}
			if output.CurrentPage != tt.wantPage {
				t.Errorf("CurrentPage = %d, want %d", output.CurrentPage, tt.wantPage)
			}
			if output.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", output.TotalPages, tt.wantTotalPages)
			}
			if tt.wantCurrent && output.CurrentTrack == nil {
				t.Error("expected CurrentTrack to be set")
			}
			if !tt.wantCurrent && output.CurrentTrack != nil {
				t.Error("expected CurrentTrack to be nil")
			}
		})
	}
}

func TestQueueService_Remove(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name      string
		input     QueueRemoveInput
		setupRepo func(*mockRepository)
		wantErr   error
		wantID    string
	}{
		{
			name: "remove from middle - position 2",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 2,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Queue: [0:current, 1:track-1, 2:track-2, 3:track-3]
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
				state.Queue.Add(mockTrack("track-3"))
			},
			wantID: "track-2", // position 2
		},
		{
			name: "remove first queued - position 1",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Queue: [0:current, 1:track-1, 2:track-2]
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
			},
			wantID: "track-1", // position 1
		},
		{
			name: "empty queue - no queued tracks",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 1,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "only current track - no queued tracks",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "invalid position - too high",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 10,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "position zero - should be handled by Skip at handler level",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "invalid position - negative",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: -1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
			},
			wantErr: ErrInvalidPosition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			service := NewQueueService(repo, nil)
			output, err := service.Remove(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(output.RemovedTrack.ID) != tt.wantID {
				t.Errorf("removed track ID = %q, want %q", output.RemovedTrack.ID, tt.wantID)
			}
		})
	}
}

func TestQueueService_Add(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name         string
		input        QueueAddInput
		setupRepo    func(*mockRepository)
		wantPosition int
		wantWasIdle  bool
	}{
		{
			name: "add to empty queue - connected, publishes with wasIdle=true",
			input: QueueAddInput{
				GuildID: guildID,
				Track:   mockTrack("track-1"),
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantPosition: 0, // becomes current (wasIdle)
			wantWasIdle:  true,
		},
		{
			name: "add to non-empty queue - position 1",
			input: QueueAddInput{
				GuildID: guildID,
				Track:   mockTrack("track-2"),
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("track-1")) // Already playing
			},
			wantPosition: 1,
			wantWasIdle:  false,
		},
		{
			name: "add multiple tracks - position increases",
			input: QueueAddInput{
				GuildID: guildID,
				Track:   mockTrack("track-3"),
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
			},
			wantPosition: 3,
			wantWasIdle:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			publisher := &mockEventPublisher{}

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			service := NewQueueService(repo, publisher)
			output, err := service.Add(context.Background(), tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.Position != tt.wantPosition {
				t.Errorf("Position = %d, want %d", output.Position, tt.wantPosition)
			}

			// Verify the event was published
			if len(publisher.trackEnqueued) != 1 {
				t.Fatalf("expected 1 TrackEnqueuedEvent, got %d", len(publisher.trackEnqueued))
			}
			event := publisher.trackEnqueued[0]
			if event.GuildID != tt.input.GuildID {
				t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
			}
			if event.WasIdle != tt.wantWasIdle {
				t.Errorf("event WasIdle = %v, want %v", event.WasIdle, tt.wantWasIdle)
			}
		})
	}
}

func TestQueueService_Clear(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name          string
		input         QueueClearInput
		setupRepo     func(*mockRepository)
		wantErr       error
		wantCount     int
		wantRemaining int // tracks remaining after clear (should be 1 = current)
	}{
		{
			name: "clear queue - keeps current track",
			input: QueueClearInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
				state.Queue.Add(mockTrack("track-3"))
			},
			wantCount:     3, // 3 queued tracks cleared
			wantRemaining: 1, // current track remains
		},
		{
			name: "empty queue - only current track",
			input: QueueClearInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				// No queued tracks
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "not connected",
			input: QueueClearInput{
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

			service := NewQueueService(repo, nil)
			output, err := service.Clear(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.ClearedCount != tt.wantCount {
				t.Errorf("ClearedCount = %d, want %d", output.ClearedCount, tt.wantCount)
			}

			// Verify remaining tracks
			state := repo.Get(guildID)
			if state.Queue.Len() != tt.wantRemaining {
				t.Errorf("remaining tracks = %d, want %d", state.Queue.Len(), tt.wantRemaining)
			}
		})
	}
}
