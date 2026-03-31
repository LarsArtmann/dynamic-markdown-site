package server

import (
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func TestFindSuggestions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requested      string
		paths          []domain.URLPath
		maxSuggestions int
		wantCount      int
		wantFirst      string // expected first suggestion path
	}{
		{
			name:           "empty paths returns nil",
			requested:      "/test",
			paths:          nil,
			maxSuggestions: 5,
			wantCount:      0,
		},
		{
			name:           "empty request returns nil",
			requested:      "",
			paths:          []domain.URLPath{domain.MustURLPath("/home")},
			maxSuggestions: 5,
			wantCount:      0,
		},
		{
			name:           "exact match is excluded",
			requested:      "/home",
			paths:          []domain.URLPath{domain.MustURLPath("/home"), domain.MustURLPath("/about")},
			maxSuggestions: 5,
			wantCount:      1,
			wantFirst:      "/about",
		},
		{
			name:           "typo correction with Levenshtein",
			requested:      "/abou",
			paths:          []domain.URLPath{domain.MustURLPath("/about"), domain.MustURLPath("/home"), domain.MustURLPath("/contact")},
			maxSuggestions: 5,
			wantCount:      3,
			wantFirst:      "/about",
		},
		{
			name:           "prefix match gets boosted",
			requested:      "/bl",
			paths:          []domain.URLPath{domain.MustURLPath("/blog"), domain.MustURLPath("/about"), domain.MustURLPath("/home")},
			maxSuggestions: 5,
			wantCount:      3,
			wantFirst:      "/blog",
		},
		{
			name:           "max suggestions respected",
			requested:      "/t",
			paths:          []domain.URLPath{
				domain.MustURLPath("/test1"),
				domain.MustURLPath("/test2"),
				domain.MustURLPath("/test3"),
				domain.MustURLPath("/test4"),
				domain.MustURLPath("/test5"),
				domain.MustURLPath("/test6"),
			},
			maxSuggestions: 3,
			wantCount:      3,
		},
		{
			name:           "substring match gets boosted",
			requested:      "log",
			paths:          []domain.URLPath{domain.MustURLPath("/blog"), domain.MustURLPath("/about"), domain.MustURLPath("/login")},
			maxSuggestions: 5,
			wantCount:      3,
			wantFirst:      "/blog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findSuggestions(tt.requested, tt.paths, tt.maxSuggestions)

			if len(got) != tt.wantCount {
				t.Errorf("findSuggestions() returned %d suggestions, want %d", len(got), tt.wantCount)
			}

			if tt.wantFirst != "" && len(got) > 0 {
				if got[0].Path.String() != tt.wantFirst {
					t.Errorf("findSuggestions() first suggestion = %q, want %q", got[0].Path.String(), tt.wantFirst)
				}
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "identical strings",
			a:    "hello",
			b:    "hello",
			want: 0,
		},
		{
			name: "empty first string",
			a:    "",
			b:    "hello",
			want: 5,
		},
		{
			name: "empty second string",
			a:    "hello",
			b:    "",
			want: 5,
		},
		{
			name: "single insertion",
			a:    "hello",
			b:    "hellos",
			want: 1,
		},
		{
			name: "single deletion",
			a:    "hello",
			b:    "hell",
			want: 1,
		},
		{
			name: "single substitution",
			a:    "hello",
			b:    "hallo",
			want: 1,
		},
		{
			name: "typo correction",
			a:    "kitten",
			b:    "sitting",
			want: 3,
		},
		{
			name: "completely different",
			a:    "abc",
			b:    "xyz",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := levenshteinDistance(tt.a, tt.b)

			if got != tt.want {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSuggestedPath_ScoreOrdering(t *testing.T) {
	t.Parallel()

	paths := []domain.URLPath{
		domain.MustURLPath("/documentation"),
		domain.MustURLPath("/docs"),
		domain.MustURLPath("/about"),
		domain.MustURLPath("/home"),
	}

	suggestions := findSuggestions("/doc", paths, 5)

	// Check that results are sorted by score descending
	for i := 1; i < len(suggestions); i++ {
		if suggestions[i].Score > suggestions[i-1].Score {
			t.Errorf("suggestions not sorted by score: %v > %v at index %d", suggestions[i].Score, suggestions[i-1].Score, i)
		}
	}

	// "/docs" should be first (exact prefix match with boost)
	if len(suggestions) > 0 && suggestions[0].Path.String() != "/docs" {
		t.Errorf("expected /docs to be first suggestion, got %s", suggestions[0].Path.String())
	}
}
