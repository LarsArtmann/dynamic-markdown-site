package content

import (
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func TestSearcher_Search_TitleMatching(t *testing.T) {
	t.Parallel()

	now := time.Now()

	repo := repoWithFile(t, now, "/docs/guide", "Getting Started Guide", "# Guide content")
	searcher := NewSearcher(repo)

	tests := []struct {
		name          string
		query         string
		wantCount     int
		wantScores    map[string]float64
		wantHighlight map[string]string
	}{
		{
			name:      "exact title match scores 1.0",
			query:     "Getting Started Guide",
			wantCount: 1,
			wantScores: map[string]float64{
				"Getting Started Guide": 1.0,
			},
			wantHighlight: map[string]string{
				"Getting Started Guide": "<mark>Getting Started Guide</mark>",
			},
		},
		{
			name:      "partial title match scores 0.5",
			query:     "Getting",
			wantCount: 1,
			wantScores: map[string]float64{
				"Getting Started Guide": 0.5,
			},
			wantHighlight: map[string]string{
				"Getting Started Guide": "<mark>Getting</mark> Started Guide",
			},
		},
		{
			name:      "case insensitive matching",
			query:     "getting started",
			wantCount: 1,
			wantScores: map[string]float64{
				"Getting Started Guide": 0.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results, err := searcher.Search(tt.query)
			if err != nil {
				t.Fatalf("Searcher.Search() error = %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf(
					"Searcher.Search() returned %d results, want %d",
					len(results),
					tt.wantCount,
				)
			}

			for _, result := range results {
				title := result.Node.Title()
				if wantScore, ok := tt.wantScores[title]; ok {
					if result.Score != wantScore {
						t.Errorf("result[%q].Score = %v, want %v", title, result.Score, wantScore)
					}
				}

				if wantHighlight, ok := tt.wantHighlight[title]; ok {
					if result.Highlighted != wantHighlight {
						t.Errorf(
							"result[%q].Highlighted = %q, want %q",
							title,
							result.Highlighted,
							wantHighlight,
						)
					}
				}
			}
		})
	}
}

func TestSearcher_Search_NestedDirectories(t *testing.T) {
	t.Parallel()

	now := time.Now()

	repo := NewInMemoryRepository()

	// Create nested structure
	root, _ := repo.Root()
	docsDir, _ := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Documentation", now)
	apiDir, _ := domain.NewDirectoryNode(domain.MustURLPath("/docs/api"), "API", now)

	file1 := newFile(t, now, "/docs/intro", "Introduction", "Welcome")
	file2 := newFile(t, now, "/docs/api/v1", "API v1", "REST API v1")
	file3 := newFile(t, now, "/blog/post1", "Blog Post", "Latest news")

	root.AddChild(docsDir)
	docsDir.AddChild(file1)
	docsDir.AddChild(apiDir)
	apiDir.AddChild(file2)
	root.AddChild(file3)
	repo.Add(file1)
	repo.Add(file2)
	repo.Add(file3)

	searcher := NewSearcher(repo)

	results, err := searcher.Search("API")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected results from nested directories, got none")
	}

	// Verify we found matches from nested directories
	foundV1 := false
	for _, result := range results {
		if result.Node.Title() == "API v1" {
			foundV1 = true
		}
	}

	if !foundV1 {
		t.Error("expected to find 'API v1' from nested directory")
	}
}
