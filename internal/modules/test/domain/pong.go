package domain

import "strings"

// PongResult represents the result of evaluating a pong trigger.
type PongResult struct {
	ShouldRespond bool
	Response      string
}

// NewPongResult evaluates the content and creates a PongResult.
func NewPongResult(content string) *PongResult {
	shouldRespond := strings.Contains(content, "🏓")

	response := ""
	if shouldRespond {
		response = "Pong 🏓"
	}

	return &PongResult{
		ShouldRespond: shouldRespond,
		Response:      response,
	}
}
