package bot

import "github.com/bwmarrin/discordgo"

// InteractionHandler handles a Discord interaction and returns a response.
type InteractionHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, r Responder) error

// EventHandler is a generic handler for any Discord event.
// It should be a function matching one of discordgo's handler signatures,
// e.g., func(s *discordgo.Session, m *discordgo.MessageCreate)
type EventHandler any

// ModuleDependencies provides dependencies that modules may need during initialization.
type ModuleDependencies struct {
	Session *discordgo.Session
}

// Module defines the interface that all bot modules must implement.
type Module interface {
	// Name returns the unique identifier for this module.
	Name() string

	// Commands returns the slash commands that this module provides.
	Commands() []*discordgo.ApplicationCommand

	// CommandHandlers returns a map of command names to their handlers.
	CommandHandlers() map[string]InteractionHandler

	// EventHandlers returns event handlers for this module.
	// Each handler should match a discordgo handler signature.
	EventHandlers() []EventHandler

	// Init initializes the module with the provided dependencies.
	Init(deps ModuleDependencies) error

	// Shutdown gracefully shuts down the module.
	Shutdown() error
}

// ConfigurableModule is an optional interface for modules that need configuration.
// Modules implementing this interface will have LoadConfig called before Init.
type ConfigurableModule interface {
	// LoadConfig loads and validates module-specific configuration.
	// Called before Init() and before Discord connection is established.
	// Should return an error if required configuration is missing or invalid.
	LoadConfig() error
}
