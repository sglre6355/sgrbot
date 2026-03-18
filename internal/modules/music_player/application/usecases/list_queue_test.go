package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestListQueueUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		page     int
		pageSize int
	}
	type want struct {
		totalTracks int
		currentPage int
		totalPages  int
		currentID   string
		loopMode    string
		err         error
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
			want: want{err: ErrNotConnected},
		},
		{
			name: "empty queue returns empty output",
			deps: deps{
				state: newIdleState,
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{totalTracks: 0, currentPage: 1, totalPages: 1},
		},
		{
			name: "active queue returns current track and metadata",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2", "3") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{
				totalTracks: 3,
				currentPage: 1,
				totalPages:  1,
				currentID:   "1",
				loopMode:    "none",
			},
		},
		{
			name: "respects custom page size",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2", "3", "4", "5") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{pageSize: 2},
			want: want{
				totalTracks: 5,
				currentPage: 1,
				totalPages:  3,
				currentID:   "1",
				loopMode:    "none",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()

			uc := NewListQueueUsecase[string](
				newStubPlayerStateRepo(state),
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), ListQueueInput[string]{
				ConnectionInfo: "guild1",
				Page:           tt.args.page,
				PageSize:       tt.args.pageSize,
			})

			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err: got %v, want %v", err, tt.want.err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.TotalTracks != tt.want.totalTracks {
				t.Errorf("TotalTracks: got %d, want %d", out.TotalTracks, tt.want.totalTracks)
			}
			if out.CurrentPage != tt.want.currentPage {
				t.Errorf("CurrentPage: got %d, want %d", out.CurrentPage, tt.want.currentPage)
			}
			if out.TotalPages != tt.want.totalPages {
				t.Errorf("TotalPages: got %d, want %d", out.TotalPages, tt.want.totalPages)
			}
			if tt.want.currentID != "" {
				if out.CurrentTrack == nil {
					t.Fatal("CurrentTrack: got nil, want non-nil")
				}
				if out.CurrentTrack.ID != tt.want.currentID {
					t.Errorf(
						"CurrentTrack.ID: got %q, want %q",
						out.CurrentTrack.ID,
						tt.want.currentID,
					)
				}
			}
			if tt.want.loopMode != "" && out.LoopMode != tt.want.loopMode {
				t.Errorf("LoopMode: got %q, want %q", out.LoopMode, tt.want.loopMode)
			}
		})
	}
}
