package music_player

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/presentation/discord"
)

func init() {
	bot.Register(&MusicPlayerModule{})
}

// Compile-time interface checks.
var _ bot.ConfigurableModule = (*MusicPlayerModule)(nil)

// MusicPlayerModule provides music playback commands.
type MusicPlayerModule struct {
	config          *Config
	commandHandlers *discord.CommandHandlers
	autocomplete    *discord.AutocompleteHandler
	eventHandlers   *discord.EventHandlers
	lavalinkAdapter *infrastructure.LavalinkAdapter

	// Event-driven components
	eventBus            *infrastructure.ChannelEventBus
	playbackHandler     *application.PlaybackEventHandler
	notificationHandler *application.NotificationEventHandler

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
	return discord.Commands()
}

// CommandHandlers returns the command handlers for this module.
func (m *MusicPlayerModule) CommandHandlers() map[string]bot.InteractionHandler {
	return map[string]bot.InteractionHandler{
		"join":   m.commandHandlers.HandleJoin,
		"leave":  m.commandHandlers.HandleLeave,
		"play":   m.commandHandlers.HandlePlay,
		"stop":   m.commandHandlers.HandleStop,
		"pause":  m.commandHandlers.HandlePause,
		"resume": m.commandHandlers.HandleResume,
		"skip":   m.commandHandlers.HandleSkip,
		"queue":  m.commandHandlers.HandleQueue,
		"loop":   m.commandHandlers.HandleLoop,
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
	trackLoader := usecases.NewTrackLoaderService(nil)

	m.commandHandlers = discord.NewCommandHandlers(nil, nil, queue, nil, nil)
	m.autocomplete = discord.NewAutocompleteHandler(queue, trackLoader)

	return nil
}

func (m *MusicPlayerModule) initWithLavalink(deps bot.ModuleDependencies) error {
	// Create cancellable context for event handlers
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Create event bus (needed by Lavalink adapter for publishing events)
	m.eventBus = infrastructure.NewChannelEventBus(infrastructure.DefaultEventBufferSize)

	// Create Lavalink adapter
	lavalinkConfig := infrastructure.LavalinkConfig{
		Address:  m.config.LavalinkAddress,
		Password: m.config.LavalinkPassword,
	}

	lavalinkAdapter, err := infrastructure.NewLavalinkAdapter(
		deps.Session,
		m.eventBus,
		lavalinkConfig,
	)
	if err != nil {
		return err
	}
	m.lavalinkAdapter = lavalinkAdapter

	// Create infrastructure
	repo := infrastructure.NewMemoryRepository()
	voiceState := infrastructure.NewVoiceStateProvider(deps.Session)
	userInfoProv := infrastructure.NewDiscordUserInfoProvider(deps.Session)
	notifier := infrastructure.NewNotifier(deps.Session, lavalinkAdapter, userInfoProv)

	// Create services with event bus
	trackLoader := usecases.NewTrackLoaderService(lavalinkAdapter)
	voiceChannel := usecases.NewVoiceChannelService(
		repo,
		lavalinkAdapter,
		voiceState,
		m.eventBus,
		notifier,
	)
	playback := usecases.NewPlaybackService(
		repo,
		lavalinkAdapter,
		m.eventBus,
		notifier,
		lavalinkAdapter,
		voiceState,
	)
	queue := usecases.NewQueueService(repo, m.eventBus)

	notificationChannel := usecases.NewNotificationChannelService(repo)

	// Create application event handlers
	m.playbackHandler = application.NewPlaybackEventHandler(
		repo,
		lavalinkAdapter,
		m.eventBus,
		m.eventBus,
	)
	m.notificationHandler = application.NewNotificationEventHandler(
		repo,
		m.eventBus,
		notifier,
		userInfoProv,
	)

	// Register event handlers
	if err := m.playbackHandler.Start(); err != nil {
		return err
	}
	if err := m.notificationHandler.Start(); err != nil {
		return err
	}

	// Create presentation handlers
	botID, err := snowflake.Parse(deps.Session.State.User.ID)
	if err != nil {
		return err
	}
	m.commandHandlers = discord.NewCommandHandlers(
		voiceChannel,
		playback,
		queue,
		trackLoader,
		notificationChannel,
	)
	m.autocomplete = discord.NewAutocompleteHandler(queue, trackLoader)
	m.eventHandlers = discord.NewEventHandlers(botID, voiceChannel)

	slog.Info("music_player module initialized with Lavalink")

	return nil
}

// Shutdown cleans up module resources.
func (m *MusicPlayerModule) Shutdown() error {
	// Cancel context first to signal event handlers to stop
	if m.cancel != nil {
		m.cancel()
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
		if len(data.Options) > 0 {
			switch data.Options[0].Name {
			case "remove":
				m.autocomplete.HandleQueueRemove(s, i)
			case "seek":
				m.autocomplete.HandleQueueSeek(s, i)
			}
		}
	}
}
