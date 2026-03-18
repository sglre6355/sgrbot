package usecases

import (
	"context"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// --- Test helpers ---

func newTestTrack(id string) domain.Track {
	return *domain.ConstructTrack(
		domain.TrackID(id), "Track "+id, "Author", time.Minute,
		"https://example.com/"+id, "", domain.TrackSourceYouTube, false,
	)
}

func newTestEntry(id string) domain.QueueEntry {
	return domain.ConstructQueueEntry(
		newTestTrack(id), domain.UserID("user1"), time.Now(), false,
	)
}

func newActiveState(ids ...string) domain.PlayerState {
	ps := domain.NewPlayerState()
	for _, id := range ids {
		ps.Append(newTestEntry(id))
	}
	return *ps
}

func newIdleState() domain.PlayerState {
	return *domain.NewPlayerState()
}

func newPlayerService() *domain.PlayerService {
	return domain.NewPlayerService(nil)
}

// --- Stub: PlayerStateRepository ---

type stubPlayerStateRepository struct {
	states  map[domain.PlayerStateID]domain.PlayerState
	saveErr error
	delErr  error
}

func newStubPlayerStateRepo(states ...domain.PlayerState) *stubPlayerStateRepository {
	m := make(map[domain.PlayerStateID]domain.PlayerState, len(states))
	for _, s := range states {
		m[s.ID()] = s
	}
	return &stubPlayerStateRepository{states: m}
}

func (r *stubPlayerStateRepository) FindByID(
	_ context.Context,
	id domain.PlayerStateID,
) (domain.PlayerState, error) {
	s, ok := r.states[id]
	if !ok {
		return domain.PlayerState{}, domain.ErrPlayerStateNotFound
	}
	return s, nil
}

func (r *stubPlayerStateRepository) Save(
	_ context.Context,
	state domain.PlayerState,
) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.states[state.ID()] = state
	return nil
}

func (r *stubPlayerStateRepository) Delete(
	_ context.Context,
	id domain.PlayerStateID,
) error {
	if r.delErr != nil {
		return r.delErr
	}
	delete(r.states, id)
	return nil
}

// --- Stub: TrackRepository ---

type stubTrackRepository struct {
	tracks map[domain.TrackID]domain.Track
}

func newStubTrackRepo(tracks ...domain.Track) *stubTrackRepository {
	m := make(map[domain.TrackID]domain.Track, len(tracks))
	for _, t := range tracks {
		m[t.ID()] = t
	}
	return &stubTrackRepository{tracks: m}
}

func (r *stubTrackRepository) FindByID(
	_ context.Context,
	id domain.TrackID,
) (domain.Track, error) {
	t, ok := r.tracks[id]
	if !ok {
		return domain.Track{}, domain.ErrEmptyTrackID
	}
	return t, nil
}

func (r *stubTrackRepository) FindByIDs(
	_ context.Context,
	ids ...domain.TrackID,
) ([]domain.Track, error) {
	result := make([]domain.Track, 0, len(ids))
	for _, id := range ids {
		t, ok := r.tracks[id]
		if !ok {
			return nil, domain.ErrEmptyTrackID
		}
		result = append(result, t)
	}
	return result, nil
}

// --- Stub: AudioGateway ---

type stubAudioGateway struct {
	playedEntries []domain.QueueEntry
	stopped       []domain.PlayerStateID
	paused        []domain.PlayerStateID
	resumed       []domain.PlayerStateID
	err           error
}

func (g *stubAudioGateway) Play(
	_ context.Context,
	_ domain.PlayerStateID,
	entry domain.QueueEntry,
) error {
	if g.err != nil {
		return g.err
	}
	g.playedEntries = append(g.playedEntries, entry)
	return nil
}

func (g *stubAudioGateway) Stop(
	_ context.Context,
	id domain.PlayerStateID,
) error {
	if g.err != nil {
		return g.err
	}
	g.stopped = append(g.stopped, id)
	return nil
}

func (g *stubAudioGateway) Pause(
	_ context.Context,
	id domain.PlayerStateID,
) error {
	if g.err != nil {
		return g.err
	}
	g.paused = append(g.paused, id)
	return nil
}

func (g *stubAudioGateway) Resume(
	_ context.Context,
	id domain.PlayerStateID,
) error {
	if g.err != nil {
		return g.err
	}
	g.resumed = append(g.resumed, id)
	return nil
}

// --- Stub: EventPublisher ---

type stubEventPublisher struct {
	published []domain.Event
}

func (p *stubEventPublisher) Publish(_ context.Context, events ...domain.Event) {
	p.published = append(p.published, events...)
}

// --- Stub: PlayerStateLocator ---

type stubPlayerStateLocator struct {
	mapping map[string]*domain.PlayerStateID
}

func newStubLocator(
	mappings map[string]domain.PlayerStateID,
) *stubPlayerStateLocator {
	m := make(map[string]*domain.PlayerStateID, len(mappings))
	for k, v := range mappings {
		v := v
		m[k] = &v
	}
	return &stubPlayerStateLocator{mapping: m}
}

func newStubLocatorNil() *stubPlayerStateLocator {
	return &stubPlayerStateLocator{mapping: map[string]*domain.PlayerStateID{}}
}

func (l *stubPlayerStateLocator) FindPlayerStateID(
	_ context.Context,
	connectionInfo string,
) *domain.PlayerStateID {
	return l.mapping[connectionInfo]
}

// --- Stub: VoiceConnectionGateway ---

type stubVoiceConnectionGateway struct {
	joined []domain.PlayerStateID
	left   []domain.PlayerStateID
	err    error
}

func (g *stubVoiceConnectionGateway) Join(
	_ context.Context,
	id domain.PlayerStateID,
	_ string,
) error {
	if g.err != nil {
		return g.err
	}
	g.joined = append(g.joined, id)
	return nil
}

func (g *stubVoiceConnectionGateway) Leave(
	_ context.Context,
	id domain.PlayerStateID,
) error {
	if g.err != nil {
		return g.err
	}
	g.left = append(g.left, id)
	return nil
}

// --- Stub: UserVoiceStateProvider ---

type stubUserVoiceStateProvider struct {
	info *string
	err  error
}

func (p *stubUserVoiceStateProvider) GetUserVoiceConnectionInfo(
	_ context.Context,
	_ string,
	_ domain.UserID,
) (*string, error) {
	return p.info, p.err
}

// --- Stub: TrackResolver ---

type stubTrackResolver struct {
	result domain.TrackList
	err    error
}

func (r *stubTrackResolver) ResolveQuery(
	_ context.Context,
	_ string,
) (domain.TrackList, error) {
	return r.result, r.err
}

// --- Stub: NowPlayingPublisher ---

type stubNowPlayingPublisher struct {
	shown   []domain.PlayerStateID
	cleared []domain.PlayerStateID
	err     error
}

func (p *stubNowPlayingPublisher) Show(
	id domain.PlayerStateID,
	_ domain.Track,
	_ domain.User,
	_ time.Time,
) error {
	if p.err != nil {
		return p.err
	}
	p.shown = append(p.shown, id)
	return nil
}

func (p *stubNowPlayingPublisher) Clear(id domain.PlayerStateID) error {
	if p.err != nil {
		return p.err
	}
	p.cleared = append(p.cleared, id)
	return nil
}

// --- Stub: NowPlayingDestinationSetter ---

type stubNowPlayingDestinationSetter struct {
	destinations map[domain.PlayerStateID]string
}

func (s *stubNowPlayingDestinationSetter) SetDestination(
	id domain.PlayerStateID,
	destination string,
) {
	if s.destinations == nil {
		s.destinations = make(map[domain.PlayerStateID]string)
	}
	s.destinations[id] = destination
}
