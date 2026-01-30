package application

import (
	"testing"
)

func TestPingInteractor_Execute(t *testing.T) {
	interactor := NewPingInteractor()

	result := interactor.Execute()

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.Message != "Pong!" {
		t.Errorf("expected message %q, got %q", "Pong!", result.Message)
	}
}

func TestPingInteractor_Execute_ReturnsNewResultEachTime(t *testing.T) {
	interactor := NewPingInteractor()

	result1 := interactor.Execute()
	result2 := interactor.Execute()

	if result1 == result2 {
		t.Error("expected different result instances")
	}
}
