package infrastructure

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// mockRepository is a test double for domain.PlayerStateRepository.
type mockRepository struct {
	states map[snowflake.ID]*domain.PlayerState
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		states: make(map[snowflake.ID]*domain.PlayerState),
	}
}

func (m *mockRepository) Get(guildID snowflake.ID) *domain.PlayerState {
	return m.states[guildID]
}

func (m *mockRepository) Save(state *domain.PlayerState) {
	m.states[state.GuildID] = state
}

func (m *mockRepository) Delete(guildID snowflake.ID) {
	delete(m.states, guildID)
}

// mockNotifier is a test double for ports.NotificationSender.
type mockNotifier struct {
	mu                sync.Mutex
	sentNowPlaying    []*ports.NowPlayingInfo
	sentQueueAdded    []*ports.QueueAddedInfo
	sentErrors        []string
	deletedMessages   []snowflake.ID
	sendNowPlayingErr error
	deleteMessageErr  error
	lastMessageID     snowflake.ID
}

func (m *mockNotifier) SendNowPlaying(
	_ snowflake.ID,
	info *ports.NowPlayingInfo,
) (snowflake.ID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendNowPlayingErr != nil {
		return 0, m.sendNowPlayingErr
	}
	m.sentNowPlaying = append(m.sentNowPlaying, info)
	m.lastMessageID++
	return m.lastMessageID, nil
}

func (m *mockNotifier) DeleteMessage(_ snowflake.ID, messageID snowflake.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteMessageErr != nil {
		return m.deleteMessageErr
	}
	m.deletedMessages = append(m.deletedMessages, messageID)
	return nil
}

func (m *mockNotifier) SendQueueAdded(_ snowflake.ID, info *ports.QueueAddedInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentQueueAdded = append(m.sentQueueAdded, info)
	return nil
}

func (m *mockNotifier) SendError(_ snowflake.ID, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentErrors = append(m.sentErrors, message)
	return nil
}

// getSentNowPlaying returns a copy of sentNowPlaying for thread-safe access.
func (m *mockNotifier) getSentNowPlaying() []*ports.NowPlayingInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*ports.NowPlayingInfo, len(m.sentNowPlaying))
	copy(result, m.sentNowPlaying)
	return result
}

// getDeletedMessages returns a copy of deletedMessages for thread-safe access.
func (m *mockNotifier) getDeletedMessages() []snowflake.ID {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]snowflake.ID, len(m.deletedMessages))
	copy(result, m.deletedMessages)
	return result
}

func mockTrack(id string) *domain.Track {
	return &domain.Track{
		ID:          domain.TrackID(id),
		Title:       "Track " + id,
		Artist:      "Artist",
		Duration:    3 * time.Minute,
		RequesterID: snowflake.ID(123),
	}
}

// --- PlaybackEventHandler Tests ---

func TestPlaybackEventHandler_TrackEnqueued_WhenIdle_StartsPlayback(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	// Create state that is idle (no current track)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	repo.Save(state)

	playNextCh := make(chan snowflake.ID, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, calledGuildID snowflake.ID) (*domain.Track, error) {
			playNextCh <- calledGuildID
			return mockTrack("track-1"), nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish event with wasIdle=true
	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: guildID,
		Track:   mockTrack("track-1"),
		WasIdle: true,
	})

	// Wait for event processing
	select {
	case calledGuildID := <-playNextCh:
		if calledGuildID != guildID {
			t.Errorf("expected guildID %d, got %d", guildID, calledGuildID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track enqueued and idle")
	}
}

func TestPlaybackEventHandler_TrackEnqueued_WhenNotIdle_DoesNotStartPlayback(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	playNextCh := make(chan struct{}, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			playNextCh <- struct{}{}
			return mockTrack("track-1"), nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish event with wasIdle=false
	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: snowflake.ID(1),
		Track:   mockTrack("track-1"),
		WasIdle: false,
	})

	// Wait for event processing - should NOT be called
	select {
	case <-playNextCh:
		t.Error("expected playNextFunc NOT to be called when track enqueued but not idle")
	case <-time.After(100 * time.Millisecond):
		// Success - playNextFunc was not called
	}
}

func TestPlaybackEventHandler_TrackEnqueued_AfterQueueEnds_StartsPlayback(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	// Create state where queue has finished (currentIndex past end)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.Queue.Add(mockTrack("old-track"))
	state.Queue.Start()
	state.Queue.Advance(domain.LoopModeNone) // past end, now idle
	repo.Save(state)

	// Verify the state is idle (past end)
	if !state.IsIdle() {
		t.Fatal("expected state to be idle after queue ended")
	}

	playNextCh := make(chan snowflake.ID, 1)
	newTrack := mockTrack("new-track")

	handler := NewPlaybackEventHandler(
		func(_ context.Context, calledGuildID snowflake.ID) (*domain.Track, error) {
			playNextCh <- calledGuildID
			return newTrack, nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Add new track to queue (this is what Add() does)
	state.Queue.Add(newTrack)

	// Publish event with wasIdle=true (because Add() returned wasIdle=true)
	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: guildID,
		Track:   newTrack,
		WasIdle: true,
	})

	// Wait for event processing - should be called because track ID matches
	select {
	case calledGuildID := <-playNextCh:
		if calledGuildID != guildID {
			t.Errorf("expected guildID %d, got %d", guildID, calledGuildID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track enqueued after queue ended")
	}
}

func TestPlaybackEventHandler_TrackEnqueued_DifferentTrackCurrent_DoesNotStartPlayback(
	t *testing.T,
) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	// Create state where a different track is already current
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.SetPlaying(mockTrack("current-track"))
	repo.Save(state)

	playNextCh := make(chan struct{}, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			playNextCh <- struct{}{}
			return mockTrack("track-1"), nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish event with wasIdle=true but for a different track
	// (simulates concurrent enqueue where another track won the race)
	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: guildID,
		Track:   mockTrack("second-track"),
		WasIdle: true,
	})

	// Wait for event processing - should NOT be called
	select {
	case <-playNextCh:
		t.Error("expected playNextFunc NOT to be called when different track is current")
	case <-time.After(100 * time.Millisecond):
		// Success - playNextFunc was not called
	}
}

func TestPlaybackEventHandler_TrackEnded_Finished_AdvancesQueue(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.SetPlaying(mockTrack("current"))
	state.Queue.Add(mockTrack("next"))
	repo.Save(state)

	playNextCh := make(chan struct{}, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			playNextCh <- struct{}{}
			return mockTrack("next"), nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish track ended with "finished" reason
	bus.PublishTrackEnded(domain.TrackEndedEvent{
		GuildID: guildID,
		Reason:  domain.TrackEndFinished,
	})

	// Wait for event processing
	select {
	case <-playNextCh:
		// Success - playNextFunc was called
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track finished")
	}
}

func TestPlaybackEventHandler_TrackEnded_LoadFailed_WithLoopModeTrack_AdvancesToNextTrack(
	t *testing.T,
) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.SetPlaying(mockTrack("failing"))
	state.Queue.Add(mockTrack("next"))
	state.SetLoopMode(domain.LoopModeTrack) // Set loop mode to track
	repo.Save(state)

	playNextCh := make(chan struct{}, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			playNextCh <- struct{}{}
			return mockTrack("next"), nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish track ended with TrackEndLoadFailed reason
	bus.PublishTrackEnded(domain.TrackEndedEvent{
		GuildID: guildID,
		Reason:  domain.TrackEndLoadFailed,
	})

	// Wait for event processing
	select {
	case <-playNextCh:
		// Success - playNextFunc was called
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track load failed")
	}

	// Verify that the queue advanced to the next track (not looped on the failing track)
	// The current track should now be "next", not "failing"
	current := state.Queue.Current()
	if current == nil {
		t.Fatal("expected current track to exist")
	}
	if current.ID != domain.TrackID("next") {
		t.Errorf("expected current track to be 'next', got %q", current.ID)
	}

	// Verify that the failing track was removed from the queue
	tracks := state.Queue.List()
	if len(tracks) != 1 {
		t.Errorf("expected 1 track in queue, got %d", len(tracks))
	}
	for _, track := range tracks {
		if track.ID == domain.TrackID("failing") {
			t.Error("expected failing track to be removed from queue")
		}
	}
}

func TestPlaybackEventHandler_TrackEnded_Stopped_DoesNotAdvanceQueue(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.SetPlaying(mockTrack("current"))
	state.Queue.Add(mockTrack("next"))
	repo.Save(state)

	playNextCh := make(chan struct{}, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			playNextCh <- struct{}{}
			return mockTrack("next"), nil
		},
		func(_ context.Context, _ snowflake.ID) error { return nil },
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish track ended with "stopped" reason (user-initiated stop)
	bus.PublishTrackEnded(domain.TrackEndedEvent{
		GuildID: guildID,
		Reason:  domain.TrackEndStopped,
	})

	// Wait for event processing - should NOT be called
	select {
	case <-playNextCh:
		t.Error("expected playNextFunc NOT to be called when track stopped")
	case <-time.After(100 * time.Millisecond):
		// Success - playNextFunc was not called
	}
}

func TestPlaybackEventHandler_QueueCleared_StopsPlayback(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	channelID := snowflake.ID(200)
	messageID := snowflake.ID(999)

	state := domain.NewPlayerState(guildID, snowflake.ID(100), channelID)
	state.SetNowPlayingMessage(channelID, messageID)
	repo.Save(state)

	stopCh := make(chan snowflake.ID, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			return nil, nil
		},
		func(_ context.Context, guildID snowflake.ID) error {
			stopCh <- guildID
			return nil
		},
		repo,
		bus, // subscriber
		bus, // publisher
	)

	handler.Start()

	// Publish QueueCleared event
	bus.PublishQueueCleared(domain.QueueClearedEvent{
		GuildID:               guildID,
		NotificationChannelID: channelID,
	})

	// Wait for event processing
	select {
	case stoppedGuildID := <-stopCh:
		if stoppedGuildID != guildID {
			t.Errorf("expected guildID %d, got %d", guildID, stoppedGuildID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected stopFunc to be called when queue cleared")
	}

	// Also verify that PlaybackFinished was published (for message deletion)
	time.Sleep(50 * time.Millisecond)
	// Note: We can't directly verify PlaybackFinished from here as it goes to the bus,
	// but we've verified the stopFunc was called which is the critical part.
}

// --- NotificationEventHandler Tests ---

func TestNotificationEventHandler_PlaybackStarted_SendsNowPlaying(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	track := mockTrack("track-1")
	state.SetPlaying(track) // Set current track to match the event
	repo.Save(state)

	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start()

	bus.PublishPlaybackStarted(domain.PlaybackStartedEvent{
		GuildID:               guildID,
		Track:                 track,
		NotificationChannelID: snowflake.ID(200),
	})

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	sentNowPlaying := notifier.getSentNowPlaying()
	if len(sentNowPlaying) != 1 {
		t.Fatalf("expected 1 now playing notification, got %d", len(sentNowPlaying))
	}

	sent := sentNowPlaying[0]
	if sent.Title != track.Title {
		t.Errorf("expected title %q, got %q", track.Title, sent.Title)
	}
}

func TestNotificationEventHandler_PlaybackStarted_StoresMessageID(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	track := mockTrack("track-1")
	state.SetPlaying(track) // Set current track to match the event
	repo.Save(state)

	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start()

	bus.PublishPlaybackStarted(domain.PlaybackStartedEvent{
		GuildID:               guildID,
		Track:                 track,
		NotificationChannelID: snowflake.ID(200),
	})

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	if state.GetNowPlayingMessage() == nil {
		t.Error("expected NowPlayingMessage to be set")
	}
}

func TestNotificationEventHandler_PlaybackStarted_SkipsIfTrackNoLongerCurrent(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	// Set a different track as current (simulating the race condition where
	// the original track failed and was removed before this handler runs)
	state.SetPlaying(mockTrack("different-track"))
	repo.Save(state)

	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start()

	// Publish event for a track that is NOT the current track
	bus.PublishPlaybackStarted(domain.PlaybackStartedEvent{
		GuildID:               guildID,
		Track:                 mockTrack("failed-track"),
		NotificationChannelID: snowflake.ID(200),
	})

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	// Should NOT have sent a notification since the track is not current
	sentNowPlaying := notifier.getSentNowPlaying()
	if len(sentNowPlaying) != 0 {
		t.Errorf("expected 0 now playing notifications, got %d", len(sentNowPlaying))
	}
}

func TestNotificationEventHandler_PlaybackFinished_DeletesMessage(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	channelID := snowflake.ID(200)
	messageID := snowflake.ID(999)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), channelID)
	state.SetNowPlayingMessage(channelID, messageID)
	repo.Save(state)

	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start()

	bus.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
		GuildID:               guildID,
		NotificationChannelID: snowflake.ID(200),
		LastMessageID:         &messageID,
	})

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	deletedMessages := notifier.getDeletedMessages()
	if len(deletedMessages) != 1 {
		t.Fatalf("expected 1 deleted message, got %d", len(deletedMessages))
	}

	if deletedMessages[0] != messageID {
		t.Errorf(
			"expected message ID %d to be deleted, got %d",
			messageID,
			deletedMessages[0],
		)
	}
}

func TestNotificationEventHandler_PlaybackFinished_NilMessageID_DoesNotDelete(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start()

	bus.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
		GuildID:               snowflake.ID(1),
		NotificationChannelID: snowflake.ID(200),
		LastMessageID:         nil,
	})

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	deletedMessages := notifier.getDeletedMessages()
	if len(deletedMessages) != 0 {
		t.Error("expected no messages to be deleted when LastMessageID is nil")
	}
}
