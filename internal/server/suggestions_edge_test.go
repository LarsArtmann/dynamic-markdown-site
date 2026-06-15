package server

import (
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func TestFindSuggestions_EmptyRequestedReturnsNil(t *testing.T) {
	t.Parallel()

	paths := []domain.URLPath{pathMust(t, "/a"), pathMust(t, "/b")}
	if got := findSuggestions("", paths, 5); got != nil {
		t.Errorf("expected nil for empty request, got %v", got)
	}
}

func TestFindSuggestions_EmptyPathsReturnsNil(t *testing.T) {
	t.Parallel()

	if got := findSuggestions("/foo", nil, 5); got != nil {
		t.Errorf("expected nil for empty paths, got %v", got)
	}
}

func TestFindSuggestions_ExactMatchIsExcluded(t *testing.T) {
	t.Parallel()

	assertNoSuggestions(t, "/guide", "/guide")
}

func TestFindSuggestions_CaseInsensitiveExactMatchExcluded(t *testing.T) {
	t.Parallel()

	assertNoSuggestions(t, "/Guide", "/guide")
}

func TestFindSuggestions_TypoYieldsSuggestion(t *testing.T) {
	t.Parallel()

	paths := []domain.URLPath{
		pathMust(t, "/guide"),
		pathMust(t, "/intro"),
		pathMust(t, "/reference"),
	}
	got := findSuggestions("/guied", paths, 5)
	if len(got) == 0 {
		t.Fatal("expected at least one suggestion for /guied")
	}
	if got[0].Path.String() != "/guide" {
		t.Errorf("top suggestion = %q, want /guide", got[0].Path.String())
	}
}

func TestFindSuggestions_PrefixMatchScoresHigher(t *testing.T) {
	t.Parallel()

	// Both /guide and /guidebook are very similar to /guider; the
	// /guidebook variant wins because it has a shared prefix with /guider.
	paths := []domain.URLPath{
		pathMust(t, "/guidebook"),
		pathMust(t, "/something-totally-different"),
	}
	got := findSuggestions("/guider", paths, 5)
	if len(got) == 0 {
		t.Fatal("expected at least one suggestion")
	}
	// Top suggestion should be the prefix-matched one.
	if got[0].Path.String() != "/guidebook" {
		t.Errorf("top suggestion = %q, want /guidebook", got[0].Path.String())
	}
}

func TestFindSuggestions_RespectsLimit(t *testing.T) {
	t.Parallel()

	paths := []domain.URLPath{
		pathMust(t, "/a"),
		pathMust(t, "/b"),
		pathMust(t, "/c"),
		pathMust(t, "/d"),
	}
	got := findSuggestions("/a", paths, 2)
	if len(got) > 2 {
		t.Errorf("got %d suggestions, want <= 2", len(got))
	}
}

func TestFindSuggestions_ThresholdDropsPoorMatches(t *testing.T) {
	t.Parallel()

	assertNoSuggestions(t, "/a", "/zzzzzzzzzzzz")
}

// pathMust is a tiny helper to construct a URLPath or fail the test.
func pathMust(t *testing.T, s string) domain.URLPath {
	t.Helper()
	p, err := domain.NewURLPath(s)
	if err != nil {
		t.Fatalf("invalid path %q: %v", s, err)
	}

	return p
}

// assertNoSuggestions fails the test if findSuggestions returns any results
// for the given requested/available path pair. It exists to make "exact match
// excluded" and "threshold drops poor match" tests read in one line.
func assertNoSuggestions(t *testing.T, requested, available string) {
	t.Helper()

	paths := []domain.URLPath{pathMust(t, available)}
	if got := findSuggestions(requested, paths, 5); len(got) != 0 {
		t.Errorf("expected no suggestions for %q with paths %v, got %v", requested, paths, got)
	}
}
