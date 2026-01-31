package domain

import (
	"testing"
)

func TestNewSearchQuery(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedQuery  string
		expectedSource SearchSource
		expectedIsURL  bool
	}{
		{
			name:           "search term",
			input:          "never gonna give you up",
			expectedQuery:  "never gonna give you up",
			expectedSource: SourceYouTube,
			expectedIsURL:  false,
		},
		{
			name:           "search term with whitespace",
			input:          "  hello world  ",
			expectedQuery:  "hello world",
			expectedSource: SourceYouTube,
			expectedIsURL:  false,
		},
		{
			name:           "https URL",
			input:          "https://youtube.com/watch?v=dQw4w9WgXcQ",
			expectedQuery:  "https://youtube.com/watch?v=dQw4w9WgXcQ",
			expectedSource: SourceDirect,
			expectedIsURL:  true,
		},
		{
			name:           "http URL",
			input:          "http://example.com/audio.mp3",
			expectedQuery:  "http://example.com/audio.mp3",
			expectedSource: SourceDirect,
			expectedIsURL:  true,
		},
		{
			name:           "www URL",
			input:          "www.youtube.com/watch?v=abc",
			expectedQuery:  "www.youtube.com/watch?v=abc",
			expectedSource: SourceDirect,
			expectedIsURL:  true,
		},
		{
			name:           "empty string",
			input:          "",
			expectedQuery:  "",
			expectedSource: SourceYouTube,
			expectedIsURL:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSearchQuery(tt.input)

			if q.Query != tt.expectedQuery {
				t.Errorf("Query = %q, expected %q", q.Query, tt.expectedQuery)
			}
			if q.Source != tt.expectedSource {
				t.Errorf("Source = %q, expected %q", q.Source, tt.expectedSource)
			}
			if q.IsURL != tt.expectedIsURL {
				t.Errorf("IsURL = %v, expected %v", q.IsURL, tt.expectedIsURL)
			}
		})
	}
}

func TestNewSearchQueryWithSource(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		source         SearchSource
		expectedSource SearchSource
		expectedIsURL  bool
	}{
		{
			name:           "youtube music search",
			input:          "test song",
			source:         SourceYouTubeMusic,
			expectedSource: SourceYouTubeMusic,
			expectedIsURL:  false,
		},
		{
			name:           "soundcloud search",
			input:          "test song",
			source:         SourceSoundCloud,
			expectedSource: SourceSoundCloud,
			expectedIsURL:  false,
		},
		{
			name:           "URL ignores source",
			input:          "https://example.com/track",
			source:         SourceYouTubeMusic,
			expectedSource: SourceDirect,
			expectedIsURL:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSearchQueryWithSource(tt.input, tt.source)

			if q.Source != tt.expectedSource {
				t.Errorf("Source = %q, expected %q", q.Source, tt.expectedSource)
			}
			if q.IsURL != tt.expectedIsURL {
				t.Errorf("IsURL = %v, expected %v", q.IsURL, tt.expectedIsURL)
			}
		})
	}
}

func TestSearchQuery_LavalinkQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    *SearchQuery
		expected string
	}{
		{
			name: "youtube search",
			query: &SearchQuery{
				Query:  "test song",
				Source: SourceYouTube,
				IsURL:  false,
			},
			expected: "ytsearch:test song",
		},
		{
			name: "youtube music search",
			query: &SearchQuery{
				Query:  "test song",
				Source: SourceYouTubeMusic,
				IsURL:  false,
			},
			expected: "ytmsearch:test song",
		},
		{
			name: "soundcloud search",
			query: &SearchQuery{
				Query:  "test song",
				Source: SourceSoundCloud,
				IsURL:  false,
			},
			expected: "scsearch:test song",
		},
		{
			name: "direct URL",
			query: &SearchQuery{
				Query:  "https://youtube.com/watch?v=abc",
				Source: SourceDirect,
				IsURL:  true,
			},
			expected: "https://youtube.com/watch?v=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.LavalinkQuery(); got != tt.expected {
				t.Errorf("LavalinkQuery() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestSearchQuery_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		query    *SearchQuery
		expected bool
	}{
		{
			name:     "valid query",
			query:    &SearchQuery{Query: "test"},
			expected: true,
		},
		{
			name:     "empty query",
			query:    &SearchQuery{Query: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
