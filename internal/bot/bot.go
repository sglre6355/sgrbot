package bot

import (
	"fmt"
	"log/slog"
	"maps"

	"github.com/bwmarrin/discordgo"
)

// Bot manages the Discord bot lifecycle and module coordination.
type Bot struct {
	config   *Config
	session  *discordgo.Session
	modules  []Module
	handlers map[string]InteractionHandler
}

// NewBot creates a new Bot instance with the given configuration.
func NewBot(cfg *Config) *Bot {
	return &Bot{
		config:   cfg,
		modules:  make([]Module, 0),
		handlers: make(map[string]InteractionHandler),
	}
}

// LoadModules loads modules from the global registry.
func (b *Bot) LoadModules() {
	b.modules = Modules()
}

// Start initializes the bot, connects to Discord, and registers commands.
func (b *Bot) Start() error {
	// Create Discord session
	session, err := discordgo.New("Bot " + b.config.DiscordToken)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}
	b.session = session

	// Initialize modules
	if err := b.initModules(); err != nil {
		return fmt.Errorf("failed to initialize modules: %w", err)
	}

	// Build handler map
	b.buildHandlerMap()

	// Register interaction handler
	b.session.AddHandler(b.handleInteraction)

	// Register module event handlers
	b.registerEventHandlers()

	// Open connection
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Register commands
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	slog.Info("started bot",
		"user_id", b.session.State.User.ID,
		"username", b.session.State.User.Username,
	)

	return nil
}

// Stop gracefully shuts down the bot.
func (b *Bot) Stop() error {
	// Shutdown modules
	for _, mod := range b.modules {
		if err := mod.Shutdown(); err != nil {
			slog.Warn("failed to shutdown module", "module", mod.Name(), "error", err)
		}
	}

	// Close Discord session
	if b.session != nil {
		return b.session.Close()
	}

	return nil
}

// initModules initializes all loaded modules.
func (b *Bot) initModules() error {
	deps := ModuleDependencies{
		Config: b.config,
	}

	for _, mod := range b.modules {
		if err := mod.Init(deps); err != nil {
			return fmt.Errorf("failed to initialize %s module: %w", mod.Name(), err)
		}
		slog.Debug("initialized module", "module", mod.Name())
	}

	moduleNames := make([]string, len(b.modules))
	for i, mod := range b.modules {
		moduleNames[i] = mod.Name()
	}
	slog.Info("initialized modules", "modules", moduleNames)

	return nil
}

// buildHandlerMap builds the command name to handler mapping.
func (b *Bot) buildHandlerMap() {
	for _, mod := range b.modules {
		maps.Copy(b.handlers, mod.CommandHandlers())
	}
}

// registerEventHandlers registers all module event handlers with the session.
func (b *Bot) registerEventHandlers() {
	for _, mod := range b.modules {
		for _, handler := range mod.EventHandlers() {
			b.session.AddHandler(handler)
		}
	}
}

// collectCommands gathers all commands from loaded modules.
func (b *Bot) collectCommands() []*discordgo.ApplicationCommand {
	var commands []*discordgo.ApplicationCommand
	for _, mod := range b.modules {
		commands = append(commands, mod.Commands()...)
	}
	return commands
}

// registerCommands registers all module commands with Discord.
func (b *Bot) registerCommands() error {
	commands := b.collectCommands()

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(
			b.session.State.User.ID,
			"", // Empty string registers commands globally
			cmd,
		)
		if err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name, err)
		}
		slog.Debug("registered command", "command", cmd.Name)
	}

	return nil
}

// Embed colors for responses.
const (
	colorYellow = 0xFFFF00
	colorRed    = 0xFF0000
)

// handleInteraction routes incoming interactions to the appropriate handler.
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	cmdName := i.ApplicationCommandData().Name
	handler, ok := b.handlers[cmdName]
	if !ok {
		slog.Warn("found no handler for command", "command", cmdName)
		b.respondWithEmbed(s, i, "Unknown Command", "This command is not recognized.", colorYellow)
		return
	}

	responder := NewDiscordResponder(s, i.Interaction)
	if err := handler(s, i, responder); err != nil {
		slog.Error("failed to handle command", "command", cmdName, "error", err)
		b.respondWithEmbed(s, i, "Error", "An error occurred while processing your command.",
			colorRed)
	}
}

// respondWithEmbed sends an embed response to an interaction.
func (b *Bot) respondWithEmbed(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	title, description string,
	color int,
) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       color,
				},
			},
		},
	})
	if err != nil {
		slog.Error("failed to send embed response", "error", err)
	}
}
