package events

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
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	playNextCh := make(chan snowflake.ID, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, guildID snowflake.ID) (*domain.Track, error) {
			playNextCh <- guildID
			return mockTrack("track-1"), nil
		},
		repo,
		bus,
	)

	handler.Start(t.Context())
	defer handler.Stop()

	// Publish event with wasIdle=true
	bus.PublishTrackEnqueued(TrackEnqueuedEvent{
		GuildID: snowflake.ID(1),
		Track:   mockTrack("track-1"),
		WasIdle: true,
	})

	// Wait for event processing
	select {
	case guildID := <-playNextCh:
		if guildID != snowflake.ID(1) {
			t.Errorf("expected guildID 1, got %d", guildID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track enqueued and idle")
	}
}

func TestPlaybackEventHandler_TrackEnqueued_WhenNotIdle_DoesNotStartPlayback(t *testing.T) {
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	playNextCh := make(chan struct{}, 1)

	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			playNextCh <- struct{}{}
			return mockTrack("track-1"), nil
		},
		repo,
		bus,
	)

	handler.Start(t.Context())
	defer handler.Stop()

	// Publish event with wasIdle=false
	bus.PublishTrackEnqueued(TrackEnqueuedEvent{
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

func TestPlaybackEventHandler_TrackEnded_Finished_AdvancesQueue(t *testing.T) {
	bus := NewBus(10)
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
		repo,
		bus,
	)

	handler.Start(t.Context())
	defer handler.Stop()

	// Publish track ended with "finished" reason
	bus.PublishTrackEnded(TrackEndedEvent{
		GuildID: guildID,
		Reason:  TrackEndFinished,
	})

	// Wait for event processing
	select {
	case <-playNextCh:
		// Success - playNextFunc was called
	case <-time.After(100 * time.Millisecond):
		t.Error("expected playNextFunc to be called when track finished")
	}
}

func TestPlaybackEventHandler_TrackEnded_Stopped_DoesNotAdvanceQueue(t *testing.T) {
	bus := NewBus(10)
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
		repo,
		bus,
	)

	handler.Start(t.Context())
	defer handler.Stop()

	// Publish track ended with "stopped" reason (user-initiated stop)
	bus.PublishTrackEnded(TrackEndedEvent{
		GuildID: guildID,
		Reason:  TrackEndStopped,
	})

	// Wait for event processing - should NOT be called
	select {
	case <-playNextCh:
		t.Error("expected playNextFunc NOT to be called when track stopped")
	case <-time.After(100 * time.Millisecond):
		// Success - playNextFunc was not called
	}
}

func TestPlaybackEventHandler_StopsOnContextCancellation(t *testing.T) {
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	handler := NewPlaybackEventHandler(
		func(_ context.Context, _ snowflake.ID) (*domain.Track, error) {
			return nil, nil
		},
		repo,
		bus,
	)

	ctx, cancel := context.WithCancel(context.Background())
	handler.Start(ctx)

	// Cancel context
	cancel()

	// Handler should stop gracefully
	done := make(chan struct{})
	go func() {
		handler.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("handler did not stop after context cancellation")
	}
}

// --- NotificationEventHandler Tests ---

func TestNotificationEventHandler_PlaybackStarted_SendsNowPlaying(t *testing.T) {
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	repo.Save(state)

	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start(t.Context())
	defer handler.Stop()

	track := mockTrack("track-1")
	bus.PublishPlaybackStarted(PlaybackStartedEvent{
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
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	guildID := snowflake.ID(1)
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	repo.Save(state)

	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start(t.Context())

	bus.PublishPlaybackStarted(PlaybackStartedEvent{
		GuildID:               guildID,
		Track:                 mockTrack("track-1"),
		NotificationChannelID: snowflake.ID(200),
	})

	// Wait for event processing and stop handler to ensure all events are processed
	time.Sleep(100 * time.Millisecond)
	handler.Stop()

	if state.GetNowPlayingMessage() == nil {
		t.Error("expected NowPlayingMessage to be set")
	}
}

func TestNotificationEventHandler_PlaybackFinished_DeletesMessage(t *testing.T) {
	bus := NewBus(10)
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

	handler.Start(t.Context())

	bus.PublishPlaybackFinished(PlaybackFinishedEvent{
		GuildID:               guildID,
		NotificationChannelID: snowflake.ID(200),
		LastMessageID:         &messageID,
	})

	// Wait for event processing and stop handler to ensure all events are processed
	time.Sleep(100 * time.Millisecond)
	handler.Stop()

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
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	handler.Start(t.Context())

	bus.PublishPlaybackFinished(PlaybackFinishedEvent{
		GuildID:               snowflake.ID(1),
		NotificationChannelID: snowflake.ID(200),
		LastMessageID:         nil,
	})

	// Wait for event processing and stop handler to ensure all events are processed
	time.Sleep(100 * time.Millisecond)
	handler.Stop()

	deletedMessages := notifier.getDeletedMessages()
	if len(deletedMessages) != 0 {
		t.Error("expected no messages to be deleted when LastMessageID is nil")
	}
}

func TestNotificationEventHandler_StopsOnContextCancellation(t *testing.T) {
	bus := NewBus(10)
	defer bus.Close()

	repo := newMockRepository()
	notifier := &mockNotifier{}

	handler := NewNotificationEventHandler(notifier, repo, bus)

	ctx, cancel := context.WithCancel(context.Background())
	handler.Start(ctx)

	// Cancel context
	cancel()

	// Handler should stop gracefully
	done := make(chan struct{})
	go func() {
		handler.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("handler did not stop after context cancellation")
	}
}
