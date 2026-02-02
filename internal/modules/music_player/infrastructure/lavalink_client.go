package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/events"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// voiceConnectionTimeout is the maximum time to wait for voice connection to be established.
const voiceConnectionTimeout = 10 * time.Second

// pendingVoiceConnection tracks the state of a pending voice connection.
type pendingVoiceConnection struct {
	mu             sync.Mutex
	hasVoiceState  bool
	hasVoiceServer bool
	ready          chan struct{}
}

// onEvent marks an event as received and signals ready if both events are present.
func (p *pendingVoiceConnection) onEvent(isVoiceState bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if isVoiceState {
		p.hasVoiceState = true
	} else {
		p.hasVoiceServer = true
	}

	if p.hasVoiceState && p.hasVoiceServer {
		select {
		case <-p.ready:
			// Already closed
		default:
			close(p.ready)
		}
	}
}

// LavalinkAdapter wraps DisGoLink to implement the port interfaces.
type LavalinkAdapter struct {
	link    disgolink.Client
	session *discordgo.Session
	botID   snowflake.ID

	pendingMu sync.Mutex
	pending   map[snowflake.ID]*pendingVoiceConnection

	bus *events.Bus
}

// LavalinkConfig contains Lavalink connection configuration.
type LavalinkConfig struct {
	Address  string
	Password string
}

// NewLavalinkAdapter creates a new LavalinkAdapter.
func NewLavalinkAdapter(
	session *discordgo.Session,
	config LavalinkConfig,
) (*LavalinkAdapter, error) {
	botID, err := snowflake.Parse(session.State.User.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bot ID: %w", err)
	}

	adapter := &LavalinkAdapter{
		session: session,
		botID:   botID,
		pending: make(map[snowflake.ID]*pendingVoiceConnection),
	}

	// Create DisGoLink client
	link := disgolink.New(botID,
		disgolink.WithListenerFunc(adapter.onTrackStart),
		disgolink.WithListenerFunc(adapter.onTrackEnd),
		disgolink.WithListenerFunc(adapter.onTrackException),
		disgolink.WithListenerFunc(adapter.onTrackStuck),
	)
	adapter.link = link

	// Add Lavalink node
	node, err := link.AddNode(context.Background(), disgolink.NodeConfig{
		Name:     "main",
		Address:  config.Address,
		Password: config.Password,
		Secure:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add Lavalink node: %w", err)
	}

	slog.Info("connected to Lavalink", "node", node.Config().Name, "address", config.Address)

	return adapter, nil
}

// Link returns the underlying DisGoLink client for event registration.
func (c *LavalinkAdapter) Link() disgolink.Client {
	return c.link
}

// JoinChannel connects to a voice channel.
// It waits for both VoiceStateUpdate and VoiceServerUpdate events before returning.
func (c *LavalinkAdapter) JoinChannel(ctx context.Context, guildID, channelID snowflake.ID) error {
	// Create pending connection tracker
	pending := &pendingVoiceConnection{
		ready: make(chan struct{}),
	}

	c.pendingMu.Lock()
	c.pending[guildID] = pending
	c.pendingMu.Unlock()

	// Cleanup pending entry when done
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, guildID)
		c.pendingMu.Unlock()
	}()

	// Use discordgo to update voice state
	err := c.session.ChannelVoiceJoinManual(guildID.String(), channelID.String(), false, false)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %w", err)
	}

	// Wait for voice connection to be established (both events received)
	select {
	case <-pending.ready:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for voice connection: %w", ctx.Err())
	case <-time.After(voiceConnectionTimeout):
		return fmt.Errorf("timeout waiting for voice connection")
	}
}

// LeaveChannel disconnects from the voice channel.
func (c *LavalinkAdapter) LeaveChannel(ctx context.Context, guildID snowflake.ID) error {
	// Destroy the player
	player := c.link.ExistingPlayer(guildID)
	if player != nil {
		if err := player.Destroy(ctx); err != nil {
			slog.Warn("failed to destroy player", "guild", guildID, "error", err)
		}
	}

	// Leave voice channel
	err := c.session.ChannelVoiceJoinManual(guildID.String(), "", false, false)
	if err != nil {
		return fmt.Errorf("failed to leave voice channel: %w", err)
	}
	return nil
}

// Play plays a track.
func (c *LavalinkAdapter) Play(
	ctx context.Context,
	guildID snowflake.ID,
	track *domain.Track,
) error {
	player := c.link.Player(guildID)

	// Use WithEncodedTrack to avoid userData:null issue
	if err := player.Update(ctx, lavalink.WithEncodedTrack(track.Encoded)); err != nil {
		return fmt.Errorf("failed to play track: %w", err)
	}

	return nil
}

// Stop stops the current playback.
func (c *LavalinkAdapter) Stop(ctx context.Context, guildID snowflake.ID) error {
	player := c.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithNullTrack()); err != nil {
		return fmt.Errorf("failed to stop playback: %w", err)
	}

	return nil
}

// Pause pauses the current playback.
func (c *LavalinkAdapter) Pause(ctx context.Context, guildID snowflake.ID) error {
	player := c.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithPaused(true)); err != nil {
		return fmt.Errorf("failed to pause playback: %w", err)
	}

	return nil
}

// Resume resumes the current playback.
func (c *LavalinkAdapter) Resume(ctx context.Context, guildID snowflake.ID) error {
	player := c.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithPaused(false)); err != nil {
		return fmt.Errorf("failed to resume playback: %w", err)
	}

	return nil
}

// LoadTracks loads tracks from Lavalink.
func (c *LavalinkAdapter) LoadTracks(
	ctx context.Context,
	query string,
) (*ports.LoadResult, error) {
	node := c.link.BestNode()
	if node == nil {
		return nil, fmt.Errorf("no available Lavalink node")
	}

	result, err := node.LoadTracks(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load tracks: %w", err)
	}

	return c.convertLoadResult(result), nil
}

// convertLoadResult converts Lavalink result to ports result.
func (c *LavalinkAdapter) convertLoadResult(result *lavalink.LoadResult) *ports.LoadResult {
	switch data := result.Data.(type) {
	case lavalink.Track:
		return &ports.LoadResult{
			Type:   ports.LoadTypeTrack,
			Tracks: []*ports.TrackInfo{c.convertTrack(data)},
		}

	case lavalink.Playlist:
		tracks := make([]*ports.TrackInfo, len(data.Tracks))
		for i, track := range data.Tracks {
			tracks[i] = c.convertTrack(track)
		}
		return &ports.LoadResult{
			Type:       ports.LoadTypePlaylist,
			Tracks:     tracks,
			PlaylistID: data.Info.Name,
		}

	case lavalink.Search:
		tracks := make([]*ports.TrackInfo, len(data))
		for i, track := range data {
			tracks[i] = c.convertTrack(track)
		}
		return &ports.LoadResult{
			Type:   ports.LoadTypeSearch,
			Tracks: tracks,
		}

	case lavalink.Empty:
		return &ports.LoadResult{
			Type: ports.LoadTypeEmpty,
		}

	case lavalink.Exception:
		return &ports.LoadResult{
			Type: ports.LoadTypeError,
		}

	default:
		return &ports.LoadResult{
			Type: ports.LoadTypeEmpty,
		}
	}
}

// convertTrack converts a Lavalink track to TrackInfo.
func (c *LavalinkAdapter) convertTrack(track lavalink.Track) *ports.TrackInfo {
	info := track.Info
	artworkURL := ""
	if info.ArtworkURL != nil {
		artworkURL = *info.ArtworkURL
	}

	return &ports.TrackInfo{
		Identifier: info.Identifier,
		Encoded:    track.Encoded,
		Title:      info.Title,
		Artist:     info.Author,
		Duration:   time.Duration(info.Length) * time.Millisecond,
		URI:        getStringPtr(info.URI),
		ArtworkURL: artworkURL,
		SourceName: info.SourceName,
		IsStream:   info.IsStream,
	}
}

func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// OnVoiceServerUpdate handles Discord voice server updates.
// This must be called from the Discord event handler.
func (c *LavalinkAdapter) OnVoiceServerUpdate(event *discordgo.VoiceServerUpdate) {
	guildID, err := snowflake.Parse(event.GuildID)
	if err != nil {
		slog.Error("failed to parse guild ID in voice server update", "error", err)
		return
	}

	c.link.OnVoiceServerUpdate(context.Background(), guildID, event.Token, event.Endpoint)

	// Signal that we received the voice server update
	c.pendingMu.Lock()
	pending := c.pending[guildID]
	c.pendingMu.Unlock()

	if pending != nil {
		pending.onEvent(false)
	}
}

// OnVoiceStateUpdate handles Discord voice state updates.
// This must be called from the Discord event handler.
func (c *LavalinkAdapter) OnVoiceStateUpdate(event *discordgo.VoiceStateUpdate) {
	// Only handle updates for the bot itself
	if event.UserID != c.botID.String() {
		return
	}

	guildID, err := snowflake.Parse(event.GuildID)
	if err != nil {
		slog.Error("failed to parse guild ID in voice state update", "error", err)
		return
	}

	sessionID := event.SessionID

	// Parse the channel ID - if empty, the bot is disconnecting
	var channelID *snowflake.ID
	if event.ChannelID != "" {
		id, err := snowflake.Parse(event.ChannelID)
		if err != nil {
			slog.Error("failed to parse channel ID in voice state update", "error", err)
			return
		}
		channelID = &id
	}

	c.link.OnVoiceStateUpdate(context.Background(), guildID, channelID, sessionID)

	// Signal that we received the voice state update (only for connections, not disconnections)
	if channelID != nil {
		c.pendingMu.Lock()
		pending := c.pending[guildID]
		c.pendingMu.Unlock()

		if pending != nil {
			pending.onEvent(true)
		}
	}
}

// SetEventBus sets the event bus for publishing Lavalink events.
func (c *LavalinkAdapter) SetEventBus(bus *events.Bus) {
	c.bus = bus
}

func (c *LavalinkAdapter) onTrackStart(player disgolink.Player, event lavalink.TrackStartEvent) {
	slog.Debug("track started", "guild", player.GuildID(), "track", event.Track.Info.Title)
}

func (c *LavalinkAdapter) onTrackEnd(player disgolink.Player, event lavalink.TrackEndEvent) {
	slog.Debug("track ended", "guild", player.GuildID(), "reason", event.Reason)

	if c.bus != nil {
		reason := convertEndReason(event.Reason)
		c.bus.Publish(events.TrackEndedEvent{
			GuildID: player.GuildID(),
			Reason:  reason,
		})
	}
}

func (c *LavalinkAdapter) onTrackException(
	player disgolink.Player,
	event lavalink.TrackExceptionEvent,
) {
	slog.Warn("track exception", "guild", player.GuildID(), "error", event.Exception.Message)
}

func (c *LavalinkAdapter) onTrackStuck(player disgolink.Player, event lavalink.TrackStuckEvent) {
	slog.Warn("track stuck", "guild", player.GuildID(), "threshold", event.Threshold)
}

func convertEndReason(reason lavalink.TrackEndReason) events.TrackEndReason {
	switch reason {
	case lavalink.TrackEndReasonFinished:
		return events.TrackEndFinished
	case lavalink.TrackEndReasonLoadFailed:
		return events.TrackEndLoadFailed
	case lavalink.TrackEndReasonStopped:
		return events.TrackEndStopped
	case lavalink.TrackEndReasonReplaced:
		return events.TrackEndReplaced
	case lavalink.TrackEndReasonCleanup:
		return events.TrackEndCleanup
	default:
		return events.TrackEndStopped
	}
}

// Ensure LavalinkAdapter implements port interfaces.
var (
	_ ports.AudioPlayer     = (*LavalinkAdapter)(nil)
	_ ports.VoiceConnection = (*LavalinkAdapter)(nil)
	_ ports.TrackResolver   = (*LavalinkAdapter)(nil)
)
