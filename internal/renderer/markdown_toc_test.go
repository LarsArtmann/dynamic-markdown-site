package renderer

import (
	"testing"
)

func TestRenderTOC(t *testing.T) {
	t.Parallel()

	renderer := NewGoldmarkRenderer()

	input := `# Top Level

## Section One

Content.

## Section Two

More content.

### Subsection

Detailed content.
`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if len(result.TOC) == 0 {
		t.Error("expected TOC to be generated")
	}

	// Top-level (h1) should be excluded.
	for _, item := range result.TOC {
		if item.Level < 2 {
			t.Errorf("TOC should exclude h1, found level %d", item.Level)
		}
	}
}

func TestRenderTOCEmpty(t *testing.T) {
	t.Parallel()

	renderer := NewGoldmarkRenderer()

	input := "Plain text with no headings.\n"
	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if len(result.TOC) != 0 {
		t.Errorf("expected empty TOC, got %d items", len(result.TOC))
	}
}

func TestGenerateAnchorID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "Section One", "section-one"},
		{"with special chars", "What's Next?", "whats-next"},
		{"with code", "Using `go.mod`", "using-gomod"},
		{"multiple spaces", "Multiple   Spaces", "multiple---spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateAnchorID(tt.input)
			if got != tt.expected {
				t.Errorf("generateAnchorID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
