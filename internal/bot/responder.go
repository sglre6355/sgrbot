package bot

import "github.com/bwmarrin/discordgo"

// Responder provides an abstraction for responding to Discord interactions.
// This interface enables testing handlers without a live Discord connection.
type Responder interface {
	// Respond sends a response to an interaction.
	Respond(response *discordgo.InteractionResponse) error
}

// DiscordResponder implements Responder using a live Discord session.
type DiscordResponder struct {
	session     *discordgo.Session
	interaction *discordgo.Interaction
}

// NewDiscordResponder creates a new DiscordResponder.
func NewDiscordResponder(s *discordgo.Session, i *discordgo.Interaction) *DiscordResponder {
	return &DiscordResponder{
		session:     s,
		interaction: i,
	}
}

// Respond sends a response to the interaction via Discord API.
func (r *DiscordResponder) Respond(response *discordgo.InteractionResponse) error {
	return r.session.InteractionRespond(r.interaction, response)
}

// MockResponder is a test double for Responder.
type MockResponder struct {
	LastResponse *discordgo.InteractionResponse
	Err          error
}

// Respond records the response for testing.
func (m *MockResponder) Respond(response *discordgo.InteractionResponse) error {
	m.LastResponse = response
	return m.Err
}
