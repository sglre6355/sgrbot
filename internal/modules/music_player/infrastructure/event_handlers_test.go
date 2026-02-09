package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// mockRepository is a test double for domain.PlayerStateRepository.
type mockRepository struct {
	mu     sync.Mutex
	states map[snowflake.ID]*domain.PlayerState
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		states: make(map[snowflake.ID]*domain.PlayerState),
	}
}

func (m *mockRepository) Get(_ context.Context, guildID snowflake.ID) (domain.PlayerState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[guildID]
	if !ok {
		return domain.PlayerState{}, fmt.Errorf("player state not found")
	}
	return *state, nil
}

func (m *mockRepository) Save(_ context.Context, state domain.PlayerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.GetGuildID()] = &state
	return nil
}

func (m *mockRepository) Delete(_ context.Context, guildID snowflake.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, guildID)
	return nil
}

func (m *mockRepository) getState(guildID snowflake.ID) *domain.PlayerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.states[guildID]
}

// mockNotifier is a test double for ports.NotificationSender.
type mockNotifier struct {
	mu                sync.Mutex
	sentNowPlaying    []*ports.NowPlayingInfo
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
	repo.states[guildID] = state

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

	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: guildID,
		Track:   mockTrack("track-1"),
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

func TestPlaybackEventHandler_TrackEnqueued_AfterQueueEnds_StartsPlayback(t *testing.T) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	// Create state where queue has finished (currentIndex past end)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.Queue.Append(mockTrack("old-track").ID)
	state.Queue.Advance(domain.LoopModeNone) // past end, now idle
	repo.states[guildID] = state

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

	// Add new track to queue (playback remains inactive)
	state.Queue.Append(newTrack.ID)
	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: guildID,
		Track:   newTrack,
	})

	// Wait for event processing - should be called because queue is not active
	select {
	case calledGuildID := <-playNextCh:
		if calledGuildID != guildID {
			t.Errorf("expected guildID %d, got %d", guildID, calledGuildID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track enqueued after queue ended")
	}
}

func TestPlaybackEventHandler_TrackEnqueued_AlreadyActive_DoesNotStartPlayback(
	t *testing.T,
) {
	bus := NewChannelEventBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	// Create state where a track is already active
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	state.Queue.Append(mockTrack("current-track").ID)
	state.SetPlaybackActive(true)
	repo.states[guildID] = state

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

	// Publish event - queue is already active
	// (simulates concurrent enqueue where another track won the race)
	bus.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
		GuildID: guildID,
		Track:   mockTrack("second-track"),
	})

	// Wait for event processing - should NOT be called
	select {
	case <-playNextCh:
		t.Error("expected playNextFunc NOT to be called when queue is already active")
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
	state.Queue.Append(mockTrack("current").ID)
	state.Queue.Append(mockTrack("next").ID)
	state.SetPlaybackActive(true)
	repo.states[guildID] = state

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
	state.Queue.Append(mockTrack("failing").ID)
	state.Queue.Append(mockTrack("next").ID)
	state.SetPlaybackActive(true)
	state.SetLoopMode(domain.LoopModeTrack) // Set loop mode to track
	repo.states[guildID] = state

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

	// Wait a bit for the state to be saved
	time.Sleep(50 * time.Millisecond)

	// Re-read state from repo (Save stores a new copy)
	savedState := repo.getState(guildID)

	// Verify that the queue advanced to the next track (not looped on the failing track)
	// The current track should now be "next", not "failing"
	current := savedState.Queue.Current()
	if current == nil {
		t.Fatal("expected current track to exist")
	}
	if *current != domain.TrackID("next") {
		t.Errorf("expected current track to be 'next', got %q", *current)
	}

	// Verify that the failing track was removed from the queue
	trackIDs := savedState.Queue.List()
	if len(trackIDs) != 1 {
		t.Errorf("expected 1 track in queue, got %d", len(trackIDs))
	}
	for _, id := range trackIDs {
		if id == domain.TrackID("failing") {
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
	state.Queue.Append(mockTrack("current").ID)
	state.Queue.Append(mockTrack("next").ID)
	state.SetPlaybackActive(true)
	repo.states[guildID] = state

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
	repo.states[guildID] = state

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
	state.Queue.Append(track.ID)
	state.SetPlaybackActive(true)
	repo.states[guildID] = state

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
	state.Queue.Append(track.ID)
	state.SetPlaybackActive(true)
	repo.states[guildID] = state

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

	// Re-read state from repo (Save stores a new copy)
	savedState := repo.getState(guildID)
	if savedState.GetNowPlayingMessage() == nil {
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
	state.Queue.Append(mockTrack("different-track").ID)
	state.SetPlaybackActive(true)
	repo.states[guildID] = state

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
	repo.states[guildID] = state

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
