package presentation

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/test/application"
)

// PingHandler handles the /ping command.
type PingHandler struct {
	interactor *application.PingInteractor
}

// NewPingHandler creates a new PingHandler.
func NewPingHandler() *PingHandler {
	return &PingHandler{
		interactor: application.NewPingInteractor(),
	}
}

// Handle processes the ping command and sends the response.
func (h *PingHandler) Handle(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	result := h.interactor.Execute()

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: result.Message,
		},
	})
}

// PongHandler handles messages containing the 🏓 emoji.
type PongHandler struct {
	interactor *application.PongInteractor
}

// NewPongHandler creates a new PongHandler.
func NewPongHandler() *PongHandler {
	return &PongHandler{
		interactor: application.NewPongInteractor(),
	}
}

// HandleMessage is the discordgo event handler for MessageCreate events.
func (h *PongHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	result := h.interactor.Execute(m.Content)
	if result.ShouldRespond {
		if _, err := s.ChannelMessageSend(m.ChannelID, result.Response); err != nil {
			slog.Error("failed to send message", "channel", m.ChannelID, "error", err)
		}
	}
}
