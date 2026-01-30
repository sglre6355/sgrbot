package application

import (
	"testing"
)

func TestPongInteractor_Execute_ShouldRespond(t *testing.T) {
	interactor := NewPongInteractor()

	result := interactor.Execute("Hello 🏓 world")

	if !result.ShouldRespond {
		t.Error("expected ShouldRespond to be true")
	}
	if result.Response != "Pong 🏓" {
		t.Errorf("expected response %q, got %q", "Pong 🏓", result.Response)
	}
}

func TestPongInteractor_Execute_ShouldNotRespond(t *testing.T) {
	interactor := NewPongInteractor()

	result := interactor.Execute("Hello world")

	if result.ShouldRespond {
		t.Error("expected ShouldRespond to be false")
	}
	if result.Response != "" {
		t.Errorf("expected empty response, got %q", result.Response)
	}
}
