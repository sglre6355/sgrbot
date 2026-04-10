package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestAddToQueueUsecase_Execute(t *testing.T) {
	track1 := newTestTrack("t1")
	track2 := newTestTrack("t2")

	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		input AddToQueueInput[string]
	}
	type want struct {
		startIndex int
		count      int
		played     int
		err        error
	}

	tests := []struct {
		name string
		deps deps
		args args
		want want
	}{
		{
			name: "not connected returns ErrNotConnected",
			deps: deps{
				state:   newIdleState,
				locator: func(_ domain.PlayerStateID) *stubPlayerStateLocator { return newStubLocatorNil() },
			},
			args: args{
				input: AddToQueueInput[string]{
					ConnectionInfo: "guild1",
					TrackURLs:      []string{"https://example.com/t1"},
					RequesterID:    "u1",
				},
			},
			want: want{err: ErrNotConnected},
		},
		{
			name: "empty track URLs returns ErrInvalidArgument",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("existing") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{
				input: AddToQueueInput[string]{
					ConnectionInfo: "guild1",
					TrackURLs:      []string{},
					RequesterID:    "u1",
				},
			},
			want: want{err: ErrInvalidArgument},
		},
		{
			name: "add to idle state starts playback",
			deps: deps{
				state: newIdleState,
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{
				input: AddToQueueInput[string]{
					ConnectionInfo: "guild1",
					TrackURLs:      []string{"https://example.com/t1"},
					RequesterID:    "u1",
				},
			},
			want: want{startIndex: 0, count: 1, played: 1},
		},
		{
			name: "add to active state does not start playback",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("existing") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{
				input: AddToQueueInput[string]{
					ConnectionInfo: "guild1",
					TrackURLs:      []string{"https://example.com/t2"},
					RequesterID:    "u1",
				},
			},
			want: want{startIndex: 1, count: 1, played: 0},
		},
		{
			name: "multiple tracks added at once",
			deps: deps{
				state: newIdleState,
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{
				input: AddToQueueInput[string]{
					ConnectionInfo: "guild1",
					TrackURLs:      []string{"https://example.com/t1", "https://example.com/t2"},
					RequesterID:    "u1",
				},
			},
			want: want{startIndex: 0, count: 2, played: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			locator := tt.deps.locator(state.ID())
			repo := newStubPlayerStateRepo(state)
			audio := &stubAudioGateway{}
			events := &stubEventPublisher{}

			uc := NewAddToQueueUsecase[string](
				newPlayerService(),
				repo,
				newStubTrackRepo(track1, track2),
				audio,
				events,
				locator,
			)

			out, err := uc.Execute(context.Background(), tt.args.input)

			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err: got %v, want %v", err, tt.want.err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.StartIndex != tt.want.startIndex {
				t.Errorf("StartIndex: got %d, want %d", out.StartIndex, tt.want.startIndex)
			}
			if out.Count != tt.want.count {
				t.Errorf("Count: got %d, want %d", out.Count, tt.want.count)
			}
			if len(audio.playedEntries) != tt.want.played {
				t.Errorf("Play calls: got %d, want %d", len(audio.playedEntries), tt.want.played)
			}
			if len(events.published) == 0 {
				t.Error("expected at least one event to be published")
			}
		})
	}
}
