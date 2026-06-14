package content

import (
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func TestSearcher_Search_EmptyRepository(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryRepository()
	searcher := NewSearcher(repo)

	results, err := searcher.Search("anything")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results from empty repository, got %d", len(results))
	}
}

func TestSearcher_Search_WhitespaceQuery(t *testing.T) {
	t.Parallel()

	now := time.Now()
	repo := NewInMemoryRepository()
	file, _ := domain.NewFileNode(domain.MustURLPath("/test"), "Test", []byte("content"), now, 7)
	repo.Add(file)

	searcher := NewSearcher(repo)

	tests := []struct {
		name  string
		query string
	}{
		{"single space", " "},
		{"multiple spaces", "   "},
		{"tabs", "\t\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searcher.Search(tt.query)
			if err != nil {
				t.Errorf("Search(%q) error = %v", tt.query, err)
			}
			// Whitespace queries should either return no results or handle gracefully
			// The current implementation would search for whitespace literally
			_ = results
		})
	}
}

func TestSearcher_Search_SpecialCharacters(t *testing.T) {
	t.Parallel()

	now := time.Now()

	repo := NewInMemoryRepository()
	root, _ := repo.Root()
	file := newFile(t, now, "/api", "API v1.0.0", "content")
	root.AddChild(file)
	repo.Add(file)

	searcher := NewSearcher(repo)

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"dots in query", "v1.0", 1},
		{"partial version", "1.0.0", 1},
		{"with space", "API v1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results, err := searcher.Search(tt.query)
			if err != nil {
				t.Errorf("Search(%q) error = %v", tt.query, err)

				return
			}

			if len(results) != tt.wantCount {
				t.Errorf(
					"Search(%q) returned %d results, want %d",
					tt.query,
					len(results),
					tt.wantCount,
				)
			}
		})
	}
}
