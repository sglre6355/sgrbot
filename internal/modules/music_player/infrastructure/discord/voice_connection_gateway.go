package discord

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// voiceConnectionTimeout is the maximum time to wait for voice connection to be established.
const voiceConnectionTimeout = 1 * time.Second

// Ensure DiscordVoiceConnectionGateway implements required interfaces.
var (
	_ ports.VoiceConnectionGateway[discord.VoiceConnectionInfo] = (*DiscordVoiceConnectionGateway)(
		nil,
	)
)

// pendingVoiceConnectionManager tracks the state of a pending voice connection.
type pendingVoiceConnectionManager struct {
	mu             sync.Mutex
	hasVoiceState  bool
	hasVoiceServer bool
	ready          chan struct{}
}

// onEvent marks an event as received and signals ready if both events are present.
func (p *pendingVoiceConnectionManager) onEvent(isVoiceState bool) {
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

// voiceEventBuffer buffers voice events to ensure both VoiceStateUpdate and
// VoiceServerUpdate are received before forwarding to Lavalink.
// This prevents "Partial Lavalink voice state" errors when events arrive out of order.
type voiceEventBuffer struct {
	mu sync.Mutex

	// From VoiceStateUpdate
	hasVoiceState bool
	channelID     *snowflake.ID
	sessionID     string

	// From VoiceServerUpdate
	hasVoiceServer bool
	token          string
	endpoint       string
}

// setVoiceState stores voice state data and returns true if both events are now ready.
func (b *voiceEventBuffer) setVoiceState(channelID *snowflake.ID, sessionID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.hasVoiceState = true
	b.channelID = channelID
	b.sessionID = sessionID

	return b.hasVoiceState && b.hasVoiceServer
}

// setVoiceServer stores voice server data and returns true if both events are now ready.
func (b *voiceEventBuffer) setVoiceServer(token, endpoint string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.hasVoiceServer = true
	b.token = token
	b.endpoint = endpoint

	return b.hasVoiceState && b.hasVoiceServer
}

// getData returns the buffered data.
// The buffer retains values so that a subsequent lone VoiceServerUpdate
// (e.g. Discord voice server rotation) can be forwarded immediately
// using the existing session info. Cleanup is handled by clearVoiceBuffer
// on disconnect.
func (b *voiceEventBuffer) getData() (channelID *snowflake.ID, sessionID, token, endpoint string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.channelID, b.sessionID, b.token, b.endpoint
}

func (b *voiceEventBuffer) hasVoiceServerData() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.hasVoiceServer
}

// DiscordVoiceConnectionGateway manages voice connections via Discord and Lavalink.
// It owns the guild <-> PlayerState bidirectional mapping and voice event buffering.
type DiscordVoiceConnectionGateway struct {
	session *discordgo.Session
	link    disgolink.Client
	botID   snowflake.ID

	stateMu      sync.RWMutex
	stateToGuild map[core.PlayerStateID]snowflake.ID
	guildToState map[snowflake.ID]core.PlayerStateID

	pendingMu sync.Mutex
	pending   map[snowflake.ID]*pendingVoiceConnectionManager

	voiceBufferMu sync.Mutex
	voiceBuffers  map[snowflake.ID]*voiceEventBuffer
}

// NewDiscordVoiceConnectionGateway creates a new DiscordVoiceConnectionGateway.
// The link client can be set later via SetLink if not available at construction time.
func NewDiscordVoiceConnectionGateway(
	session *discordgo.Session,
	botID snowflake.ID,
) *DiscordVoiceConnectionGateway {
	return &DiscordVoiceConnectionGateway{
		session:      session,
		botID:        botID,
		stateToGuild: make(map[core.PlayerStateID]snowflake.ID),
		guildToState: make(map[snowflake.ID]core.PlayerStateID),
		pending:      make(map[snowflake.ID]*pendingVoiceConnectionManager),
		voiceBuffers: make(map[snowflake.ID]*voiceEventBuffer),
	}
}

// SetLink sets the disgolink client. This is used to break the circular dependency
// between the voice connection gateway and the audio gateway.
func (c *DiscordVoiceConnectionGateway) SetLink(link disgolink.Client) {
	c.link = link
}

// ResolveGuildID looks up the GuildID for a PlayerStateID.
func (c *DiscordVoiceConnectionGateway) ResolveGuildID(
	id core.PlayerStateID,
) (snowflake.ID, error) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	guildID, ok := c.stateToGuild[id]
	if !ok {
		return 0, fmt.Errorf("no guild mapping for player state %s", id)
	}
	return guildID, nil
}

// ResolvePlayerStateID looks up the PlayerStateID for a GuildID.
func (c *DiscordVoiceConnectionGateway) ResolvePlayerStateID(
	guildID snowflake.ID,
) (core.PlayerStateID, bool) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	id, ok := c.guildToState[guildID]
	return id, ok
}

// --- ports.VoiceConnectionGateway ---

// FindPlayerStateID returns the player state ID associated with the given connection info.
func (c *DiscordVoiceConnectionGateway) FindPlayerStateID(
	_ context.Context,
	info discord.VoiceConnectionInfo,
) *core.PlayerStateID {
	guildID, err := snowflake.Parse(info.GuildID)
	if err != nil {
		return nil
	}

	playerStateID, ok := c.ResolvePlayerStateID(guildID)
	if !ok {
		return nil
	}
	if info.ChannelID != "" && !c.isConnectedToChannel(guildID, info.ChannelID) {
		return nil
	}

	return &playerStateID
}

// Join connects to a voice channel and registers the player state mapping.
func (c *DiscordVoiceConnectionGateway) Join(
	ctx context.Context,
	playerStateID core.PlayerStateID,
	info discord.VoiceConnectionInfo,
) error {
	guildID, err := snowflake.Parse(info.GuildID)
	if err != nil {
		return fmt.Errorf("invalid guild ID %q: %w", info.GuildID, err)
	}
	channelID, err := snowflake.Parse(info.ChannelID)
	if err != nil {
		return fmt.Errorf("invalid channel ID %q: %w", info.ChannelID, err)
	}

	buffer := c.getOrCreateVoiceBuffer(guildID)

	// Create pending connection tracker
	pending := &pendingVoiceConnectionManager{
		hasVoiceServer: buffer.hasVoiceServerData(),
		ready:          make(chan struct{}),
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
	err = c.session.ChannelVoiceJoinManual(guildID.String(), channelID.String(), false, false)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %w", err)
	}

	// Wait for voice connection to be established (both events received)
	select {
	case <-pending.ready:
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for voice connection: %w", ctx.Err())
	case <-time.After(voiceConnectionTimeout):
		return fmt.Errorf("timeout waiting for voice connection")
	}

	// Register mapping
	c.registerMapping(guildID, playerStateID)

	return nil
}

// Leave disconnects from the voice channel and removes the player state mapping.
func (c *DiscordVoiceConnectionGateway) Leave(
	ctx context.Context,
	playerStateID core.PlayerStateID,
) error {
	guildID, err := c.ResolveGuildID(playerStateID)
	if err != nil {
		return nil
	}

	// Destroy the player
	player := c.link.ExistingPlayer(guildID)
	if player != nil {
		if err := player.Destroy(ctx); err != nil {
			slog.Warn("failed to destroy player", "guild", guildID, "error", err)
		}
	}

	// Leave voice channel
	if err := c.session.ChannelVoiceJoinManual(guildID.String(), "", false, false); err != nil {
		return fmt.Errorf("failed to leave voice channel: %w", err)
	}

	c.unregisterGuild(guildID)

	return nil
}

// --- Discord voice event handlers ---

// OnVoiceServerUpdate handles Discord voice server updates.
// This must be called from the Discord event handler.
func (c *DiscordVoiceConnectionGateway) OnVoiceServerUpdate(event *discordgo.VoiceServerUpdate) {
	guildID, err := snowflake.Parse(event.GuildID)
	if err != nil {
		slog.Error("failed to parse guild ID in voice server update", "error", err)
		return
	}

	// Get or create voice buffer for this guild
	buffer := c.getOrCreateVoiceBuffer(guildID)

	// Store voice server data and check if both events are ready
	if buffer.setVoiceServer(event.Token, event.Endpoint) {
		// Both events received, forward to Lavalink
		c.forwardBufferedVoiceEvents(guildID, buffer)
	}

	// Signal that we received the voice server update (for Join waiting)
	c.pendingMu.Lock()
	pending := c.pending[guildID]
	c.pendingMu.Unlock()

	if pending != nil {
		pending.onEvent(false)
	}
}

// OnVoiceStateUpdate handles Discord voice state updates.
// This must be called from the Discord event handler.
func (c *DiscordVoiceConnectionGateway) OnVoiceStateUpdate(event *discordgo.VoiceStateUpdate) {
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

	// Handle disconnect immediately (no need to wait for VoiceServerUpdate)
	if channelID == nil {
		c.link.OnVoiceStateUpdate(context.Background(), guildID, nil, sessionID)
		c.unregisterGuild(guildID)
		c.clearVoiceBuffer(guildID)
		return
	}

	// Get or create voice buffer for this guild
	buffer := c.getOrCreateVoiceBuffer(guildID)

	// Store voice state data and check if both events are ready
	if buffer.setVoiceState(channelID, sessionID) {
		// Both events received, forward to Lavalink
		c.forwardBufferedVoiceEvents(guildID, buffer)
	}

	// Signal that we received the voice state update (for Join waiting)
	c.pendingMu.Lock()
	pending := c.pending[guildID]
	c.pendingMu.Unlock()

	if pending != nil {
		pending.onEvent(true)
	}
}

// getOrCreateVoiceBuffer returns the voice buffer for a guild, creating one if needed.
func (c *DiscordVoiceConnectionGateway) getOrCreateVoiceBuffer(
	guildID snowflake.ID,
) *voiceEventBuffer {
	c.voiceBufferMu.Lock()
	defer c.voiceBufferMu.Unlock()

	buffer, exists := c.voiceBuffers[guildID]
	if !exists {
		buffer = &voiceEventBuffer{}
		c.voiceBuffers[guildID] = buffer
	}
	return buffer
}

// clearVoiceBuffer removes the voice buffer for a guild.
func (c *DiscordVoiceConnectionGateway) clearVoiceBuffer(guildID snowflake.ID) {
	c.voiceBufferMu.Lock()
	defer c.voiceBufferMu.Unlock()
	delete(c.voiceBuffers, guildID)
}

func (c *DiscordVoiceConnectionGateway) registerMapping(
	guildID snowflake.ID,
	playerStateID core.PlayerStateID,
) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	c.stateToGuild[playerStateID] = guildID
	c.guildToState[guildID] = playerStateID
}

func (c *DiscordVoiceConnectionGateway) unregisterGuild(guildID snowflake.ID) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	playerStateID, ok := c.guildToState[guildID]
	if !ok {
		return
	}

	delete(c.guildToState, guildID)
	delete(c.stateToGuild, playerStateID)
}

func (c *DiscordVoiceConnectionGateway) isConnectedToChannel(
	guildID snowflake.ID,
	channelID string,
) bool {
	targetChannelID, err := snowflake.Parse(channelID)
	if err != nil {
		return false
	}

	c.voiceBufferMu.Lock()
	buffer, ok := c.voiceBuffers[guildID]
	c.voiceBufferMu.Unlock()

	if ok {
		currentChannelID, _, _, _ := buffer.getData()
		if currentChannelID != nil {
			return *currentChannelID == targetChannelID
		}
	}

	if c.session == nil || c.session.State == nil {
		return false
	}

	guild, err := c.session.State.Guild(guildID.String())
	if err != nil {
		return false
	}

	for _, voiceState := range guild.VoiceStates {
		if voiceState.UserID == c.botID.String() {
			return voiceState.ChannelID == channelID
		}
	}

	return false
}

// forwardBufferedVoiceEvents sends the buffered voice events to Lavalink.
func (c *DiscordVoiceConnectionGateway) forwardBufferedVoiceEvents(
	guildID snowflake.ID,
	buffer *voiceEventBuffer,
) {
	channelID, sessionID, token, endpoint := buffer.getData()

	slog.Debug("forwarding buffered voice events to Lavalink",
		"guild", guildID,
		"channel", channelID,
		"hasSessionID", sessionID != "",
	)

	// Forward to Lavalink in the correct order
	c.link.OnVoiceStateUpdate(context.Background(), guildID, channelID, sessionID)
	c.link.OnVoiceServerUpdate(context.Background(), guildID, token, endpoint)
}
