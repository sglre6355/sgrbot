package domain

// LoopMode represents the loop mode for queue playback.
type LoopMode int

const (
	LoopModeNone  LoopMode = iota // Default: no looping
	LoopModeTrack                 // Repeat current track indefinitely
	LoopModeQueue                 // Repeat entire queue when reaching end
)

// String returns a human-readable representation of the loop mode.
func (m LoopMode) String() string {
	switch m {
	case LoopModeTrack:
		return "track"
	case LoopModeQueue:
		return "queue"
	default:
		return "none"
	}
}

// ParseLoopMode converts a string to domain.LoopMode.
func ParseLoopMode(s string) LoopMode {
	switch s {
	case "track":
		return LoopModeTrack
	case "queue":
		return LoopModeQueue
	default:
		return LoopModeNone
	}
}
