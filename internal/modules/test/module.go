package test

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/test/presentation"
)

func init() {
	bot.Register(&TestModule{})
}

// TestModule provides test commands like /ping.
type TestModule struct {
	pingHandler *presentation.PingHandler
	pongHandler *presentation.PongHandler
}

// Name returns the module name.
func (m *TestModule) Name() string {
	return "test"
}

// Commands returns the slash commands for this module.
func (m *TestModule) Commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Replies with Pong!",
		},
	}
}

// CommandHandlers returns the command handlers for this module.
func (m *TestModule) CommandHandlers() map[string]bot.InteractionHandler {
	return map[string]bot.InteractionHandler{
		"ping": m.pingHandler.Handle,
	}
}

// EventHandlers returns the event handlers for this module.
func (m *TestModule) EventHandlers() []bot.EventHandler {
	return []bot.EventHandler{
		m.pongHandler.HandleMessage,
	}
}

// Init initializes the module.
func (m *TestModule) Init(deps bot.ModuleDependencies) error {
	m.pingHandler = presentation.NewPingHandler()
	m.pongHandler = presentation.NewPongHandler()
	return nil
}

// Shutdown cleans up module resources.
func (m *TestModule) Shutdown() error {
	return nil
}
