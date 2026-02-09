package usecases

import (
	"context"
	"fmt"
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

func (m *mockRepository) Get(_ context.Context, guildID snowflake.ID) (domain.PlayerState, error) {
	state, ok := m.states[guildID]
	if !ok {
		return domain.PlayerState{}, fmt.Errorf("player state not found")
	}
	return *state, nil
}

func (m *mockRepository) Save(_ context.Context, state domain.PlayerState) error {
	m.states[state.GetGuildID()] = &state
	return nil
}

// createConnectedState creates a PlayerState with the given IDs and saves it to the mock repository.
// Returns the state pointer for further modification (e.g., adding tracks).
func (m *mockRepository) createConnectedState(
	guildID, voiceChannelID, notificationChannelID snowflake.ID,
) *domain.PlayerState {
	state := domain.NewPlayerState(guildID, voiceChannelID, notificationChannelID)
	m.states[guildID] = state
	return state
}

func (m *mockRepository) Delete(_ context.Context, guildID snowflake.ID) error {
	m.deleted = append(m.deleted, guildID)
	delete(m.states, guildID)
	return nil
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

type mockTrackProvider struct {
	tracks map[domain.TrackID]*domain.Track
}

func newMockTrackProvider() *mockTrackProvider {
	return &mockTrackProvider{
		tracks: make(map[domain.TrackID]*domain.Track),
	}
}

func (m *mockTrackProvider) LoadTrack(id domain.TrackID) (domain.Track, error) {
	t, ok := m.tracks[id]
	if !ok {
		return domain.Track{}, fmt.Errorf("track %q not found", id)
	}
	return *t, nil
}

func (m *mockTrackProvider) LoadTracks(ids ...domain.TrackID) ([]domain.Track, error) {
	result := make([]domain.Track, 0, len(ids))
	for _, id := range ids {
		t, ok := m.tracks[id]
		if !ok {
			return nil, fmt.Errorf("track %q not found", id)
		}
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockTrackProvider) Store(track *domain.Track) {
	m.tracks[track.ID] = track
}

// setupPlaying sets up a PlayerState with a track playing.
// It stores the track in the track provider, appends it to the queue, and activates playback.
func setupPlaying(state *domain.PlayerState, tp *mockTrackProvider, track *domain.Track) {
	tp.Store(track)
	state.Queue.Append(track.ID)
	state.SetPlaybackActive(true)
}
