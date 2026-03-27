// Package content provides content management and search functionality.
//
// Search Implementation Notes:
//
// The current search implementation uses a simple O(n) recursive traversal
// of the content tree. This is acceptable for small to medium sites but
// will degrade linearly as content grows.
//
// Performance Characteristics:
//   - Time Complexity: O(n) where n = total number of content nodes
//   - Space Complexity: O(r) where r = number of matching results
//   - Title search: Case-insensitive substring match
//   - Content search: Full text scan (expensive for large files)
//   - Scoring: Exact match (1.0) > Title contains (0.5) > Content contains (0.3)
//
// Recommended Limits:
//   - Optimal: < 1,000 markdown files
//   - Acceptable: < 10,000 markdown files
//   - Degraded: > 10,000 markdown files (consider indexed search)
//
// For large sites, consider integrating bleve or similar full-text
// search engine for sub-millisecond query times.
package content

import (
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// SearchResult represents a single search result.
type SearchResult struct {
	Node        domain.ContentNode
	Score       float64
	Highlighted string
	Snippet     string // Context snippet for content matches
}

// Searcher provides search functionality over content.
type Searcher struct {
	repo Repository
}

// NewSearcher creates a new searcher for the given repository.
func NewSearcher(repo Repository) *Searcher {
	return &Searcher{repo: repo}
}

// Search performs a case-insensitive search over content titles and body.
func (s *Searcher) Search(query string) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}

	queryLower := strings.ToLower(query)

	var results []SearchResult

	root, err := s.repo.Root()
	if err != nil {
		return nil, errors.Wrapf(err, "search %q: %w", query, err)
	}

	s.searchNode(root, queryLower, &results)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

func (s *Searcher) searchNode(node domain.ContentNode, query string, results *[]SearchResult) {
	title := strings.ToLower(node.Title())

	var (
		score       float64
		highlighted string
		snippet     string
	)

	// Title matching takes priority

	if title == query {
		score = 1.0
		highlighted = highlightMatch(node.Title(), query)
	} else if strings.Contains(title, query) {
		score = 0.5
		highlighted = highlightMatch(node.Title(), query)
	} else if file, ok := node.(*domain.FileNode); ok {
		// Fall back to content body search
		content := strings.ToLower(string(file.Content()))
		if strings.Contains(content, query) {
			score = 0.3
			highlighted = node.Title() // No highlighting for content matches, use title
			snippet = extractSnippet(file.Content(), query, 50)
		}
	}

	if score > 0 {
		*results = append(*results, SearchResult{
			Node:        node,
			Score:       score,
			Highlighted: highlighted,
			Snippet:     snippet,
		})
	}

	if dir, ok := node.(*domain.DirectoryNode); ok {
		for _, child := range dir.Children() {
			s.searchNode(child, query, results)
		}
	}
}

func highlightMatch(text, query string) string {
	if query == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	index := strings.Index(lowerText, lowerQuery)
	if index == -1 {
		return text
	}

	return text[:index] + "<mark>" + text[index:index+len(query)] + "</mark>" + text[index+len(query):]
}

// extractSnippet extracts a context snippet around the first match of query in content.
// padding specifies how many characters to include before and after the match.
func extractSnippet(content []byte, query string, padding int) string {
	if query == "" || len(content) == 0 {
		return ""
	}

	lowerContent := strings.ToLower(string(content))
	lowerQuery := strings.ToLower(query)

	index := strings.Index(lowerContent, lowerQuery)
	if index == -1 {
		return ""
	}

	// Calculate snippet boundaries
	start := max(index-padding, 0)

	end := min(index+len(query)+padding, len(content))

	snippet := string(content[start:end])

	// Add ellipsis if truncated
	if start > 0 {
		snippet = "..." + snippet
	}

	if end < len(content) {
		snippet += "..."
	}

	return highlightMatch(snippet, query)
}
