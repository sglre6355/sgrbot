package ports

import (
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackProvider defines the interface for resolving TrackIDs to full Track objects.
type TrackProvider interface {
	// LoadTrack returns the Track for the given ID, or error if not found.
	LoadTrack(id domain.TrackID) (domain.Track, error)

	// LoadTracks returns Tracks for the given IDs, or error if any not found.
	LoadTracks(ids ...domain.TrackID) ([]domain.Track, error)
}
