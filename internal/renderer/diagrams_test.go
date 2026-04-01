package renderer

import (
	"strings"
	"testing"
)

func TestNewDiagramRenderer(t *testing.T) {
	t.Parallel()

	renderer, err := NewDiagramRenderer()
	if err != nil {
		t.Fatalf("failed to create diagram renderer: %v", err)
	}

	if renderer == nil {
		t.Fatal("renderer is nil")
	}

	if renderer.ruler == nil {
		t.Fatal("ruler is nil")
	}
}

func TestDiagramRendererRenderD2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantError bool
	}{
		{
			name:      "simple diagram",
			content:   "x -> y",
			wantError: false,
		},
		{
			name:      "diagram with shapes",
			content:   "direction: right\n\nA: {\n  shape: circle\n}\n\nB: {\n  shape: rectangle\n}\n\nA -> B",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			renderer, err := NewDiagramRenderer()
			if err != nil {
				t.Fatalf("failed to create diagram renderer: %v", err)
			}

			svg, err := renderer.RenderD2(tt.content)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(svg) == 0 {
				t.Error("expected non-empty SVG")
			}

			svgStr := string(svg)
			if !strings.Contains(svgStr, "<svg") {
				t.Error("expected SVG to contain <svg tag")
			}
		})
	}
}

func TestRenderMermaidToHTML(t *testing.T) {
	t.Parallel()

	content := "graph TD;\n    A-->B;"
	result := RenderMermaidToHTML(content)

	if !strings.Contains(result, `class="mermaid"`) {
		t.Error("expected mermaid class in output")
	}

	if !strings.Contains(result, "graph TD") {
		t.Error("expected content in output")
	}

	if !strings.Contains(result, "</pre>") {
		t.Error("expected closing </pre> tag")
	}
}

func TestRenderMermaidDiagramThroughGoldmark(t *testing.T) {
	t.Parallel()

	diagramRenderer, err := NewDiagramRenderer()
	if err != nil {
		t.Fatalf("failed to create diagram renderer: %v", err)
	}

	renderer := NewGoldmarkRendererWithDiagrams(diagramRenderer)

	tests := []struct {
		name             string
		input            string
		shouldContain    string
		shouldNotContain string
		expectHasMermaid bool
	}{
		{
			name:             "simple mermaid flowchart",
			input:            "```mermaid\ngraph TD;\n    A-->B;\n```",
			shouldContain:    `<pre class="mermaid">`,
			shouldNotContain: "chroma",
			expectHasMermaid: true,
		},
		{
			name:             "mermaid with backticks in content",
			input:            "```mermaid\ngraph TD\n    A[\"`text`\"]-->B\n```",
			shouldContain:    `<pre class="mermaid">`,
			expectHasMermaid: true,
		},
		{
			name:             "mermaid with HTML-like content",
			input:            "```mermaid\ngraph TD\n    A[<div>]-->B\n```",
			shouldContain:    "&lt;div&gt;",
			expectHasMermaid: true,
		},
		{
			name:             "regular code block unaffected",
			input:            "```go\nfmt.Println(\"hello\")\n```",
			shouldNotContain: `<pre class="mermaid">`,
			expectHasMermaid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := renderer.Render([]byte(tt.input))
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}

			html := string(result.HTML)

			if tt.shouldContain != "" && !strings.Contains(html, tt.shouldContain) {
				t.Errorf("expected HTML to contain %q, got: %s", tt.shouldContain, html)
			}

			if tt.shouldNotContain != "" && strings.Contains(html, tt.shouldNotContain) {
				t.Errorf("expected HTML NOT to contain %q, got: %s", tt.shouldNotContain, html)
			}

			if result.HasMermaid != tt.expectHasMermaid {
				t.Errorf("HasMermaid = %v, want %v", result.HasMermaid, tt.expectHasMermaid)
			}
		})
	}
}

func TestRenderD2DiagramThroughGoldmark(t *testing.T) {
	t.Parallel()

	diagramRenderer, err := NewDiagramRenderer()
	if err != nil {
		t.Fatalf("failed to create diagram renderer: %v", err)
	}

	renderer := NewGoldmarkRendererWithDiagrams(diagramRenderer)

	input := "```d2\nx -> y\n```"

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(result.HTML)

	if !strings.Contains(html, `<div class="diagram d2-diagram">`) {
		t.Errorf("expected HTML to contain d2 diagram div, got: %s", html)
	}

	if !strings.Contains(html, "<svg") {
		t.Errorf("expected HTML to contain SVG, got: %s", html)
	}

	if result.HasMermaid {
		t.Error("expected HasMermaid to be false for D2-only content")
	}
}

func TestMixedCodeBlocksAndDiagrams(t *testing.T) {
	t.Parallel()

	diagramRenderer, err := NewDiagramRenderer()
	if err != nil {
		t.Fatalf("failed to create diagram renderer: %v", err)
	}

	renderer := NewGoldmarkRendererWithDiagrams(diagramRenderer)

	input := "# Document\n\n```go\nfmt.Println(\"hello\")\n```\n\n```mermaid\ngraph TD;\n    A-->B;\n```\n\n```d2\nx -> y\n```"

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(result.HTML)

	if !strings.Contains(html, `<pre class="mermaid">`) {
		t.Errorf("expected mermaid diagram in HTML, got: %s", html)
	}

	if !strings.Contains(html, "d2-diagram") && !strings.Contains(html, "language-d2") {
		t.Errorf("expected D2 diagram (or fallback) in HTML, got: %s", html)
	}

	if !strings.Contains(html, "color:#") {
		t.Error("expected syntax highlighting for Go code block")
	}

	if !result.HasMermaid {
		t.Error("expected HasMermaid to be true")
	}
}
