package domain

import (
	"strings"
)

// SearchSource represents the source for searching tracks.
type SearchSource string

const (
	// SourceYouTube searches YouTube.
	SourceYouTube SearchSource = "ytsearch"
	// SourceYouTubeMusic searches YouTube Music.
	SourceYouTubeMusic SearchSource = "ytmsearch"
	// SourceSoundCloud searches SoundCloud.
	SourceSoundCloud SearchSource = "scsearch"
	// SourceDirect indicates a direct URL (no search prefix).
	SourceDirect SearchSource = ""
)

// SearchQuery represents a query for searching tracks.
type SearchQuery struct {
	Query  string       // The search term or URL
	Source SearchSource // The search source
	IsURL  bool         // Whether the query is a direct URL
}

// NewSearchQuery creates a SearchQuery from user input.
// If the input is a URL, it returns a direct query.
// Otherwise, it uses YouTube search as the default.
func NewSearchQuery(input string) *SearchQuery {
	input = strings.TrimSpace(input)

	if isURL(input) {
		return &SearchQuery{
			Query:  input,
			Source: SourceDirect,
			IsURL:  true,
		}
	}

	return &SearchQuery{
		Query:  input,
		Source: SourceYouTube,
		IsURL:  false,
	}
}

// NewSearchQueryWithSource creates a SearchQuery with a specific source.
func NewSearchQueryWithSource(input string, source SearchSource) *SearchQuery {
	input = strings.TrimSpace(input)

	if isURL(input) {
		return &SearchQuery{
			Query:  input,
			Source: SourceDirect,
			IsURL:  true,
		}
	}

	return &SearchQuery{
		Query:  input,
		Source: source,
		IsURL:  false,
	}
}

// LavalinkQuery returns the query string formatted for Lavalink.
func (q *SearchQuery) LavalinkQuery() string {
	if q.IsURL {
		return q.Query
	}
	return string(q.Source) + ":" + q.Query
}

// IsValid returns true if the query is not empty.
func (q *SearchQuery) IsValid() bool {
	return q.Query != ""
}

// isURL checks if the input looks like a URL.
func isURL(input string) bool {
	return strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "www.")
}
