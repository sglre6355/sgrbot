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
		name            string
		input           QueueListInput
		setupRepo       func(*mockRepository)
		wantTotalTracks int // total tracks in queue
		wantPlayed      int // played tracks on current page
		wantCurrent     bool
		wantUpcoming    int // upcoming tracks on current page
		wantPage        int
		wantTotalPages  int
		wantPageStart   int // 0-indexed start position
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
			wantTotalTracks: 0,
			wantPlayed:      0,
			wantCurrent:     false,
			wantUpcoming:    0,
			wantPage:        1,
			wantTotalPages:  1,
			wantPageStart:   0,
		},
		{
			name: "single page with tracks - active",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				for i := range 5 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.SetPlaybackActive(true)
			},
			wantTotalTracks: 5,
			wantPlayed:      0,
			wantCurrent:     true,
			wantUpcoming:    4,
			wantPage:        1,
			wantTotalPages:  1,
			wantPageStart:   0,
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
				for i := range 8 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.SetPlaybackActive(true)
			},
			wantTotalTracks: 8,
			wantPlayed:      0,
			wantCurrent:     true,
			wantUpcoming:    2,
			wantPage:        1,
			wantTotalPages:  3,
			wantPageStart:   0,
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
				for i := range 8 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.SetPlaybackActive(true)
			},
			wantTotalTracks: 8,
			wantPlayed:      0,
			wantCurrent:     false, // current at index 0, not on page 3
			wantUpcoming:    2,     // 8 tracks, page 3 with size 3 = tracks 6-7
			wantPage:        3,
			wantTotalPages:  3,
			wantPageStart:   6,
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
				for i := range 5 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.SetPlaybackActive(true)
			},
			wantTotalTracks: 5,
			wantPlayed:      0,
			wantCurrent:     false, // current at index 0, not on clamped page 2
			wantUpcoming:    2,     // 5 tracks, page 2 (clamped) with size 3 = tracks 3-4
			wantPage:        2,
			wantTotalPages:  2,
			wantPageStart:   3,
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
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone)
				state.Advance(domain.LoopModeNone)
				state.SetPlaybackActive(true)
			},
			wantTotalTracks: 5,
			wantPlayed:      2, // indices 0, 1
			wantCurrent:     true,
			wantUpcoming:    2, // indices 3, 4
			wantPage:        1,
			wantTotalPages:  1,
			wantPageStart:   0,
		},
		{
			name: "with playing track - prepends track",
			input: QueueListInput{
				GuildID: guildID,
				Page:    1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("queued")})
			},
			wantTotalTracks: 2, // current + queued
			wantPlayed:      0,
			wantCurrent:     true,
			wantUpcoming:    1,
			wantPage:        1,
			wantTotalPages:  1,
			wantPageStart:   0,
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
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				for range 7 {
					state.Advance(domain.LoopModeNone)
				}
				state.SetPlaybackActive(true)
			},
			wantTotalTracks: 10,
			wantPlayed:      1, // index 6
			wantCurrent:     true,
			wantUpcoming:    1, // index 8
			wantPage:        3, // index 7 / pageSize 3 + 1 = page 3
			wantTotalPages:  4,
			wantPageStart:   6,
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
				// Add 10 tracks and advance past end to become idle
				for i := range 10 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				// Advance past end: 10 advances from index 0
				for range 10 {
					state.Advance(domain.LoopModeNone)
				}
			},
			wantTotalTracks: 10,
			wantPlayed:      0,
			wantCurrent:     false,
			wantUpcoming:    3, // all tracks are upcoming when idle
			wantPage:        1, // idle defaults to page 1
			wantTotalPages:  4,
			wantPageStart:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			service := NewQueueService(repo, &mockEventPublisher{})
			output, err := service.List(context.Background(), tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.TotalTracks != tt.wantTotalTracks {
				t.Errorf("TotalTracks = %d, want %d", output.TotalTracks, tt.wantTotalTracks)
			}
			if len(output.PlayedTrackIDs) != tt.wantPlayed {
				t.Errorf(
					"len(PlayedTrackIDs) = %d, want %d",
					len(output.PlayedTrackIDs),
					tt.wantPlayed,
				)
			}
			hasCurrent := output.CurrentTrackID != ""
			if hasCurrent != tt.wantCurrent {
				t.Errorf("has CurrentTrackID = %v, want %v", hasCurrent, tt.wantCurrent)
			}
			if len(output.UpcomingTrackIDs) != tt.wantUpcoming {
				t.Errorf(
					"len(UpcomingTrackIDs) = %d, want %d",
					len(output.UpcomingTrackIDs),
					tt.wantUpcoming,
				)
			}
			if output.CurrentPage != tt.wantPage {
				t.Errorf("CurrentPage = %d, want %d", output.CurrentPage, tt.wantPage)
			}
			if output.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", output.TotalPages, tt.wantTotalPages)
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
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-1")})
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-2")})
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-3")})
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
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone)
				state.Advance(domain.LoopModeNone) // currentIndex=2
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
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-1")})
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
				// currentIndex=0 after SetPlaybackActive
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-1")})
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
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone)
				state.Advance(domain.LoopModeNone) // currentIndex=2
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
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-1")})
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

			service := NewQueueService(repo, &mockEventPublisher{})
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

			if output.RemovedTrackID != tt.wantID {
				t.Errorf("removed track ID = %q, want %q", output.RemovedTrackID, tt.wantID)
			}
		})
	}
}

func TestQueueService_Add(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name              string
		input             QueueAddInput
		setupRepo         func(*mockRepository)
		wantErr           error
		wantStartPosition int
		wantCount         int
		wantEvent         bool // expect CurrentTrackChangedEvent
	}{
		{
			name: "add multiple tracks to empty queue",
			input: QueueAddInput{
				GuildID:     guildID,
				TrackIDs:    []string{"track-1", "track-2", "track-3"},
				RequesterID: snowflake.ID(123),
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantStartPosition: 0,
			wantCount:         3,
			wantEvent:         true, // was idle, now has current
		},
		{
			name: "add multiple tracks to playing queue",
			input: QueueAddInput{
				GuildID:     guildID,
				TrackIDs:    []string{"new-1", "new-2"},
				RequesterID: snowflake.ID(123),
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("queued-1")})
			},
			wantStartPosition: 2, // after current + queued-1
			wantCount:         2,
			wantEvent:         false, // was not idle
		},
		{
			name: "add empty tracks slice",
			input: QueueAddInput{
				GuildID:     guildID,
				TrackIDs:    []string{},
				RequesterID: snowflake.ID(123),
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantStartPosition: 0,
			wantCount:         0,
			wantEvent:         false,
		},
		{
			name: "not connected",
			input: QueueAddInput{
				GuildID:     guildID,
				TrackIDs:    []string{"track-1"},
				RequesterID: snowflake.ID(123),
			},
			wantErr: ErrNotConnected,
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
			if tt.wantEvent {
				if len(publisher.events) != 1 {
					t.Fatalf("expected 1 event, got %d", len(publisher.events))
				}
				event, ok := publisher.events[0].(domain.CurrentTrackChangedEvent)
				if !ok {
					t.Fatalf("expected CurrentTrackChangedEvent, got %T", publisher.events[0])
				}
				if event.GuildID != tt.input.GuildID {
					t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
				}
			} else {
				if len(publisher.events) != 0 {
					t.Errorf("expected 0 events, got %d", len(publisher.events))
				}
			}
		})
	}
}

func TestQueueService_Add_TracksOrder(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	repo := newMockRepository()
	publisher := &mockEventPublisher{}

	state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
	state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
	state.SetPlaybackActive(true)

	service := NewQueueService(repo, publisher)
	_, err := service.Add(context.Background(), QueueAddInput{
		GuildID:     guildID,
		TrackIDs:    []string{"new-1", "new-2", "new-3"},
		RequesterID: snowflake.ID(123),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tracks are in correct order
	updatedState, _ := repo.Get(context.Background(), guildID)
	tracks := updatedState.List()
	if len(tracks) != 4 {
		t.Fatalf("expected 4 tracks in queue, got %d", len(tracks))
	}

	expectedOrder := []string{"current", "new-1", "new-2", "new-3"}
	for i, expected := range expectedOrder {
		if string(tracks[i].TrackID) != expected {
			t.Errorf("track[%d] = %q, want %q", i, tracks[i].TrackID, expected)
		}
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
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone)
				state.Advance(domain.LoopModeNone) // currentIndex=2
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
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("current")})
				state.SetPlaybackActive(true)
				// No other tracks
			},
			wantErr: ErrNothingToClear,
		},
		{
			name: "KeepCurrentTrack=true - at last track keeps it and clears rest",
			input: QueueClearInput{
				GuildID:          guildID,
				KeepCurrentTrack: true,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks, then advance to last track
				for i := range 3 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone) // index=1
				state.Advance(domain.LoopModeNone) // index=2 (last)
				state.SetPlaybackActive(true)
			},
			wantCount:     2, // 2 played tracks cleared (current kept)
			wantRemaining: 1, // current track remains
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
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone)
				state.Advance(domain.LoopModeNone)
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

			service := NewQueueService(repo, &mockEventPublisher{})
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
			if state.Len() != tt.wantRemaining {
				t.Errorf("remaining tracks = %d, want %d", state.Len(), tt.wantRemaining)
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
		setupRepo   func(*mockRepository)
		wantErr     error
		wantTrackID string
	}{
		{
			name: "restart idle queue after it ended",
			input: QueueRestartInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks and advance past end
				for _, id := range []string{"track-0", "track-1", "track-2"} {
					state.Append(domain.QueueEntry{TrackID: domain.TrackID(id)})
				}
				state.Advance(domain.LoopModeNone) // index=1
				state.Advance(domain.LoopModeNone) // index=2
				state.Advance(domain.LoopModeNone) // past end, idle
			},
			wantTrackID: "track-0",
		},
		{
			name: "restart while playing (in middle of queue)",
			input: QueueRestartInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks and advance to middle
				for _, id := range []string{"track-0", "track-1", "track-2"} {
					state.Append(domain.QueueEntry{TrackID: domain.TrackID(id)})
				}
				state.Advance(domain.LoopModeNone) // index=1
			},
			wantTrackID: "track-0",
		},
		{
			name: "empty queue",
			input: QueueRestartInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// No tracks
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "not connected",
			input: QueueRestartInput{
				GuildID: guildID,
			},
			wantErr: ErrNotConnected,
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

			if output.TrackID != tt.wantTrackID {
				t.Errorf("expected track ID %q, got %q", tt.wantTrackID, output.TrackID)
			}

			// Verify event was published
			if len(publisher.events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(publisher.events))
			}
			event, ok := publisher.events[0].(domain.CurrentTrackChangedEvent)
			if !ok {
				t.Fatalf("expected CurrentTrackChangedEvent, got %T", publisher.events[0])
			}
			if event.GuildID != tt.input.GuildID {
				t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
			}
			// Verify queue is at position 0 (Restart uses Seek(0))
			state, _ := repo.Get(context.Background(), guildID)
			if state.CurrentIndex() != 0 {
				t.Errorf(
					"expected currentIndex 0 after Restart, got %d",
					state.CurrentIndex(),
				)
			}
		})
	}
}

func TestQueueService_Seek(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name        string
		input       QueueSeekInput
		setupRepo   func(*mockRepository)
		wantErr     error
		wantTrackID string
	}{
		{
			name: "seek to middle of queue",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 2,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks (Append auto-activates, at index 0)
				for i := range 5 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
			},
			wantTrackID: "track-2",
		},
		{
			name: "seek to played track (before current)",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks and advance to index 2
				for i := range 5 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone)
				state.Advance(domain.LoopModeNone) // currentIndex=2
			},
			wantTrackID: "track-0",
		},
		{
			name: "seek to upcoming track (after current)",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 4,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 5 tracks (Append auto-activates, at index 0)
				for i := range 5 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
			},
			wantTrackID: "track-4",
		},
		{
			name: "seek to current position (restarts current)",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 3 tracks and advance to index 1
				for i := range 3 {
					state.Append(
						domain.QueueEntry{TrackID: domain.TrackID("track-" + string(rune('0'+i)))},
					)
				}
				state.Advance(domain.LoopModeNone) // currentIndex=1
			},
			wantTrackID: "track-1",
		},
		{
			name: "seek from idle state (queue ended)",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add 2 tracks and advance past end
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-0")})
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-1")})
				state.Advance(domain.LoopModeNone) // index=1
				state.Advance(domain.LoopModeNone) // past end, idle
			},
			wantTrackID: "track-1",
		},
		{
			name: "empty queue error",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 0,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// No tracks
			},
			wantErr: ErrQueueEmpty,
		},
		{
			name: "invalid position error - too high",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 10,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-0")})
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-1")})
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "invalid position error - negative",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: -1,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Append(domain.QueueEntry{TrackID: domain.TrackID("track-0")})
			},
			wantErr: ErrInvalidPosition,
		},
		{
			name: "not connected error",
			input: QueueSeekInput{
				GuildID:  guildID,
				Position: 0,
			},
			wantErr: ErrNotConnected,
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

			if output.TrackID != tt.wantTrackID {
				t.Errorf("expected track ID %q, got %q", tt.wantTrackID, output.TrackID)
			}

			// Verify event was published
			if len(publisher.events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(publisher.events))
			}
			event, ok := publisher.events[0].(domain.CurrentTrackChangedEvent)
			if !ok {
				t.Fatalf("expected CurrentTrackChangedEvent, got %T", publisher.events[0])
			}
			if event.GuildID != tt.input.GuildID {
				t.Errorf("event GuildID = %d, want %d", event.GuildID, tt.input.GuildID)
			}
			// Verify queue is at the seeked position and playback active
			state, _ := repo.Get(context.Background(), guildID)
			if state.CurrentIndex() != tt.input.Position {
				t.Errorf(
					"expected currentIndex %d after Seek, got %d",
					tt.input.Position,
					state.CurrentIndex(),
				)
			}
			if !state.IsPlaybackActive() {
				t.Error("expected playback to be active after Seek")
			}
		})
	}
}
