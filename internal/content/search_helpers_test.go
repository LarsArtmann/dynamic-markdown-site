package content

import (
	"testing"
)

// searchExpectations bundles the per-title assertions run against every
// SearchResult. Zero-value maps mean "no expectation"; this lets the same
// helper drive title-score tests, highlight tests, and snippet tests.
type searchExpectations struct {
	scores    map[string]float64
	highlight map[string]string
	snippets  map[string]bool
}

// assertSearchResults checks len(results) == wantCount and then runs every
// per-title expectation. The snippet check is opt-in via wantSnippets: when
// that map is non-nil the snippet-presence invariant is enforced.
func assertSearchResults(
	t *testing.T,
	results []SearchResult,
	wantCount int,
	want searchExpectations,
) {
	t.Helper()

	if len(results) != wantCount {
		t.Errorf("Searcher.Search() returned %d results, want %d", len(results), wantCount)
	}

	for _, result := range results {
		title := result.Node.Title()

		if wantScore, ok := want.scores[title]; ok {
			if result.Score != wantScore {
				t.Errorf("result[%q].Score = %v, want %v", title, result.Score, wantScore)
			}
		}

		if wantHighlight, ok := want.highlight[title]; ok {
			if result.Highlighted != wantHighlight {
				t.Errorf(
					"result[%q].Highlighted = %q, want %q",
					title,
					result.Highlighted,
					wantHighlight,
				)
			}
		}

		if wantSnippet, ok := want.snippets[title]; ok {
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
}
