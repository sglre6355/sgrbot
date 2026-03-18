package in_memory

import (
	"context"
	"testing"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

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

func TestInMemoryPlayerStateRepository_FindByID(t *testing.T) {
	type args struct {
		setup func(t *testing.T, repo *InMemoryPlayerStateRepository) domain.PlayerStateID
	}
	type want struct {
		queueLen int
		err      error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "returns saved state",
			args: args{
				setup: func(t *testing.T, repo *InMemoryPlayerStateRepository) domain.PlayerStateID {
					t.Helper()
					ps := domain.NewPlayerState()
					ps.Append(newTestEntry("1"))
					if err := repo.Save(context.Background(), *ps); err != nil {
						t.Fatalf("setup Save: %v", err)
					}
					return ps.ID()
				},
			},
			want: want{queueLen: 1, err: nil},
		},
		{
			name: "not found returns error",
			args: args{
				setup: func(_ *testing.T, _ *InMemoryPlayerStateRepository) domain.PlayerStateID {
					return domain.NewPlayerStateID()
				},
			},
			want: want{err: domain.ErrPlayerStateNotFound},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryPlayerStateRepository()
			id := tt.args.setup(t, repo)

			found, err := repo.FindByID(context.Background(), id)

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err == nil && found.Len() != tt.want.queueLen {
				t.Errorf("Len: got %d, want %d", found.Len(), tt.want.queueLen)
			}
		})
	}
}

func TestInMemoryPlayerStateRepository_Delete(t *testing.T) {
	type args struct {
		setup func(t *testing.T, repo *InMemoryPlayerStateRepository) domain.PlayerStateID
	}
	type want struct {
		errAfterDelete error
		errOnDelete    error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "delete existing state",
			args: args{
				setup: func(t *testing.T, repo *InMemoryPlayerStateRepository) domain.PlayerStateID {
					t.Helper()
					ps := domain.NewPlayerState()
					if err := repo.Save(context.Background(), *ps); err != nil {
						t.Fatalf("setup Save: %v", err)
					}
					return ps.ID()
				},
			},
			want: want{errOnDelete: nil, errAfterDelete: domain.ErrPlayerStateNotFound},
		},
		{
			name: "delete non-existent does not error",
			args: args{
				setup: func(_ *testing.T, _ *InMemoryPlayerStateRepository) domain.PlayerStateID {
					return domain.NewPlayerStateID()
				},
			},
			want: want{errOnDelete: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryPlayerStateRepository()
			id := tt.args.setup(t, repo)

			err := repo.Delete(context.Background(), id)

			if err != tt.want.errOnDelete {
				t.Fatalf("Delete err: got %v, want %v", err, tt.want.errOnDelete)
			}
			if tt.want.errAfterDelete != nil {
				_, err := repo.FindByID(context.Background(), id)
				if err != tt.want.errAfterDelete {
					t.Fatalf("FindByID after delete: got %v, want %v", err, tt.want.errAfterDelete)
				}
			}
		})
	}
}

func TestInMemoryPlayerStateRepository_Count(t *testing.T) {
	type args struct {
		setup func(t *testing.T, repo *InMemoryPlayerStateRepository)
	}
	type want struct {
		count int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty repository",
			args: args{setup: func(_ *testing.T, _ *InMemoryPlayerStateRepository) {}},
			want: want{count: 0},
		},
		{
			name: "two saved states",
			args: args{setup: func(t *testing.T, repo *InMemoryPlayerStateRepository) {
				t.Helper()
				if err := repo.Save(context.Background(), *domain.NewPlayerState()); err != nil {
					t.Fatalf("setup Save: %v", err)
				}
				if err := repo.Save(context.Background(), *domain.NewPlayerState()); err != nil {
					t.Fatalf("setup Save: %v", err)
				}
			}},
			want: want{count: 2},
		},
		{
			name: "save then delete",
			args: args{setup: func(t *testing.T, repo *InMemoryPlayerStateRepository) {
				t.Helper()
				ps := domain.NewPlayerState()
				if err := repo.Save(context.Background(), *ps); err != nil {
					t.Fatalf("setup Save: %v", err)
				}
				if err := repo.Save(context.Background(), *domain.NewPlayerState()); err != nil {
					t.Fatalf("setup Save: %v", err)
				}
				if err := repo.Delete(context.Background(), ps.ID()); err != nil {
					t.Fatalf("setup Delete: %v", err)
				}
			}},
			want: want{count: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryPlayerStateRepository()
			tt.args.setup(t, repo)

			if got := repo.Count(); got != tt.want.count {
				t.Fatalf("Count: got %d, want %d", got, tt.want.count)
			}
		})
	}
}

func TestInMemoryPlayerStateRepository_SaveOverwrites(t *testing.T) {
	type args struct {
		entriesOnSecondSave int
	}
	type want struct {
		queueLen int
		count    int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "second save overwrites first",
			args: args{entriesOnSecondSave: 2},
			want: want{queueLen: 2, count: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryPlayerStateRepository()
			ctx := context.Background()

			ps := domain.NewPlayerState()
			if err := repo.Save(ctx, *ps); err != nil {
				t.Fatalf("first Save: %v", err)
			}

			for i := range tt.args.entriesOnSecondSave {
				ps.Append(newTestEntry(string(rune('A' + i))))
			}
			if err := repo.Save(ctx, *ps); err != nil {
				t.Fatalf("second Save: %v", err)
			}

			found, err := repo.FindByID(ctx, ps.ID())
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if found.Len() != tt.want.queueLen {
				t.Errorf("Len: got %d, want %d", found.Len(), tt.want.queueLen)
			}
			if repo.Count() != tt.want.count {
				t.Errorf("Count: got %d, want %d", repo.Count(), tt.want.count)
			}
		})
	}
}
