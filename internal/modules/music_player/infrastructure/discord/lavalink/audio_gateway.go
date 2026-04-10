package lavalink

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/infrastructure/discord"
)

// Ensure LavalinkAudioGateway implements required interfaces.
var _ ports.AudioGateway = (*LavalinkAudioGateway)(nil)

// LavalinkAudioGateway implements ports.AudioGateway using Lavalink.
type LavalinkAudioGateway struct {
	link            disgolink.Client
	trackCache      *TrackCache
	voiceConnection *discord.DiscordVoiceConnectionGateway
	events          ports.EventPublisher

	mu             sync.RWMutex
	currentEntries map[domain.PlayerStateID]domain.QueueEntry
}

// NewLavalinkAudioGateway creates a new LavalinkAudioGateway.
// It registers track event listeners on the provided disgolink client.
func NewLavalinkAudioGateway(
	link disgolink.Client,
	trackCache *TrackCache,
	voiceConnection *discord.DiscordVoiceConnectionGateway,
	events ports.EventPublisher,
) *LavalinkAudioGateway {
	gw := &LavalinkAudioGateway{
		link:            link,
		trackCache:      trackCache,
		voiceConnection: voiceConnection,
		events:          events,
		currentEntries:  make(map[domain.PlayerStateID]domain.QueueEntry),
	}
	return gw
}

// SetLink sets the disgolink client. This is used to break the circular dependency
// when the client is created after the audio gateway.
func (g *LavalinkAudioGateway) SetLink(link disgolink.Client) {
	g.link = link
}

// ListenerOpts returns disgolink config options that register this gateway's
// track event listeners. Pass these to lavalink.NewClient.
func (g *LavalinkAudioGateway) ListenerOpts() []disgolink.ConfigOpt {
	return []disgolink.ConfigOpt{
		disgolink.WithListenerFunc(g.onLavalinkTrackStart),
		disgolink.WithListenerFunc(g.onLavalinkTrackEnd),
		disgolink.WithListenerFunc(g.onLavalinkTrackException),
		disgolink.WithListenerFunc(g.onLavalinkTrackStuck),
	}
}

// --- ports.AudioGateway ---

// Play resolves a fresh encoded track from Lavalink and starts playback.
func (g *LavalinkAudioGateway) Play(
	ctx context.Context,
	playerStateID domain.PlayerStateID,
	entry domain.QueueEntry,
) error {
	guildID, err := g.voiceConnection.ResolveGuildID(playerStateID)
	if err != nil {
		return err
	}

	track, err := resolveFromLavalink(ctx, g.link, *entry.Track())
	if err != nil {
		return fmt.Errorf("failed to resolve track %q: %w", entry.Track().ID(), err)
	}

	// Update trackCache with fresh data
	g.trackCache.ConvertAndCache(track)

	player := g.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithEncodedTrack(track.Encoded)); err != nil {
		return fmt.Errorf("failed to play track: %w", err)
	}

	g.mu.Lock()
	g.currentEntries[playerStateID] = entry
	g.mu.Unlock()

	return nil
}

// Stop stops the current playback.
func (g *LavalinkAudioGateway) Stop(ctx context.Context, playerStateID domain.PlayerStateID) error {
	guildID, err := g.voiceConnection.ResolveGuildID(playerStateID)
	if err != nil {
		return err
	}

	player := g.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithNullTrack()); err != nil {
		return fmt.Errorf("failed to stop playback: %w", err)
	}

	return nil
}

// Pause pauses the current playback.
func (g *LavalinkAudioGateway) Pause(
	ctx context.Context,
	playerStateID domain.PlayerStateID,
) error {
	guildID, err := g.voiceConnection.ResolveGuildID(playerStateID)
	if err != nil {
		return err
	}

	player := g.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithPaused(true)); err != nil {
		return fmt.Errorf("failed to pause playback: %w", err)
	}

	return nil
}

// Resume resumes the current playback.
func (g *LavalinkAudioGateway) Resume(
	ctx context.Context,
	playerStateID domain.PlayerStateID,
) error {
	guildID, err := g.voiceConnection.ResolveGuildID(playerStateID)
	if err != nil {
		return err
	}

	player := g.link.Player(guildID)

	if err := player.Update(ctx, lavalink.WithPaused(false)); err != nil {
		return fmt.Errorf("failed to resume playback: %w", err)
	}

	return nil
}

// --- Lavalink event listeners ---

func (g *LavalinkAudioGateway) onLavalinkTrackStart(
	player disgolink.Player,
	event lavalink.TrackStartEvent,
) {
	slog.Debug("track started", "guild", player.GuildID(), "track", event.Track.Info.Title)
}

func (g *LavalinkAudioGateway) onLavalinkTrackEnd(
	player disgolink.Player,
	event lavalink.TrackEndEvent,
) {
	slog.Debug("track ended", "guild", player.GuildID(), "reason", event.Reason)

	if event.Reason != lavalink.TrackEndReasonFinished &&
		event.Reason != lavalink.TrackEndReasonLoadFailed {
		return
	}

	playerStateID, ok := g.voiceConnection.ResolvePlayerStateID(player.GuildID())
	if !ok {
		slog.Warn("no player state mapping for guild", "guild", player.GuildID())
		return
	}

	g.mu.RLock()
	entry := g.currentEntries[playerStateID]
	g.mu.RUnlock()

	g.events.Publish(context.Background(), domain.TrackEndedEvent{
		PlayerStateID: playerStateID,
		Entry:         entry,
		TrackFailed:   event.Reason == lavalink.TrackEndReasonLoadFailed,
	})
}

func (g *LavalinkAudioGateway) onLavalinkTrackException(
	player disgolink.Player,
	event lavalink.TrackExceptionEvent,
) {
	slog.Warn("track exception", "guild", player.GuildID(), "error", event.Exception.Message)
}

func (g *LavalinkAudioGateway) onLavalinkTrackStuck(
	player disgolink.Player,
	event lavalink.TrackStuckEvent,
) {
	slog.Warn("track stuck", "guild", player.GuildID(), "threshold", event.Threshold)
}
