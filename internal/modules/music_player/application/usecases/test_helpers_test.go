package usecases

import (
	"context"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func mockTrack(id string) *domain.Track {
	return &domain.Track{
		ID:          domain.TrackID(id),
		Encoded:     "encoded-" + id,
		Title:       "Track " + id,
		Artist:      "Artist",
		Duration:    3 * time.Minute,
		RequesterID: snowflake.ID(123),
	}
}

type mockRepository struct {
	states  map[snowflake.ID]*domain.PlayerState
	deleted []snowflake.ID
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

// createConnectedState creates a PlayerState with the given IDs and saves it to the mock repository.
// Returns the state for further modification (e.g., adding tracks).
func (m *mockRepository) createConnectedState(
	guildID, voiceChannelID, notificationChannelID snowflake.ID,
) *domain.PlayerState {
	state := domain.NewPlayerState(guildID, voiceChannelID, notificationChannelID)
	m.Save(state)
	return state
}

func (m *mockRepository) Delete(guildID snowflake.ID) {
	m.deleted = append(m.deleted, guildID)
	delete(m.states, guildID)
}

type mockAudioPlayer struct {
	playErr   error
	stopErr   error
	pauseErr  error
	resumeErr error
}

func (m *mockAudioPlayer) Play(_ context.Context, _ snowflake.ID, _ *domain.Track) error {
	return m.playErr
}

func (m *mockAudioPlayer) Stop(_ context.Context, _ snowflake.ID) error {
	return m.stopErr
}

func (m *mockAudioPlayer) Pause(_ context.Context, _ snowflake.ID) error {
	return m.pauseErr
}

func (m *mockAudioPlayer) Resume(_ context.Context, _ snowflake.ID) error {
	return m.resumeErr
}

type mockVoiceConnection struct {
	joinErr  error
	leaveErr error
}

func (m *mockVoiceConnection) JoinChannel(_ context.Context, _, _ snowflake.ID) error {
	return m.joinErr
}

func (m *mockVoiceConnection) LeaveChannel(_ context.Context, _ snowflake.ID) error {
	return m.leaveErr
}

type mockTrackResolver struct {
	loadErr    error
	loadResult *ports.LoadResult
}

func (m *mockTrackResolver) LoadTracks(_ context.Context, _ string) (*ports.LoadResult, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.loadResult, nil
}

type mockVoiceStateProvider struct {
	channels map[snowflake.ID]snowflake.ID // userID -> channelID
	err      error
}

func (m *mockVoiceStateProvider) GetUserVoiceChannel(
	guildID, userID snowflake.ID,
) (snowflake.ID, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.channels[userID], nil
}

type mockEventPublisher struct {
	trackEnqueued    []domain.TrackEnqueuedEvent
	playbackStarted  []domain.PlaybackStartedEvent
	playbackFinished []domain.PlaybackFinishedEvent
	trackEnded       []domain.TrackEndedEvent
	queueCleared     []domain.QueueClearedEvent
}

func (m *mockEventPublisher) PublishTrackEnqueued(event domain.TrackEnqueuedEvent) {
	m.trackEnqueued = append(m.trackEnqueued, event)
}

func (m *mockEventPublisher) PublishPlaybackStarted(event domain.PlaybackStartedEvent) {
	m.playbackStarted = append(m.playbackStarted, event)
}

func (m *mockEventPublisher) PublishPlaybackFinished(event domain.PlaybackFinishedEvent) {
	m.playbackFinished = append(m.playbackFinished, event)
}

func (m *mockEventPublisher) PublishTrackEnded(event domain.TrackEndedEvent) {
	m.trackEnded = append(m.trackEnded, event)
}

func (m *mockEventPublisher) PublishQueueCleared(event domain.QueueClearedEvent) {
	m.queueCleared = append(m.queueCleared, event)
}
