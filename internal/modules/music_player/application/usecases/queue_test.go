package usecases

import (
	"context"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestQueueService_List(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name             string
		input            QueueListInput
		setupRepo        func(*mockRepository, *mockTrackProvider)
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
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
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
			name: "single page with tracks - active",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.SetPlaybackActive(true)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				for i := range 8 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.SetPlaybackActive(true)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				for i := range 8 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.SetPlaybackActive(true)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.SetPlaybackActive(true)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)
				state.Queue.Advance(0)
				state.SetPlaybackActive(true)
			},
			wantTotalTracks:  5,
			wantPageTracks:   5,
			wantPage:         1,
			wantTotalPages:   1,
			wantCurrentIndex: 2, // advanced twice from 0
			wantPageStart:    0,
		},
		{
			name: "with setupPlaying - prepends track",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("queued"))
				state.Queue.Append(mockTrack("queued").ID)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 10 tracks and advance to index 7 (page 3 with size 3)
				for i := range 10 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				for range 7 {
					state.Queue.Advance(0)
				}
				state.SetPlaybackActive(true)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 10 tracks and advance past end to become idle
				for i := range 10 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				// Advance past end: 10 advances from index 0
				for range 10 {
					state.Queue.Advance(domain.LoopModeNone)
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
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, nil, tp)
			output, err := service.List(context.Background(), tt.input)
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
		setupRepo func(*mockRepository, *mockTrackProvider)
		wantErr   error
		wantID    string
	}{
		{
			name: "remove upcoming track",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 2,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Queue: [0:current, 1:track-1, 2:track-2, 3:track-3], currentIndex=0
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("track-1"))
				tp.Store(mockTrack("track-2"))
				tp.Store(mockTrack("track-3"))
				state.Queue.Append(mockTrack("track-1").ID)
				state.Queue.Append(mockTrack("track-2").ID)
				state.Queue.Append(mockTrack("track-3").ID)
			},
			wantID: "track-2",
		},
		{
			name: "remove played track (before current index)",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add tracks and advance to index 2
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
				state.SetPlaybackActive(true)
			},
			wantID: "track-0", // played track at index 0
		},
		{
			name: "empty queue",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("track-1"))
				state.Queue.Append(mockTrack("track-1").ID)
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "remove current track - returns ErrIsCurrentTrack",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// currentIndex=0 after setupPlaying
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("track-1"))
				state.Queue.Append(mockTrack("track-1").ID)
			},
			wantErr: ErrIsCurrentTrack,
		},
		{
			name: "remove current track after advancing - returns ErrIsCurrentTrack",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: 2,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add tracks and advance to index 2
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
				state.SetPlaybackActive(true)
			},
			wantErr: ErrIsCurrentTrack,
		},
		{
			name: "invalid position - negative",
			input: QueueRemoveInput{
				GuildID:  guildID,
				Position: -1,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("track-1"))
				state.Queue.Append(mockTrack("track-1").ID)
			},
			wantErr: ErrInvalidPosition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, nil, tp)
			output, err := service.Remove(context.Background(), tt.input)

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
		setupRepo    func(*mockRepository, *mockTrackProvider)
		wantPosition int
	}{
		{
			name: "add to empty queue - connected",
			input: QueueAddInput{
				GuildID: guildID,
				Track:   mockTrack("track-1"),
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantPosition: 0,
		},
		{
			name: "add to non-empty queue - position 1",
			input: QueueAddInput{
				GuildID: guildID,
				Track:   mockTrack("track-2"),
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("track-1")) // Already playing
			},
			wantPosition: 1,
		},
		{
			name: "add multiple tracks - position increases",
			input: QueueAddInput{
				GuildID: guildID,
				Track:   mockTrack("track-3"),
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("track-1"))
				tp.Store(mockTrack("track-2"))
				state.Queue.Append(mockTrack("track-1").ID)
				state.Queue.Append(mockTrack("track-2").ID)
			},
			wantPosition: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			publisher := &mockEventPublisher{}
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, publisher, tp)
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
		setupRepo     func(*mockRepository, *mockTrackProvider)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2 (track-2 is current)
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
				state.SetPlaybackActive(true)
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("current"))
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks, then advance past all (idle state)
				for i := range 3 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)                   // index=1
				state.Queue.Advance(0)                   // index=2
				state.Queue.Advance(domain.LoopModeNone) // past end, idle
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
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
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
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
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
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
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
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, nil, tp)
			output, err := service.Clear(context.Background(), tt.input)

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
			state, _ := repo.Get(context.Background(), guildID)
			if state.Queue.Len() != tt.wantRemaining {
				t.Errorf("remaining tracks = %d, want %d", state.Queue.Len(), tt.wantRemaining)
			}
		})
	}
}

func TestQueueService_Restart(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		input       QueueRestartInput
		setupRepo   func(*mockRepository, *mockTrackProvider)
		wantErr     error
		wantTrackID string
	}{
		{
			name: "restart idle queue after it ended",
			input: QueueRestartInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks and advance past end
				for _, id := range []string{"track-0", "track-1", "track-2"} {
					track := mockTrack(id)
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)                   // index=1
				state.Queue.Advance(0)                   // index=2
				state.Queue.Advance(domain.LoopModeNone) // past end, idle
			},
			wantTrackID: "track-0",
		},
		{
			name: "restart while playing (in middle of queue)",
			input: QueueRestartInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks and advance to middle
				for _, id := range []string{"track-0", "track-1", "track-2"} {
					track := mockTrack(id)
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0) // index=1
			},
			wantTrackID: "track-0",
		},
		{
			name: "empty queue",
			input: QueueRestartInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// No tracks
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "not connected",
			input: QueueRestartInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
			},
			wantErr: ErrNotConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			publisher := &mockEventPublisher{}
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, publisher, tp)
			output, err := service.Restart(context.Background(), tt.input)

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

			if string(output.Track.ID) != tt.wantTrackID {
				t.Errorf("expected track ID %q, got %q", tt.wantTrackID, output.Track.ID)
			}

			// Verify event was published
			if len(publisher.trackEnqueued) != 1 {
				t.Fatalf("expected 1 TrackEnqueuedEvent, got %d", len(publisher.trackEnqueued))
			}
			event := publisher.trackEnqueued[0]
			if event.GuildID != tt.input.GuildID {
				t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
			}
			// Verify queue is at position 0 (Restart uses Seek(0))
			state, _ := repo.Get(context.Background(), guildID)
			if state.Queue.CurrentIndex() != 0 {
				t.Errorf(
					"expected currentIndex 0 after Restart, got %d",
					state.Queue.CurrentIndex(),
				)
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
		tp := newMockTrackProvider()

		state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
		tp.Store(mockTrack("track-1"))
		tp.Store(mockTrack("track-2"))
		state.Queue.Append(mockTrack("track-1").ID)
		state.Queue.Append(mockTrack("track-2").ID)

		service := NewQueueService(repo, publisher, tp)
		_, err := service.Clear(context.Background(), QueueClearInput{
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
		tp := newMockTrackProvider()

		state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
		tp.Store(mockTrack("track-1"))
		tp.Store(mockTrack("track-2"))
		tp.Store(mockTrack("track-3"))
		state.Queue.Append(mockTrack("track-1").ID)
		state.Queue.Append(mockTrack("track-2").ID)
		state.Queue.Append(mockTrack("track-3").ID)

		service := NewQueueService(repo, publisher, tp)
		_, err := service.Clear(context.Background(), QueueClearInput{
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

func TestQueueService_Seek(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		input       QueueSeekInput
		setupRepo   func(*mockRepository, *mockTrackProvider)
		wantErr     error
		wantTrackID string
	}{
		{
			name: "seek to middle of queue",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              2,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks (Append auto-activates, at index 0)
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
			},
			wantTrackID: "track-2",
		},
		{
			name: "seek to played track (before current)",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              0,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0)
				state.Queue.Advance(0) // currentIndex=2
			},
			wantTrackID: "track-0",
		},
		{
			name: "seek to upcoming track (after current)",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              4,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks (Append auto-activates, at index 0)
				for i := range 5 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
			},
			wantTrackID: "track-4",
		},
		{
			name: "seek to current position (restarts current)",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              1,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks and advance to index 1
				for i := range 3 {
					track := mockTrack("track-" + string(rune('0'+i)))
					tp.Store(track)
					state.Queue.Append(track.ID)
				}
				state.Queue.Advance(0) // currentIndex=1
			},
			wantTrackID: "track-1",
		},
		{
			name: "seek from idle state (queue ended)",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              1,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 2 tracks and advance past end
				tp.Store(mockTrack("track-0"))
				tp.Store(mockTrack("track-1"))
				state.Queue.Append(mockTrack("track-0").ID)
				state.Queue.Append(mockTrack("track-1").ID)
				state.Queue.Advance(0)                   // index=1
				state.Queue.Advance(domain.LoopModeNone) // past end, idle
			},
			wantTrackID: "track-1",
		},
		{
			name: "empty queue error",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              0,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// No tracks
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "invalid position error - too high",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              10,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				tp.Store(mockTrack("track-0"))
				tp.Store(mockTrack("track-1"))
				state.Queue.Append(mockTrack("track-0").ID)
				state.Queue.Append(mockTrack("track-1").ID)
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "invalid position error - negative",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              -1,
				NotificationChannelID: notificationChannelID,
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				tp.Store(mockTrack("track-0"))
				state.Queue.Append(mockTrack("track-0").ID)
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "not connected error",
			input: QueueSeekInput{
				GuildID:               guildID,
				Position:              0,
				NotificationChannelID: notificationChannelID,
			},
			wantErr: ErrNotConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			publisher := &mockEventPublisher{}
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, publisher, tp)
			output, err := service.Seek(context.Background(), tt.input)

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

			if string(output.Track.ID) != tt.wantTrackID {
				t.Errorf("expected track ID %q, got %q", tt.wantTrackID, output.Track.ID)
			}

			// Verify event was published
			if len(publisher.trackEnqueued) != 1 {
				t.Fatalf("expected 1 TrackEnqueuedEvent, got %d", len(publisher.trackEnqueued))
			}
			event := publisher.trackEnqueued[0]
			if event.GuildID != tt.input.GuildID {
				t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
			}
			// Verify queue is at the seeked position but playback inactive
			// (Seek sets position then SetPlaybackActive(false))
			state, _ := repo.Get(context.Background(), guildID)
			if state.Queue.CurrentIndex() != tt.input.Position {
				t.Errorf(
					"expected currentIndex %d after Seek, got %d",
					tt.input.Position,
					state.Queue.CurrentIndex(),
				)
			}
		})
	}
}

func TestQueueService_AddMultiple(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name              string
		input             QueueAddMultipleInput
		setupRepo         func(*mockRepository, *mockTrackProvider)
		wantErr           error
		wantStartPosition int
		wantCount         int
	}{
		{
			name: "add multiple tracks to empty queue",
			input: QueueAddMultipleInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
				Tracks: []*Track{
					mockTrack("track-1"),
					mockTrack("track-2"),
					mockTrack("track-3"),
				},
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantStartPosition: 0,
			wantCount:         3,
		},
		{
			name: "add multiple tracks to playing queue",
			input: QueueAddMultipleInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
				Tracks: []*Track{
					mockTrack("new-1"),
					mockTrack("new-2"),
				},
			},
			setupRepo: func(m *mockRepository, tp *mockTrackProvider) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				setupPlaying(state, tp, mockTrack("current"))
				tp.Store(mockTrack("queued-1"))
				state.Queue.Append(mockTrack("queued-1").ID)
			},
			wantStartPosition: 2, // after current + queued-1
			wantCount:         2,
		},
		{
			name: "add empty tracks slice",
			input: QueueAddMultipleInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
				Tracks:                []*Track{},
			},
			setupRepo: func(m *mockRepository, _ *mockTrackProvider) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantStartPosition: 0,
			wantCount:         0,
		},
		{
			name: "not connected",
			input: QueueAddMultipleInput{
				GuildID: guildID,
				Tracks:  []*Track{mockTrack("track-1")},
			},
			wantErr: ErrNotConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			publisher := &mockEventPublisher{}
			tp := newMockTrackProvider()

			if tt.setupRepo != nil {
				tt.setupRepo(repo, tp)
			}

			service := NewQueueService(repo, publisher, tp)
			output, err := service.AddMultiple(context.Background(), tt.input)

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

			if output.StartPosition != tt.wantStartPosition {
				t.Errorf("StartPosition = %d, want %d", output.StartPosition, tt.wantStartPosition)
			}

			if output.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", output.Count, tt.wantCount)
			}

			// Check event publishing
			if tt.wantCount == 0 {
				// Empty tracks should not publish event
				if len(publisher.trackEnqueued) != 0 {
					t.Errorf(
						"expected 0 events for empty tracks, got %d",
						len(publisher.trackEnqueued),
					)
				}
			} else {
				// Should publish exactly 1 event for the first track
				if len(publisher.trackEnqueued) != 1 {
					t.Fatalf("expected 1 TrackEnqueuedEvent, got %d", len(publisher.trackEnqueued))
				}
				event := publisher.trackEnqueued[0]
				if event.GuildID != tt.input.GuildID {
					t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
				}
				// Event should contain the first track
				if event.Track.ID != tt.input.Tracks[0].ID {
					t.Errorf("event Track.ID = %q, want %q", event.Track.ID, tt.input.Tracks[0].ID)
				}
			}
		})
	}
}

func TestQueueService_AddMultiple_TracksOrder(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	repo := newMockRepository()
	publisher := &mockEventPublisher{}
	tp := newMockTrackProvider()

	state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
	setupPlaying(state, tp, mockTrack("current"))

	service := NewQueueService(repo, publisher, tp)
	_, err := service.AddMultiple(context.Background(), QueueAddMultipleInput{
		GuildID:               guildID,
		NotificationChannelID: notificationChannelID,
		Tracks: []*Track{
			mockTrack("new-1"),
			mockTrack("new-2"),
			mockTrack("new-3"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tracks are in correct order
	updatedState, _ := repo.Get(context.Background(), guildID)
	tracks := updatedState.Queue.List()
	if len(tracks) != 4 {
		t.Fatalf("expected 4 tracks in queue, got %d", len(tracks))
	}

	expectedOrder := []string{"current", "new-1", "new-2", "new-3"}
	for i, expected := range expectedOrder {
		if string(tracks[i]) != expected {
			t.Errorf("track[%d] = %q, want %q", i, tracks[i], expected)
		}
	}
}
