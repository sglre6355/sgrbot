package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

func TestTrackLoaderService_LoadTrack(t *testing.T) {
	requesterID := snowflake.ID(123)

	successResult := &ports.LoadResult{
		Type: ports.LoadTypeTrack,
		Tracks: []*ports.TrackInfo{
			{
				Identifier: "dQw4w9WgXcQ",
				Encoded:    "encoded-data",
				Title:      "Test Song",
				Artist:     "Test Artist",
				Duration:   3 * time.Minute,
				URI:        "https://example.com/track",
				ArtworkURL: "https://example.com/art.jpg",
				SourceName: "youtube",
				IsStream:   false,
			},
		},
	}

	tests := []struct {
		name          string
		input         LoadTrackInput
		setupResolver func(*mockTrackResolver)
		wantErr       error
		wantTitle     string
	}{
		{
			name: "successful track loading with search query",
			input: LoadTrackInput{
				Query:       "test song",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = successResult
			},
			wantTitle: "Test Song",
		},
		{
			name: "successful track loading with URL",
			input: LoadTrackInput{
				Query:       "https://youtube.com/watch?v=123",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = successResult
			},
			wantTitle: "Test Song",
		},
		{
			name: "no results - empty type",
			input: LoadTrackInput{
				Query:       "nonexistent song",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeEmpty, Tracks: nil}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "no results - error type",
			input: LoadTrackInput{
				Query:       "error query",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeError, Tracks: nil}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "no results - empty tracks array",
			input: LoadTrackInput{
				Query:       "empty result",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{
					Type:   ports.LoadTypeSearch,
					Tracks: []*ports.TrackInfo{},
				}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "resolver error",
			input: LoadTrackInput{
				Query:       "test song",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadErr = errors.New("resolver connection failed")
			},
			wantErr: errors.New("resolver connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockTrackResolver{}

			if tt.setupResolver != nil {
				tt.setupResolver(resolver)
			}

			service := NewTrackLoaderService(resolver)
			output, err := service.LoadTrack(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.Track == nil {
				t.Error("expected Track to be set")
				return
			}

			if output.Track.Title != tt.wantTitle {
				t.Errorf("Track.Title = %q, want %q", output.Track.Title, tt.wantTitle)
			}

			if output.Track.RequesterID != requesterID {
				t.Errorf("Track.RequesterID = %d, want %d", output.Track.RequesterID, requesterID)
			}

			if output.Track.ID == "" {
				t.Error("expected Track.ID to be set")
			}
		})
	}
}
