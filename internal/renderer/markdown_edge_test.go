package renderer

import (
	"strings"
	"testing"
)

func TestRenderEmptyInput(t *testing.T) {
	t.Parallel()

	// Empty input is valid; goldmark may produce empty HTML. The important
	// thing is that no error is returned.
	renderNoError(t, NewGoldmarkRenderer(), "")
}

func TestRenderWhitespaceOnly(t *testing.T) {
	t.Parallel()

	// Whitespace-only input is valid; goldmark may return an empty or
	// near-empty HTML body. We only assert that no error occurred.
	renderNoError(t, NewGoldmarkRenderer(), "   \n\n\t\n   ")
}

func TestRenderMalformedMarkdownStillProducesHTML(t *testing.T) {
	t.Parallel()

	// Unmatched code fence, unclosed brackets, etc.
	input := "# Title\n\n```go\nfunc f() {\n  // unclosed\n\n*unclosed list\n[bad link"

	renderContains(t, NewGoldmarkRenderer(), input, "Title", "returned error for malformed input")
}

func TestRenderUnicodeContent(t *testing.T) {
	t.Parallel()

	renderContains(
		t,
		NewGoldmarkRenderer(),
		"# Café\n\nNaïve façade with über-cool résumé.",
		"Café",
		"error",
	)
}

func TestRenderEmojiContent(t *testing.T) {
	t.Parallel()

	renderContains(t, NewGoldmarkRenderer(), "# Hello 👋 World 🌍", "Hello", "error")
}

func TestRenderDeeplyNestedHeadings(t *testing.T) {
	t.Parallel()

	renderContains(
		t,
		NewGoldmarkRenderer(),
		strings.Repeat("#", 6)+" Deep Heading",
		"Deep Heading",
		"error",
	)
}

func TestRenderVeryLongLine(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	result, err := r.Render([]byte(strings.Repeat("word ", 1000)))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if len(result.HTML) == 0 {
		t.Error("expected non-empty HTML for long line")
	}
}

func TestRenderHTMLInMarkdownIsEscaped(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	result, err := r.Render([]byte("<script>alert('xss')</script>"))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if strings.Contains(string(result.HTML), "<script>") {
		t.Errorf("expected script tag to be escaped, got %s", result.HTML)
	}
}
