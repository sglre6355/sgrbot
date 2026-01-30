package application

import "github.com/sglre6355/sgrbot/internal/modules/test/domain"

// PongInteractor handles the pong use case.
type PongInteractor struct{}

// NewPongInteractor creates a new PongInteractor.
func NewPongInteractor() *PongInteractor {
	return &PongInteractor{}
}

// Execute evaluates the content and returns the pong result.
func (p *PongInteractor) Execute(content string) *domain.PongResult {
	return domain.NewPongResult(content)
}
