package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestJoinVoiceChannelUsecase_Execute(t *testing.T) {
	type deps struct {
		state          func() *domain.PlayerState
		userVoiceState *stubUserVoiceStateProvider
		locator        func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		input JoinVoiceChannelInput[string, string]
	}
	type want struct {
		connectionInfo string
		joined         int
		savedStates    int
		err            error
	}

	connInfo := "channel1"
	resolvedInfo := "resolved-channel"

	tests := []struct {
		name string
		deps deps
		args args
		want want
	}{
		{
			name: "explicit connection info creates new state",
			deps: deps{
				state:          func() *domain.PlayerState { return nil },
				userVoiceState: &stubUserVoiceStateProvider{},
				locator:        func(_ domain.PlayerStateID) *stubPlayerStateLocator { return newStubLocatorNil() },
			},
			args: args{input: JoinVoiceChannelInput[string, string]{
				UserID: "u1", ConnectionInfo: &connInfo, PartialConnectionInfo: "guild1",
			}},
			want: want{connectionInfo: "channel1", joined: 1, savedStates: 1},
		},
		{
			name: "resolves user voice state when no explicit connection",
			deps: deps{
				state:          func() *domain.PlayerState { return nil },
				userVoiceState: &stubUserVoiceStateProvider{info: &resolvedInfo},
				locator:        func(_ domain.PlayerStateID) *stubPlayerStateLocator { return newStubLocatorNil() },
			},
			args: args{input: JoinVoiceChannelInput[string, string]{
				UserID: "u1", PartialConnectionInfo: "guild1",
			}},
			want: want{connectionInfo: "resolved-channel", joined: 1, savedStates: 1},
		},
		{
			name: "user not in voice returns ErrUserNotInVoice",
			deps: deps{
				state:          func() *domain.PlayerState { return nil },
				userVoiceState: &stubUserVoiceStateProvider{info: nil},
				locator:        func(_ domain.PlayerStateID) *stubPlayerStateLocator { return newStubLocatorNil() },
			},
			args: args{input: JoinVoiceChannelInput[string, string]{
				UserID: "u1", PartialConnectionInfo: "guild1",
			}},
			want: want{err: ErrUserNotInVoice},
		},
		{
			name: "existing player state reuses it",
			deps: deps{
				state: func() *domain.PlayerState {
					s := newActiveState("1")
					return &s
				},
				userVoiceState: &stubUserVoiceStateProvider{},
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{input: JoinVoiceChannelInput[string, string]{
				UserID: "u1", ConnectionInfo: &connInfo, PartialConnectionInfo: "guild1",
			}},
			want: want{connectionInfo: "channel1", joined: 1, savedStates: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repo *stubPlayerStateRepository
			var locator *stubPlayerStateLocator

			if existing := tt.deps.state(); existing != nil {
				repo = newStubPlayerStateRepo(*existing)
				locator = tt.deps.locator(existing.ID())
			} else {
				repo = newStubPlayerStateRepo()
				locator = tt.deps.locator(domain.PlayerStateID(""))
			}

			voice := &stubVoiceConnectionGateway{}

			uc := NewJoinVoiceChannelUsecase[string, string](
				repo,
				tt.deps.userVoiceState,
				locator,
				voice,
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
			if out.ConnectionInfo != tt.want.connectionInfo {
				t.Errorf(
					"ConnectionInfo: got %q, want %q",
					out.ConnectionInfo,
					tt.want.connectionInfo,
				)
			}
			if len(voice.joined) != tt.want.joined {
				t.Errorf("Join calls: got %d, want %d", len(voice.joined), tt.want.joined)
			}
			if len(repo.states) != tt.want.savedStates {
				t.Errorf("saved states: got %d, want %d", len(repo.states), tt.want.savedStates)
			}
		})
	}
}
