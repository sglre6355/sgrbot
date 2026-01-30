package domain

import (
	"testing"
	"time"
)

func TestNewPingResult(t *testing.T) {
	result := NewPingResult()

	if result.Message != "Pong!" {
		t.Errorf("expected message %q, got %q", "Pong!", result.Message)
	}

	if result.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestPingResult_TimestampIsRecent(t *testing.T) {
	before := time.Now()
	result := NewPingResult()
	after := time.Now()

	if result.Timestamp.Before(before) || result.Timestamp.After(after) {
		t.Error("expected timestamp to be between before and after")
	}
}
