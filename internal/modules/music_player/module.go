package music_player

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure/discord"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure/discord/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure/in_memory"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure/youtube"
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
	commandHandlers *presentation.CommandHandlers
	autocomplete    *presentation.AutocompleteHandler
	eventHandlers   *presentation.EventHandlers

	voiceConnectionGateway *discord.DiscordVoiceConnectionGateway
	link                   disgolink.Client
	eventBus               *in_memory.ChannelEventBus

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
		"join":     m.commandHandlers.HandleJoin,
		"leave":    m.commandHandlers.HandleLeave,
		"play":     m.commandHandlers.HandlePlay,
		"stop":     m.commandHandlers.HandleStop,
		"pause":    m.commandHandlers.HandlePause,
		"resume":   m.commandHandlers.HandleResume,
		"skip":     m.commandHandlers.HandleSkip,
		"queue":    m.commandHandlers.HandleQueue,
		"loop":     m.commandHandlers.HandleLoop,
		"autoplay": m.commandHandlers.HandleAutoPlay,
	}
}

// EventHandlers returns the event handlers for this module.
func (m *MusicPlayerModule) EventHandlers() []bot.EventHandler {
	return []bot.EventHandler{
		func(_ *discordgo.Session, event *discordgo.VoiceServerUpdate) {
			m.voiceConnectionGateway.OnVoiceServerUpdate(event)
		},
		// Repository cleanup must run before the gateway drops the guild mapping on disconnect.
		m.eventHandlers.HandleVoiceStateUpdate,
		func(_ *discordgo.Session, event *discordgo.VoiceStateUpdate) {
			m.voiceConnectionGateway.OnVoiceStateUpdate(event)
		},
		m.autocomplete.HandleInteractionCreate,
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
	if deps.Session == nil {
		slog.Warn("music_player module initialized without session")
		return nil
	}

	return m.initWithLavalink(deps)
}

func (m *MusicPlayerModule) initWithLavalink(deps bot.ModuleDependencies) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	session := deps.Session

	botID, err := snowflake.Parse(session.State.User.ID)
	if err != nil {
		return err
	}

	// --- Infrastructure ---

	m.eventBus = in_memory.NewChannelEventBus(in_memory.DefaultEventBufferSize)

	// Create track cache (shared between track repository, resolver, and audio gateway)
	trackCache := lavalink.NewTrackCache()

	// Create audio gateway first (without the link client) so we can register its listeners
	// when creating the disgolink client.
	// Voice connection gateway is created first without the link, then we set it after.
	voiceConnectionGateway := discord.NewDiscordVoiceConnectionGateway(session, botID)
	m.voiceConnectionGateway = voiceConnectionGateway

	audioGateway := lavalink.NewLavalinkAudioGateway(
		nil,
		trackCache,
		voiceConnectionGateway,
		m.eventBus,
	)

	// Create disgolink client with audio gateway's listeners
	link := disgolink.New(botID, audioGateway.ListenerOpts()...)
	if _, err := link.AddNode(context.Background(), disgolink.NodeConfig{
		Name:     "main",
		Address:  m.config.LavalinkAddress,
		Password: m.config.LavalinkPassword,
		Secure:   false,
	}); err != nil {
		return fmt.Errorf("failed to add Lavalink node: %w", err)
	}
	m.link = link

	// Set the link on both gateways now that it exists
	voiceConnectionGateway.SetLink(link)
	audioGateway.SetLink(link)

	trackRepository := lavalink.NewLavalinkTrackRepository(link, trackCache)
	trackResolver := lavalink.NewLavalinkTrackResolver(link, trackCache)
	userVoiceStateProvider := discord.NewDiscordUserVoiceStateProvider(session)
	nowPlayingGateway := discord.NewDiscordNowPlayingGateway(session)
	userRepository := discord.NewDiscordUserRepository(session)
	playerStateRepository := in_memory.NewInMemoryPlayerStateRepository()
	trackRecommender := youtube.NewYouTubeTrackRecommender(trackResolver)

	// Domain services
	autoPlayService := domain.NewAutoPlayService(
		domain.UserID(session.State.User.ID),
		trackRecommender,
	)
	playerService := domain.NewPlayerService(autoPlayService)

	// --- Use cases ---

	addToQueue := usecases.NewAddToQueueUsecase(
		playerService,
		playerStateRepository,
		trackRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)
	clearQueue := usecases.NewClearQueueUsecase(
		playerService,
		playerStateRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)
	joinVoiceChannel := usecases.NewJoinVoiceChannelUsecase(
		playerStateRepository,
		userVoiceStateProvider,
		voiceConnectionGateway,
		voiceConnectionGateway,
	)
	leaveVoiceChannel := usecases.NewLeaveVoiceChannelUsecase(
		playerStateRepository,
		nowPlayingGateway,
		voiceConnectionGateway,
		voiceConnectionGateway,
	)
	listQueue := usecases.NewListQueueUsecase(
		playerStateRepository,
		voiceConnectionGateway,
	)
	pausePlayback := usecases.NewPausePlaybackUsecase(
		playerService,
		playerStateRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)
	removeFromQueue := usecases.NewRemoveFromQueueUsecase(
		playerService,
		playerStateRepository,
		m.eventBus,
		voiceConnectionGateway,
	)
	resolveQuery := usecases.NewResolveQueryUsecase(
		trackResolver,
	)
	restartQueue := usecases.NewRestartQueueUsecase(
		playerService,
		playerStateRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)
	resumePlayback := usecases.NewResumePlaybackUsecase(
		playerService,
		playerStateRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)
	seekQueue := usecases.NewSeekQueueUsecase(
		playerService,
		playerStateRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)
	setAutoPlay := usecases.NewSetAutoPlayUsecase(
		playerStateRepository,
		voiceConnectionGateway,
	)
	setLoopMode := usecases.NewSetLoopModeUsecase(
		playerStateRepository,
		voiceConnectionGateway,
	)
	setNowPlayingDestination := usecases.NewSetNowPlayingDestinationUsecase(
		nowPlayingGateway,
		voiceConnectionGateway,
	)
	shuffleQueue := usecases.NewShuffleQueueUsecase(
		playerService,
		playerStateRepository,
		m.eventBus,
		voiceConnectionGateway,
	)
	skipTrack := usecases.NewSkipTrackUsecase(
		playerService,
		playerStateRepository,
		audioGateway,
		m.eventBus,
		voiceConnectionGateway,
	)

	// --- Event subscriptions ---

	domainEventHandlers := application.NewDomainEventHandlers(
		playerService,
		playerStateRepository,
		userRepository,
		audioGateway,
		m.eventBus,
		nowPlayingGateway,
	)
	if err := errors.Join(
		m.eventBus.Subscribe(
			reflect.TypeFor[domain.TrackStartedEvent](),
			domainEventHandlers.HandleTrackStarted,
		),
		m.eventBus.Subscribe(
			reflect.TypeFor[domain.TrackEndedEvent](),
			domainEventHandlers.HandleTrackEnded,
		),
		m.eventBus.Subscribe(
			reflect.TypeFor[domain.QueueExhaustedEvent](),
			domainEventHandlers.HandleQueueExhausted,
		),
		m.eventBus.Subscribe(
			reflect.TypeFor[domain.PlaybackStoppedEvent](),
			domainEventHandlers.HandlePlaybackStopped,
		),
	); err != nil {
		return fmt.Errorf("failed to subscribe to domain events: %w", err)
	}

	// --- Presentation ---

	m.commandHandlers = presentation.NewCommandHandlers(
		addToQueue,
		clearQueue,
		joinVoiceChannel,
		leaveVoiceChannel,
		listQueue,
		pausePlayback,
		removeFromQueue,
		resolveQuery,
		restartQueue,
		resumePlayback,
		seekQueue,
		setAutoPlay,
		setLoopMode,
		setNowPlayingDestination,
		shuffleQueue,
		skipTrack,
	)

	m.autocomplete = presentation.NewAutocompleteHandler(
		listQueue,
		resolveQuery,
	)

	m.eventHandlers = presentation.NewEventHandlers(
		session.State.User.ID,
		leaveVoiceChannel,
	)

	slog.Info("music_player module initialized with Lavalink")

	return nil
}

// Shutdown cleans up module resources.
func (m *MusicPlayerModule) Shutdown() error {
	if m.cancel != nil {
		m.cancel()
	}

	if m.eventBus != nil {
		m.eventBus.Close()
	}

	if m.link != nil {
		m.link.Close()
	}

	return nil
}
