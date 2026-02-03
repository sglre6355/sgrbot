package music_player

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/events"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/presentation"
)

func init() {
	bot.Register(&MusicPlayerModule{})
}

// Compile-time interface checks.
var _ bot.ConfigurableModule = (*MusicPlayerModule)(nil)

// MusicPlayerModule provides music playback commands.
type MusicPlayerModule struct {
	config          *Config
	handlers        *presentation.Handlers
	autocomplete    *presentation.AutocompleteHandler
	eventHandlers   *presentation.EventHandlers
	lavalinkAdapter *infrastructure.LavalinkAdapter

	// Event-driven components
	eventBus            *events.Bus
	playbackHandler     *events.PlaybackEventHandler
	notificationHandler *events.NotificationEventHandler

	// Context for event handlers
	ctx    context.Context
	cancel context.CancelFunc
}

// Name returns the module name.
func (m *MusicPlayerModule) Name() string {
	return "music_player"
}

// Commands returns the slash commands for this module.
func (m *MusicPlayerModule) Commands() []*discordgo.ApplicationCommand {
	return presentation.Commands()
}

// CommandHandlers returns the command handlers for this module.
func (m *MusicPlayerModule) CommandHandlers() map[string]bot.InteractionHandler {
	return map[string]bot.InteractionHandler{
		"join":   m.handlers.HandleJoin,
		"leave":  m.handlers.HandleLeave,
		"play":   m.handlers.HandlePlay,
		"stop":   m.handlers.HandleStop,
		"pause":  m.handlers.HandlePause,
		"resume": m.handlers.HandleResume,
		"skip":   m.handlers.HandleSkip,
		"queue":  m.handlers.HandleQueue,
	}
}

// EventHandlers returns the event handlers for this module.
func (m *MusicPlayerModule) EventHandlers() []bot.EventHandler {
	return []bot.EventHandler{
		func(s *discordgo.Session, event *discordgo.VoiceServerUpdate) {
			m.handleVoiceServerUpdate(s, event)
		},
		func(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
			m.handleVoiceStateUpdate(s, event)
		},
		func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			m.handleInteractionCreate(s, i)
		},
	}
}

// LoadConfig loads module-specific configuration from environment variables.
func (m *MusicPlayerModule) LoadConfig() error {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return err
	}
	m.config = cfg
	return nil
}

// Init initializes the module.
func (m *MusicPlayerModule) Init(deps bot.ModuleDependencies) error {
	// Check if session is available
	if deps.Session == nil {
		slog.Warn("music_player module initialized without session, Lavalink integration disabled")
		return m.initWithoutLavalink()
	}

	return m.initWithLavalink(deps)
}

func (m *MusicPlayerModule) initWithoutLavalink() error {
	// Initialize with mock/no-op implementations for testing
	repo := infrastructure.NewMemoryRepository()

	// Create service with nil dependencies
	// These will fail at runtime if called, but allows the module to load
	queue := usecases.NewQueueService(repo, nil)
	autocomplete := usecases.NewAutocompleteService(repo, nil)

	m.handlers = presentation.NewHandlers(nil, nil, queue, nil)
	m.autocomplete = presentation.NewAutocompleteHandler(autocomplete)

	return nil
}

func (m *MusicPlayerModule) initWithLavalink(deps bot.ModuleDependencies) error {
	// Create cancellable context for event handlers
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Create Lavalink adapter
	lavalinkConfig := infrastructure.LavalinkConfig{
		Address:  m.config.LavalinkAddress,
		Password: m.config.LavalinkPassword,
	}

	lavalinkAdapter, err := infrastructure.NewLavalinkAdapter(deps.Session, lavalinkConfig)
	if err != nil {
		return err
	}
	m.lavalinkAdapter = lavalinkAdapter

	// Create event bus
	m.eventBus = events.NewBus(events.DefaultEventBufferSize)

	// Create infrastructure
	repo := infrastructure.NewMemoryRepository()
	voiceState := infrastructure.NewVoiceStateProvider(deps.Session)
	notifier := infrastructure.NewNotifier(deps.Session)

	// Create services with event bus
	voiceChannel := usecases.NewVoiceChannelService(repo, lavalinkAdapter, voiceState, m.eventBus)
	playback := usecases.NewPlaybackService(repo, lavalinkAdapter, voiceState, m.eventBus)
	queue := usecases.NewQueueService(repo, m.eventBus)
	trackLoader := usecases.NewTrackLoaderService(lavalinkAdapter)

	// Create event handlers
	m.playbackHandler = events.NewPlaybackEventHandler(playback.PlayNext, repo, m.eventBus)
	m.notificationHandler = events.NewNotificationEventHandler(notifier, repo, m.eventBus)

	// Start event handlers with cancellable context
	m.playbackHandler.Start(m.ctx)
	m.notificationHandler.Start(m.ctx)

	// Set event publisher on Lavalink adapter for event publishing
	lavalinkAdapter.SetEventPublisher(m.eventBus)

	// Create presentation handlers
	botID, err := snowflake.Parse(deps.Session.State.User.ID)
	if err != nil {
		return err
	}
	autocomplete := usecases.NewAutocompleteService(repo, lavalinkAdapter)
	m.handlers = presentation.NewHandlers(voiceChannel, playback, queue, trackLoader)
	m.autocomplete = presentation.NewAutocompleteHandler(autocomplete)
	m.eventHandlers = presentation.NewEventHandlers(botID, voiceChannel)

	slog.Info("music_player module initialized with Lavalink")

	return nil
}

// Shutdown cleans up module resources.
func (m *MusicPlayerModule) Shutdown() error {
	// Cancel context first to signal event handlers to stop
	if m.cancel != nil {
		m.cancel()
	}

	// Stop event handlers
	if m.playbackHandler != nil {
		m.playbackHandler.Stop()
	}
	if m.notificationHandler != nil {
		m.notificationHandler.Stop()
	}

	// Close event bus
	if m.eventBus != nil {
		m.eventBus.Close()
	}

	// Close Lavalink connection
	if m.lavalinkAdapter != nil {
		m.lavalinkAdapter.Link().Close()
	}

	return nil
}

// Event handlers.

func (m *MusicPlayerModule) handleVoiceServerUpdate(
	_ *discordgo.Session,
	event *discordgo.VoiceServerUpdate,
) {
	if m.lavalinkAdapter != nil {
		m.lavalinkAdapter.OnVoiceServerUpdate(event)
	}
}

func (m *MusicPlayerModule) handleVoiceStateUpdate(
	s *discordgo.Session,
	event *discordgo.VoiceStateUpdate,
) {
	if m.lavalinkAdapter != nil {
		m.lavalinkAdapter.OnVoiceStateUpdate(event)
	}
	if m.eventHandlers != nil {
		m.eventHandlers.HandleVoiceStateUpdate(s, event)
	}
}

func (m *MusicPlayerModule) handleInteractionCreate(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	if i.Type != discordgo.InteractionApplicationCommandAutocomplete {
		return
	}

	data := i.ApplicationCommandData()

	switch data.Name {
	case "play":
		m.autocomplete.HandlePlay(s, i)
	case "queue":
		if len(data.Options) > 0 && data.Options[0].Name == "remove" {
			m.autocomplete.HandleQueueRemove(s, i)
		}
	}
}
