package renderer

import (
	"strings"
	"testing"
)

func TestRenderEmptyInput(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	result, err := r.Render([]byte(""))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	// Empty input is valid; goldmark may produce empty HTML. The important
	// thing is that no error is returned.
	_ = result.HTML
}

func TestRenderWhitespaceOnly(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	result, err := r.Render([]byte("   \n\n\t\n   "))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	// Whitespace-only input is valid; goldmark may return an empty or
	// near-empty HTML body. We only assert that no error occurred.
	_ = result.HTML
}

func TestRenderMalformedMarkdownStillProducesHTML(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	// Unmatched code fence, unclosed brackets, etc.
	input := "# Title\n\n```go\nfunc f() {\n  // unclosed\n\n*unclosed list\n[bad link"

	result, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() returned error for malformed input: %v", err)
	}
	if !strings.Contains(string(result.HTML), "Title") {
		t.Errorf("expected 'Title' in output, got %s", result.HTML)
	}
}

func TestRenderUnicodeContent(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	input := "# Café\n\nNaïve façade with über-cool résumé."
	result, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(string(result.HTML), "Café") {
		t.Errorf("expected 'Café' in output, got %s", result.HTML)
	}
}

func TestRenderEmojiContent(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	input := "# Hello 👋 World 🌍"
	result, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(string(result.HTML), "Hello") {
		t.Errorf("expected 'Hello' in output, got %s", result.HTML)
	}
}

func TestRenderDeeplyNestedHeadings(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	input := strings.Repeat("#", 6) + " Deep Heading"
	result, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(string(result.HTML), "Deep Heading") {
		t.Errorf("expected 'Deep Heading' in output, got %s", result.HTML)
	}
}

func TestRenderVeryLongLine(t *testing.T) {
	t.Parallel()

	r := NewGoldmarkRenderer()
	longLine := strings.Repeat("word ", 1000)
	result, err := r.Render([]byte(longLine))
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
	input := "<script>alert('xss')</script>"
	result, err := r.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if strings.Contains(string(result.HTML), "<script>") {
		t.Errorf("expected script tag to be escaped, got %s", result.HTML)
	}
}
