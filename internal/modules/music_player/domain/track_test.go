package domain

import (
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

func TestNewTrack(t *testing.T) {
	requesterID := snowflake.ID(123456789)
	track := NewTrack(
		"track-1",
		"encoded-data",
		"Test Song",
		"Test Artist",
		3*time.Minute+30*time.Second,
		"https://example.com/track",
		"https://example.com/artwork.jpg",
		"youtube",
		false,
		requesterID,
		"TestUser",
		"https://example.com/avatar.png",
	)

	if track.ID != "track-1" {
		t.Errorf("expected ID 'track-1', got %q", track.ID)
	}
	if track.Encoded != "encoded-data" {
		t.Errorf("expected Encoded 'encoded-data', got %q", track.Encoded)
	}
	if track.Title != "Test Song" {
		t.Errorf("expected Title 'Test Song', got %q", track.Title)
	}
	if track.Artist != "Test Artist" {
		t.Errorf("expected Artist 'Test Artist', got %q", track.Artist)
	}
	if track.Duration != 3*time.Minute+30*time.Second {
		t.Errorf("expected Duration 3m30s, got %v", track.Duration)
	}
	if track.URI != "https://example.com/track" {
		t.Errorf("expected URI 'https://example.com/track', got %q", track.URI)
	}
	if track.ArtworkURL != "https://example.com/artwork.jpg" {
		t.Errorf("expected ArtworkURL 'https://example.com/artwork.jpg', got %q", track.ArtworkURL)
	}
	if track.SourceName != "youtube" {
		t.Errorf("expected SourceName 'youtube', got %q", track.SourceName)
	}
	if track.IsStream {
		t.Error("expected IsStream false")
	}
	if track.RequesterID != requesterID {
		t.Errorf("expected RequesterID %d, got %d", requesterID, track.RequesterID)
	}
	if track.RequesterName != "TestUser" {
		t.Errorf("expected RequesterName 'TestUser', got %q", track.RequesterName)
	}
	if track.RequesterAvatarURL != "https://example.com/avatar.png" {
		t.Errorf(
			"expected RequesterAvatarURL 'https://example.com/avatar.png', got %q",
			track.RequesterAvatarURL,
		)
	}
	if track.EnqueuedAt.IsZero() {
		t.Error("expected EnqueuedAt to be set")
	}
}

func TestTrack_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		track    *Track
		expected bool
	}{
		{
			name: "valid track",
			track: &Track{
				Encoded: "encoded-data",
				Title:   "Test Song",
			},
			expected: true,
		},
		{
			name: "missing encoded",
			track: &Track{
				Encoded: "",
				Title:   "Test Song",
			},
			expected: false,
		},
		{
			name: "missing title",
			track: &Track{
				Encoded: "encoded-data",
				Title:   "",
			},
			expected: false,
		},
		{
			name: "empty track",
			track: &Track{
				Encoded: "",
				Title:   "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.track.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestTrack_FormattedDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		isStream bool
		expected string
	}{
		{
			name:     "stream",
			duration: 0,
			isStream: true,
			expected: "LIVE",
		},
		{
			name:     "zero duration",
			duration: 0,
			isStream: false,
			expected: "00:00",
		},
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			isStream: false,
			expected: "00:45",
		},
		{
			name:     "minutes and seconds",
			duration: 3*time.Minute + 30*time.Second,
			isStream: false,
			expected: "03:30",
		},
		{
			name:     "hours minutes seconds",
			duration: 1*time.Hour + 5*time.Minute + 30*time.Second,
			isStream: false,
			expected: "01:05:30",
		},
		{
			name:     "exact hour",
			duration: 1 * time.Hour,
			isStream: false,
			expected: "01:00:00",
		},
		{
			name:     "double digit all",
			duration: 12*time.Hour + 34*time.Minute + 56*time.Second,
			isStream: false,
			expected: "12:34:56",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := &Track{
				Duration: tt.duration,
				IsStream: tt.isStream,
			}
			if got := track.FormattedDuration(); got != tt.expected {
				t.Errorf("FormattedDuration() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
