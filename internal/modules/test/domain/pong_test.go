package domain

import (
	"testing"
)

func TestPongResult_ShouldRespond_True(t *testing.T) {
	result := NewPongResult("Hello 🏓 world")

	if !result.ShouldRespond {
		t.Error("expected ShouldRespond to be true for message containing 🏓")
	}
}

func TestPongResult_ShouldRespond_False(t *testing.T) {
	result := NewPongResult("Hello world")

	if result.ShouldRespond {
		t.Error("expected ShouldRespond to be false for message without 🏓")
	}
}

func TestPongResult_Response(t *testing.T) {
	result := NewPongResult("Hello 🏓 world")

	expected := "Pong 🏓"
	if result.Response != expected {
		t.Errorf("expected response %q, got %q", expected, result.Response)
	}
}

func TestPongResult_Response_WhenShouldNotRespond(t *testing.T) {
	result := NewPongResult("Hello world")

	if result.Response != "" {
		t.Errorf("expected empty response when ShouldRespond is false, got %q", result.Response)
	}
}
