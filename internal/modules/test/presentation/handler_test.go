package presentation

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/bot"
)

func TestPingHandler_ReturnsMessage(t *testing.T) {
	handler := NewPingHandler()
	responder := &bot.MockResponder{}

	err := handler.Handle(nil, nil, responder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if responder.LastResponse == nil {
		t.Fatal("expected response, got nil")
	}

	if responder.LastResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("expected response type %d, got %d",
			discordgo.InteractionResponseChannelMessageWithSource,
			responder.LastResponse.Type)
	}

	data := responder.LastResponse.Data
	if data == nil {
		t.Fatal("expected response data, got nil")
	}

	if data.Content != "Pong!" {
		t.Errorf("expected content %q, got %q", "Pong!", data.Content)
	}
}

func TestPingHandler_ResponderError(t *testing.T) {
	handler := NewPingHandler()
	expectedErr := errors.New("responder failed")
	responder := &bot.MockResponder{Err: expectedErr}

	err := handler.Handle(nil, nil, responder)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}
