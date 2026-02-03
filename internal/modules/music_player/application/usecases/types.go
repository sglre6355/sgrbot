package usecases

import (
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Re-export domain types for presentation layer use.
// This allows presentation to depend only on usecases without importing domain directly.

// Track is an alias for domain.Track.
type Track = domain.Track

// TrackID is an alias for domain.TrackID.
type TrackID = domain.TrackID

// PlayerStateRepository is an alias for domain.PlayerStateRepository.
type PlayerStateRepository = domain.PlayerStateRepository
