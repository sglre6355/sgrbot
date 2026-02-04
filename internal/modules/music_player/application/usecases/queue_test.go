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
		name             string
		input            QueueListInput
		setupRepo        func(*mockRepository)
		wantTotalTracks  int // total tracks in queue
		wantPageTracks   int // tracks on current page
		wantPage         int
		wantTotalPages   int
		wantCurrentIndex int // -1 if idle
		wantPageStart    int // 0-indexed start position
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
			wantTotalTracks:  0,
			wantPageTracks:   0,
			wantPage:         1,
			wantTotalPages:   1,
			wantCurrentIndex: -1,
			wantPageStart:    0,
		},
		{
			name: "single page with tracks - idle (not started)",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks without starting playback
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
			},
			wantTotalTracks:  5,
			wantPageTracks:   5,
			wantPage:         1,
			wantTotalPages:   1,
			wantCurrentIndex: -1, // not started
			wantPageStart:    0,
		},
		{
			name: "single page with tracks - playing (started)",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and start playback
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
			},
			wantTotalTracks:  5,
			wantPageTracks:   5,
			wantPage:         1,
			wantTotalPages:   1,
			wantCurrentIndex: 0,
			wantPageStart:    0,
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
				// Add 8 tracks and start playback
				for i := range 8 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
			},
			wantTotalTracks:  8,
			wantPageTracks:   3,
			wantPage:         1,
			wantTotalPages:   3,
			wantCurrentIndex: 0,
			wantPageStart:    0,
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
				// Add 8 tracks and start playback
				for i := range 8 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
			},
			wantTotalTracks:  8,
			wantPageTracks:   2, // 8 tracks, page 3 with size 3 = tracks 6-7
			wantPage:         3,
			wantTotalPages:   3,
			wantCurrentIndex: 0,
			wantPageStart:    6,
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
				// Add 5 tracks and start playback
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
			},
			wantTotalTracks:  5,
			wantPageTracks:   2, // 5 tracks, page 2 (clamped) with size 3 = tracks 3-4
			wantPage:         2,
			wantTotalPages:   2,
			wantCurrentIndex: 0,
			wantPageStart:    3,
		},
		{
			name: "current track in middle of queue",
			input: QueueListInput{
				GuildID:  guildID,
				Page:     1,
				PageSize: 5,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				state.Queue.Advance(0)
				state.Queue.Advance(0)
			},
			wantTotalTracks:  5,
			wantPageTracks:   5,
			wantPage:         1,
			wantTotalPages:   1,
			wantCurrentIndex: 2, // advanced twice from 0
			wantPageStart:    0,
		},
		{
			name: "with SetPlaying - prepends track",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("queued"))
			},
			wantTotalTracks:  2, // current + queued
			wantPageTracks:   2,
			wantPage:         1,
			wantTotalPages:   1,
			wantCurrentIndex: 0,
			wantPageStart:    0,
		},
		{
			name: "default page - shows page containing current track",
			input: QueueListInput{
				GuildID:  guildID,
				Page:     0, // No page specified
				PageSize: 3,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 10 tracks and advance to index 7 (page 3 with size 3)
				for i := range 10 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				for range 7 {
					state.Queue.Advance(0)
				}
			},
			wantTotalTracks:  10,
			wantPageTracks:   3, // tracks 6, 7, 8 on page 3
			wantPage:         3, // index 7 / pageSize 3 + 1 = page 3
			wantTotalPages:   4,
			wantCurrentIndex: 7,
			wantPageStart:    6,
		},
		{
			name: "default page - idle defaults to page 1",
			input: QueueListInput{
				GuildID:  guildID,
				Page:     0, // No page specified
				PageSize: 3,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 10 tracks but don't start playback
				for i := range 10 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
			},
			wantTotalTracks:  10,
			wantPageTracks:   3,
			wantPage:         1, // idle defaults to page 1
			wantTotalPages:   4,
			wantCurrentIndex: -1,
			wantPageStart:    0,
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

			if output.TotalTracks != tt.wantTotalTracks {
				t.Errorf("TotalTracks = %d, want %d", output.TotalTracks, tt.wantTotalTracks)
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
			if output.CurrentIndex != tt.wantCurrentIndex {
				t.Errorf("CurrentIndex = %d, want %d", output.CurrentIndex, tt.wantCurrentIndex)
			}
			if output.PageStart != tt.wantPageStart {
				t.Errorf("PageStart = %d, want %d", output.PageStart, tt.wantPageStart)
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
			name: "remove upcoming track",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 2,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Queue: [0:current, 1:track-1, 2:track-2, 3:track-3], currentIndex=0
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
				state.Queue.Add(mockTrack("track-3"))
			},
			wantID: "track-2",
		},
		{
			name: "remove played track (before current index)",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add tracks and advance to index 2
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
			},
			wantID: "track-0", // played track at index 0
		},
		{
			name: "empty queue",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
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
			name: "remove current track - returns ErrIsCurrentTrack",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// currentIndex=0 after SetPlaying
				state.SetPlaying(mockTrack("current"))
				state.Queue.Add(mockTrack("track-1"))
			},
			wantErr: ErrIsCurrentTrack,
		},
		{
			name: "remove current track after advancing - returns ErrIsCurrentTrack",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 2,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add tracks and advance to index 2
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
			},
			wantErr: ErrIsCurrentTrack,
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
		wantRemaining int
	}{
		{
			name: "KeepCurrentTrack=true - clears played and upcoming, keeps only current",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: true,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2 (track-2 is current)
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
			},
			wantCount:     4, // 2 played + 2 upcoming cleared
			wantRemaining: 1, // only current track remains
		},
		{
			name: "KeepCurrentTrack=true - only current track, nothing to clear",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: true,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.SetPlaying(mockTrack("current"))
				// No other tracks
			},
			wantErr: ErrNothingToClear,
		},
		{
			name: "KeepCurrentTrack=true - idle state with played tracks clears all",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: true,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks, start, then advance past all (idle state)
				for i := range 3 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				state.Queue.Advance(0) // index=1
				state.Queue.Advance(0) // index=2
				state.Queue.Advance(0) // index=3 (past end, idle)
			},
			wantCount:     3, // all 3 played tracks cleared
			wantRemaining: 0, // nothing remains
		},
		{
			name: "KeepCurrentTrack=true - idle state with empty queue",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: true,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// No tracks at all
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "KeepCurrentTrack=false - clears all tracks",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: false,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2
				for i := range 5 {
					state.Queue.Add(mockTrack("track-" + string(rune('0'+i))))
				}
				state.Queue.Start()
				state.Queue.Advance(0)
				state.Queue.Advance(0)
			},
			wantCount:     5, // all 5 tracks cleared
			wantRemaining: 0, // nothing remains
		},
		{
			name: "KeepCurrentTrack=false - empty queue",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: false,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// No tracks
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

func TestQueueService_Clear_PublishesQueueClearedEvent(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	t.Run("KeepCurrentTrack=false publishes QueueClearedEvent", func(t *testing.T) {
		repo := newMockRepository()
		publisher := &mockEventPublisher{}

		state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
		state.Queue.Add(mockTrack("track-1"))
		state.Queue.Add(mockTrack("track-2"))
		state.Queue.Start()

		service := NewQueueService(repo, publisher)
		_, err := service.Clear(QueueClearInput{
			GuildID:               guildID,
			NotificationChannelID: notificationChannelID,
			KeepCurrentTrack:      false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify QueueClearedEvent was published
		if len(publisher.queueCleared) != 1 {
			t.Fatalf("expected 1 QueueClearedEvent, got %d", len(publisher.queueCleared))
		}

		event := publisher.queueCleared[0]
		if event.GuildID != guildID {
			t.Errorf("event GuildID = %d, want %d", event.GuildID, guildID)
		}
		if event.NotificationChannelID != notificationChannelID {
			t.Errorf(
				"event NotificationChannelID = %d, want %d",
				event.NotificationChannelID,
				notificationChannelID,
			)
		}
	})

	t.Run("KeepCurrentTrack=true does not publish QueueClearedEvent", func(t *testing.T) {
		repo := newMockRepository()
		publisher := &mockEventPublisher{}

		state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
		state.Queue.Add(mockTrack("track-1"))
		state.Queue.Add(mockTrack("track-2"))
		state.Queue.Add(mockTrack("track-3"))
		state.Queue.Start()

		service := NewQueueService(repo, publisher)
		_, err := service.Clear(QueueClearInput{
			GuildID:               guildID,
			NotificationChannelID: notificationChannelID,
			KeepCurrentTrack:      true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify QueueClearedEvent was NOT published
		if len(publisher.queueCleared) != 0 {
			t.Errorf("expected 0 QueueClearedEvents, got %d", len(publisher.queueCleared))
		}
	})
}
