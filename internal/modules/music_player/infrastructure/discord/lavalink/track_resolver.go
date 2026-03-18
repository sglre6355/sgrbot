package lavalink

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Ensure LavalinkTrackResolver implements required interfaces.
var _ ports.TrackResolver = (*LavalinkTrackResolver)(nil)

// LavalinkTrackResolver implements ports.TrackResolver using Lavalink.
type LavalinkTrackResolver struct {
	link       disgolink.Client
	trackCache *TrackCache
}

// NewLavalinkTrackResolver creates a new LavalinkTrackResolver.
func NewLavalinkTrackResolver(
	link disgolink.Client,
	trackCache *TrackCache,
) *LavalinkTrackResolver {
	return &LavalinkTrackResolver{
		link:       link,
		trackCache: trackCache,
	}
}

// ResolveQuery searches for tracks using the given query.
// Non-URL queries are prefixed with "ytsearch:" for YouTube search.
func (r *LavalinkTrackResolver) ResolveQuery(
	ctx context.Context,
	query string,
) (domain.TrackList, error) {
	if !isURL(query) {
		query = "ytsearch:" + query
	}

	node := r.link.BestNode()
	if node == nil {
		return domain.TrackList{}, fmt.Errorf("no available Lavalink node")
	}

	result, err := node.LoadTracks(ctx, query)
	if err != nil {
		return domain.TrackList{}, fmt.Errorf("failed to load tracks: %w", err)
	}

	switch data := result.Data.(type) {
	case lavalink.Track:
		return domain.NewTrackList(
			domain.TrackListTypeTrack,
			[]domain.Track{r.trackCache.ConvertAndCache(data)},
		), nil

	case lavalink.Playlist:
		tracks := make([]domain.Track, len(data.Tracks))
		for i, track := range data.Tracks {
			tracks[i] = r.trackCache.ConvertAndCache(track)
		}
		sourceName := data.Tracks[0].Info.SourceName
		identifier, cleanURL := extractPlaylistInfo(query, sourceName)
		return domain.NewTrackList(
			domain.TrackListTypePlaylist,
			tracks,
			domain.WithPlaylistInfo(identifier, data.Info.Name, cleanURL),
		), nil

	case lavalink.Search:
		tracks := make([]domain.Track, len(data))
		for i, track := range data {
			tracks[i] = r.trackCache.ConvertAndCache(track)
		}
		return domain.NewTrackList(
			domain.TrackListTypeSearch,
			tracks,
		), nil

	case lavalink.Exception:
		return domain.TrackList{}, fmt.Errorf("lavalink load error: %w", data)

	default:
		return domain.TrackList{}, nil
	}
}

// isURL checks if the input looks like a URL.
func isURL(input string) bool {
	return strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "www.")
}

// extractPlaylistInfo extracts a playlist identifier and clean URL from the query.
// It applies provider-specific parsing for YouTube and Spotify, falling back to
// the raw query for unrecognized providers.
func extractPlaylistInfo(query, sourceName string) (identifier, cleanURL string) {
	u, err := url.Parse(query)
	if err != nil {
		return query, query
	}
	base := u.Scheme + "://" + u.Host

	switch sourceName {
	case "youtube":
		if listID := u.Query().Get("list"); listID != "" {
			return listID, base + "/playlist?list=" + listID
		}
	case "spotify":
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			typ, id := parts[0], parts[1]
			return id, base + "/" + typ + "/" + id
		}
	}
	return query, query
}
