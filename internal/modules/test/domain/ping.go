package domain

import "time"

// PingResult represents the result of a ping operation.
type PingResult struct {
	Message   string
	Timestamp time.Time
}

// NewPingResult creates a new PingResult with the given message.
func NewPingResult() *PingResult {
	return &PingResult{
		Message:   "Pong!",
		Timestamp: time.Now(),
	}
}
