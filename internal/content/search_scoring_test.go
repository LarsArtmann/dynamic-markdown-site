package content

import (
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// titleScoringCase is a single-query, single-expected-result description used
// by TestSearcher_Search_TitleMatching. Flattening the wantScores /
// wantHighlight maps into a single result keeps table entries small and
// avoids the symmetric-clone pattern that table-driven maps produce.
type titleScoringCase struct {
	name          string
	query         string
	wantScore     float64
	wantHighlight string
}

func TestSearcher_Search_TitleMatching(t *testing.T) {
	t.Parallel()

	now := time.Now()

	repo := repoWithFile(t, now, "/docs/guide", "Getting Started Guide", "# Guide content")
	searcher := NewSearcher(repo)

	cases := []titleScoringCase{
		{
			name:          "exact title match scores 1.0",
			query:         "Getting Started Guide",
			wantScore:     1.0,
			wantHighlight: "<mark>Getting Started Guide</mark>",
		},
		{
			name:          "partial title match scores 0.5",
			query:         "Getting",
			wantScore:     0.5,
			wantHighlight: "<mark>Getting</mark> Started Guide",
		},
		{
			name:      "case insensitive matching",
			query:     "getting started",
			wantScore: 0.5,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results, err := searcher.Search(tt.query)
			if err != nil {
				t.Fatalf("Searcher.Search() error = %v", err)
			}

			wantScores := map[string]float64{"Getting Started Guide": tt.wantScore}
			wantHighlights := map[string]string{}
			if tt.wantHighlight != "" {
				wantHighlights["Getting Started Guide"] = tt.wantHighlight
			}

			assertSearchResults(t, results, 1, searchExpectations{
				scores:    wantScores,
				highlight: wantHighlights,
			})
		})
	}
}

func TestSearcher_Search_NestedDirectories(t *testing.T) {
	t.Parallel()

	now := time.Now()

	repo := NewInMemoryRepository()

	root, _ := repo.Root()
	docsDir, _ := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Documentation", now)
	apiDir, _ := domain.NewDirectoryNode(domain.MustURLPath("/docs/api"), "API", now)

	repo.Add(docsDir)
	repo.Add(apiDir)

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
