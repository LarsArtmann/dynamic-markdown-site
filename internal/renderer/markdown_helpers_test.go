package renderer

import (
	"strings"
	"testing"
)

// renderContains is a render-and-assert helper: it runs r.Render on input,
// fails the test on any error, and asserts that want is a substring of the
// resulting HTML. errMessage is prepended to the fatal error so callers can
// describe inputs that should not error (e.g. "for malformed input").
func renderContains(t *testing.T, r *GoldmarkRenderer, input, want, errMessage string) {
	t.Helper()

	result, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() %s: %v", errMessage, err)
	}

	if !strings.Contains(string(result.HTML), want) {
		t.Errorf("expected %q in output, got %s", want, result.HTML)
	}
}

// renderNoError is a render-only helper: it asserts that r.Render completes
// without error and returns the resulting HTML. Use this when the test only
// cares about parseability (e.g. empty or whitespace input).
func renderNoError(t *testing.T, r *GoldmarkRenderer, input string) {
	t.Helper()

	_, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
}
