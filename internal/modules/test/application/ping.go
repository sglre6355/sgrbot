package application

import "github.com/sglre6355/sgrbot/internal/modules/test/domain"

// PingInteractor handles the ping use case.
type PingInteractor struct{}

// NewPingInteractor creates a new PingInteractor.
func NewPingInteractor() *PingInteractor {
	return &PingInteractor{}
}

// Execute performs the ping operation and returns the result.
func (p *PingInteractor) Execute() *domain.PingResult {
	return domain.NewPingResult()
}
