package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func mockTrack(id string) *domain.Track {
	return &domain.Track{
		ID:       domain.TrackID(id),
		Title:    "Track " + id,
		Artist:   "Artist",
		Duration: 3 * time.Minute,
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
	state := domain.NewPlayerState(guildID, domain.NewQueue())
	state.SetVoiceChannelID(voiceChannelID)
	state.SetNotificationChannelID(notificationChannelID)
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

func (m *mockAudioPlayer) Play(_ context.Context, _ snowflake.ID, _ domain.TrackID) error {
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
	loadResult domain.TrackList
}

func (m *mockTrackResolver) LoadTrack(_ context.Context, _ domain.TrackID) (domain.Track, error) {
	return domain.Track{}, nil
}

func (m *mockTrackResolver) LoadTracks(
	_ context.Context,
	_ ...domain.TrackID,
) ([]domain.Track, error) {
	return nil, nil
}

func (m *mockTrackResolver) ResolveQuery(_ context.Context, _ string) (domain.TrackList, error) {
	if m.loadErr != nil {
		return domain.TrackList{}, m.loadErr
	}
	return m.loadResult, nil
}

type mockVoiceStateProvider struct {
	channels map[snowflake.ID]snowflake.ID // userID -> channelID
	err      error
}

func (m *mockVoiceStateProvider) GetUserVoiceChannel(
	_, userID snowflake.ID,
) (*snowflake.ID, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch, ok := m.channels[userID]
	if !ok {
		return nil, nil
	}
	return &ch, nil
}

type mockEventPublisher struct {
	events []domain.Event
}

func (m *mockEventPublisher) Publish(event domain.Event) error {
	m.events = append(m.events, event)
	return nil
}

type mockTrackProvider struct {
	tracks map[domain.TrackID]*domain.Track
}

func newMockTrackProvider() *mockTrackProvider {
	return &mockTrackProvider{
		tracks: make(map[domain.TrackID]*domain.Track),
	}
}

func (m *mockTrackProvider) LoadTrack(
	_ context.Context,
	id domain.TrackID,
) (domain.Track, error) {
	t, ok := m.tracks[id]
	if !ok {
		return domain.Track{}, fmt.Errorf("track %q not found", id)
	}
	return *t, nil
}

func (m *mockTrackProvider) LoadTracks(
	_ context.Context,
	ids ...domain.TrackID,
) ([]domain.Track, error) {
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

func (m *mockTrackProvider) ResolveQuery(
	_ context.Context,
	_ string,
) (domain.TrackList, error) {
	return domain.TrackList{}, nil
}

func (m *mockTrackProvider) Store(track *domain.Track) {
	m.tracks[track.ID] = track
}

type mockNotificationSender struct {
	deletedMessages []struct{ ChannelID, MessageID snowflake.ID }
}

func (m *mockNotificationSender) DeleteMessage(channelID, messageID snowflake.ID) error {
	m.deletedMessages = append(
		m.deletedMessages,
		struct{ ChannelID, MessageID snowflake.ID }{channelID, messageID},
	)
	return nil
}

func (m *mockNotificationSender) SendNowPlaying(
	_ snowflake.ID,
	_ snowflake.ID,
	_ domain.TrackID,
	_ snowflake.ID,
	_ time.Time,
) (snowflake.ID, error) {
	return 0, nil
}

func (m *mockNotificationSender) SendError(_ snowflake.ID, _ string) error {
	return nil
}

// setupPlaying sets up a PlayerState with a track playing.
// It stores the track in the track provider, appends it to the queue, and activates playback.
func setupPlaying(state *domain.PlayerState, tp *mockTrackProvider, track *domain.Track) {
	tp.Store(track)
	state.Append(domain.QueueEntry{TrackID: track.ID})
	state.SetPlaybackActive(true)
}
