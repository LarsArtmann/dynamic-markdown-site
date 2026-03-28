package content

import (
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// newFile creates a FileNode for testing purposes.
// It fails the test if node creation fails.
func newFile(t *testing.T, now time.Time, path, title, content string) *domain.FileNode {
	t.Helper()

	node, err := domain.NewFileNode(
		domain.MustURLPath(path),
		title,
		[]byte(content),
		now,
		uint64(len(content)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	return node
}

// setupRepoWithFiles creates an InMemoryRepository and adds the given files as root children.
func setupRepoWithFiles(files ...*domain.FileNode) *InMemoryRepository {
	repo := NewInMemoryRepository()

	root, _ := repo.Root()
	for _, file := range files {
		root.AddChild(file)
		repo.Add(file)
	}

	return repo
}

func TestSearcher_Search(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		setupRepo     func() *InMemoryRepository
		query         string
		wantCount     int
		wantScores    map[string]float64 // title -> expected score
		wantErr       bool
		wantHighlight map[string]string // title -> expected highlighted text
	}{
		{
			name:      "empty query returns nil",
			setupRepo: NewInMemoryRepository,
			query:     "",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "exact title match scores 1.0",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/guide", "Getting Started Guide", "# Guide content"),
				)
			},
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
			name: "partial title match scores 0.5",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/guide", "Getting Started Guide", "# Guide content"),
				)
			},
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
			name: "case insensitive matching",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/guide", "Getting Started Guide", "# Guide content"),
				)
			},
			query:     "getting started",
			wantCount: 1,
			wantScores: map[string]float64{
				"Getting Started Guide": 0.5,
			},
		},
		{
			name: "no match returns empty results",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/guide", "Getting Started Guide", "# Guide content"),
				)
			},
			query:     "nonexistent",
			wantCount: 0,
		},
		{
			name: "results sorted by score descending",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/a", "Go Tutorial", "Content about Go"),
					newFile(t, now, "/docs/b", "Go", "Go programming language"),
					newFile(t, now, "/docs/c", "Advanced Go Patterns", "Advanced patterns"),
				)
			},
			query:     "Go",
			wantCount: 3,
			wantScores: map[string]float64{
				"Go":                   1.0, // exact match
				"Go Tutorial":          0.5, // partial match
				"Advanced Go Patterns": 0.5, // partial match
			},
		},
		{
			name: "search with multiple files",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/intro", "Introduction", "Welcome to the docs"),
					newFile(t, now, "/docs/guide", "User Guide", "How to use this"),
					newFile(t, now, "/docs/api", "API Reference", "API documentation"),
					newFile(t, now, "/blog/post", "Blog Post", "Latest news"),
				)
			},
			query:     "Guide",
			wantCount: 1,
			wantScores: map[string]float64{
				"User Guide": 0.5,
			},
		},
		{
			name: "query matching multiple files",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/api-v1", "API v1", "Old API"),
					newFile(t, now, "/docs/api-v2", "API v2", "New API"),
					newFile(t, now, "/docs/guide", "Guide", "Documentation guide"),
				)
			},
			query:     "API",
			wantCount: 2,
			wantScores: map[string]float64{
				"API v1": 0.5,
				"API v2": 0.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			searcher := NewSearcher(repo)

			results, err := searcher.Search(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Searcher.Search() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if len(results) != tt.wantCount {
				t.Errorf(
					"Searcher.Search() returned %d results, want %d",
					len(results),
					tt.wantCount,
				)

				return
			}

			// Verify scores
			for _, result := range results {
				title := result.Node.Title()
				if wantScore, ok := tt.wantScores[title]; ok {
					if result.Score != wantScore {
						t.Errorf("result[%q].Score = %v, want %v", title, result.Score, wantScore)
					}
				}
			}

			// Verify highlights
			for _, result := range results {
				title := result.Node.Title()
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

			// Verify results are sorted by score descending
			for i := 1; i < len(results); i++ {
				if results[i].Score > results[i-1].Score {
					t.Errorf("results not sorted: results[%d].Score (%v) > results[%d].Score (%v)",
						i, results[i].Score, i-1, results[i-1].Score)
				}
			}
		})
	}
}

func TestSearcher_Search_NestedDirectories(t *testing.T) {
	now := time.Now()

	repo := NewInMemoryRepository()

	// Create nested structure
	root, _ := repo.Root()
	docsDir, _ := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Documentation", now)
	advancedDir, _ := domain.NewDirectoryNode(
		domain.MustURLPath("/docs/advanced"),
		"Advanced Topics",
		now,
	)

	file1, _ := domain.NewFileNode(
		domain.MustURLPath("/docs/intro"),
		"Introduction",
		[]byte("Intro content"),
		now,
		12,
	)
	file2, _ := domain.NewFileNode(
		domain.MustURLPath("/docs/advanced/tutorial"),
		"Advanced Tutorial",
		[]byte("Advanced content"),
		now,
		16,
	)

	root.AddChild(docsDir)
	docsDir.AddChild(file1)
	docsDir.AddChild(advancedDir)
	advancedDir.AddChild(file2)

	repo.Add(file1)
	repo.Add(file2)

	searcher := NewSearcher(repo)

	results, err := searcher.Search("Tutorial")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Node.Title() != "Advanced Tutorial" {
		t.Errorf("expected 'Advanced Tutorial', got %q", results[0].Node.Title())
	}
}

func TestHighlightMatch(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		query    string
		expected string
	}{
		{
			name:     "exact match",
			text:     "Hello World",
			query:    "Hello World",
			expected: "<mark>Hello World</mark>",
		},
		{
			name:     "partial match at start",
			text:     "Hello World",
			query:    "Hello",
			expected: "<mark>Hello</mark> World",
		},
		{
			name:     "partial match in middle",
			text:     "Hello World",
			query:    "lo Wo",
			expected: "Hel<mark>lo Wo</mark>rld",
		},
		{
			name:     "partial match at end",
			text:     "Hello World",
			query:    "World",
			expected: "Hello <mark>World</mark>",
		},
		{
			name:     "case insensitive",
			text:     "Hello World",
			query:    "hello",
			expected: "<mark>Hello</mark> World",
		},
		{
			name:     "no match returns original",
			text:     "Hello World",
			query:    "xyz",
			expected: "Hello World",
		},
		{
			name:     "empty query returns original",
			text:     "Hello World",
			query:    "",
			expected: "Hello World",
		},
		{
			name:     "unicode text",
			text:     "Gödel's Theorem",
			query:    "gödel",
			expected: "<mark>Gödel</mark>'s Theorem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightMatch(tt.text, tt.query)
			if result != tt.expected {
				t.Errorf(
					"highlightMatch(%q, %q) = %q, want %q",
					tt.text,
					tt.query,
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestSearcher_Search_EmptyRepository(t *testing.T) {
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

func TestSearcher_Search_ContentBody(t *testing.T) {
	now := time.Now()

	setupRepoWithFiles := func(files ...*domain.FileNode) *InMemoryRepository {
		repo := NewInMemoryRepository()

		root, _ := repo.Root()
		for _, file := range files {
			root.AddChild(file)
			repo.Add(file)
		}

		return repo
	}

	tests := []struct {
		name         string
		setupRepo    func() *InMemoryRepository
		query        string
		wantCount    int
		wantScores   map[string]float64
		wantSnippets map[string]bool // title -> should have snippet
	}{
		{
			name: "content body match with lower score",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/a", "Guide", "This is a tutorial about programming"),
					newFile(t, now, "/docs/b", "Reference", "Learn how to use the API effectively"),
				)
			},
			query:     "tutorial",
			wantCount: 1,
			wantScores: map[string]float64{
				"Guide": 0.3,
			},
			wantSnippets: map[string]bool{
				"Guide": true,
			},
		},
		{
			name: "title match takes priority over content match",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/a", "Tutorial", "This is a guide"),
					newFile(t, now, "/docs/b", "Guide", "This is a tutorial about programming"),
				)
			},
			query:     "tutorial",
			wantCount: 2,
			wantScores: map[string]float64{
				"Tutorial": 1.0, // exact title match
				"Guide":    0.3, // content match only
			},
			wantSnippets: map[string]bool{
				"Tutorial": false, // title match, no snippet
				"Guide":    true,  // content match, has snippet
			},
		},
		{
			name: "partial title match beats content match",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/a", "Go Tutorial", "Introduction"),
					newFile(t, now, "/docs/b", "Guide", "Learn Go programming"),
				)
			},
			query:     "Go",
			wantCount: 2,
			wantScores: map[string]float64{
				"Go Tutorial": 0.5, // partial title match
				"Guide":       0.3, // content match
			},
		},
		{
			name: "content match is case insensitive",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/a", "Guide", "This is a TUTORIAL about coding"),
				)
			},
			query:     "tutorial",
			wantCount: 1,
			wantScores: map[string]float64{
				"Guide": 0.3,
			},
		},
		{
			name: "multiple content matches",
			setupRepo: func() *InMemoryRepository {
				return setupRepoWithFiles(
					newFile(t, now, "/docs/a", "Intro", "Learn about Go basics"),
					newFile(t, now, "/docs/b", "Advanced", "Go concurrency patterns"),
					newFile(t, now, "/docs/c", "Reference", "API documentation"),
				)
			},
			query:     "Go",
			wantCount: 2,
			wantScores: map[string]float64{
				"Intro":    0.3,
				"Advanced": 0.3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			searcher := NewSearcher(repo)

			results, err := searcher.Search(tt.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("expected %d results, got %d", tt.wantCount, len(results))

				return
			}

			for _, result := range results {
				title := result.Node.Title()
				if wantScore, ok := tt.wantScores[title]; ok {
					if result.Score != wantScore {
						t.Errorf("result[%q].Score = %v, want %v", title, result.Score, wantScore)
					}
				}

				if wantSnippet, ok := tt.wantSnippets[title]; ok {
					hasSnippet := result.Snippet != ""
					if hasSnippet != wantSnippet {
						t.Errorf(
							"result[%q].Snippet empty=%v, want empty=%v (snippet=%q)",
							title,
							!hasSnippet,
							!wantSnippet,
							result.Snippet,
						)
					}
				}
			}
		})
	}
}

func TestExtractSnippet(t *testing.T) {
	tests := []struct {
		name    string
		content string
		query   string
		padding int
		wantHas string // substring that should be present
		wantNo  string // substring that should NOT be present (optional)
	}{
		{
			name:    "extract around match",
			content: "This is a long piece of content with the keyword hidden inside it",
			query:   "keyword",
			padding: 10,
			wantHas: "keyword",
		},
		{
			name:    "match at start of content",
			content: "keyword at the beginning of the content",
			query:   "keyword",
			padding: 10,
			wantHas: "keyword",
			wantNo:  "...k", // no ellipsis before since at start
		},
		{
			name:    "match at end of content",
			content: "content ends with the keyword",
			query:   "keyword",
			padding: 10,
			wantHas: "keyword",
		},
		{
			name:    "case insensitive match",
			content: "This has a KEYWORD in uppercase",
			query:   "keyword",
			padding: 5,
			wantHas: "KEYWORD",
		},
		{
			name:    "highlight applied to snippet",
			content: "Some text with keyword in the middle",
			query:   "keyword",
			padding: 10,
			wantHas: "<mark>keyword</mark>",
		},
		{
			name:    "no match returns empty",
			content: "This content has no match",
			query:   "nonexistent",
			padding: 10,
			wantHas: "",
		},
		{
			name:    "empty query returns empty",
			content: "Some content",
			query:   "",
			padding: 10,
			wantHas: "",
		},
		{
			name:    "empty content returns empty",
			content: "",
			query:   "test",
			padding: 10,
			wantHas: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSnippet([]byte(tt.content), tt.query, tt.padding)
			if tt.wantHas == "" {
				if result != "" {
					t.Errorf("expected empty snippet, got %q", result)
				}

				return
			}

			if !contains(result, tt.wantHas) {
				t.Errorf("snippet %q should contain %q", result, tt.wantHas)
			}

			if tt.wantNo != "" && contains(result, tt.wantNo) {
				t.Errorf("snippet %q should NOT contain %q", result, tt.wantNo)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
