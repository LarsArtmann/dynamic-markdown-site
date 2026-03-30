package renderer

import (
	"testing"
)

func TestDetectDiagrams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected int
		types    []DiagramType
	}{
		{
			name:     "no diagrams",
			content:  "# Hello\n\nThis is just text.",
			expected: 0,
		},
		{
			name: "single D2 diagram",
			content: "# Diagram\n\n```d2\nx -> y\n```",
			expected: 1,
			types:    []DiagramType{DiagramTypeD2},
		},
		{
			name: "single Mermaid diagram",
			content: "# Flowchart\n\n```mermaid\ngraph TD;\n    A-->B;\n```",
			expected: 1,
			types:    []DiagramType{DiagramTypeMermaid},
		},
		{
			name: "mixed diagrams",
			content: `# Mixed Diagrams

## D2 Diagram

` + "```d2\nx -> y\n```" + `

## Mermaid Diagram

` + "```mermaid\ngraph TD;\n    A-->B;\n```",
			expected: 2,
			types:    []DiagramType{DiagramTypeD2, DiagramTypeMermaid},
		},
		{
			name: "code block with different language",
			content: "# Code\n\n```go\nfmt.Println(\"hello\")\n```",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diagrams := DetectDiagrams(tt.content)

			if len(diagrams) != tt.expected {
				t.Errorf("expected %d diagrams, got %d", tt.expected, len(diagrams))
			}

			for i, expectedType := range tt.types {
				if i < len(diagrams) && diagrams[i].Type != expectedType {
					t.Errorf("diagram %d: expected type %v, got %v", i, expectedType, diagrams[i].Type)
				}
			}
		})
	}
}

func TestHasMermaidDiagrams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "no mermaid",
			content:  "# Hello\n\n```d2\nx -> y\n```",
			expected: false,
		},
		{
			name:     "has mermaid",
			content:  "# Hello\n\n```mermaid\ngraph TD;\n```",
			expected: true,
		},
		{
			name:     "both types",
			content:  "```d2\nx -> y\n```\n\n```mermaid\ngraph TD;\n```",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := HasMermaidDiagrams(tt.content)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDiagramTypeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dt       DiagramType
		expected string
	}{
		{DiagramTypeD2, "d2"},
		{DiagramTypeMermaid, "mermaid"},
		{DiagramType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := tt.dt.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

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

	renderer, err := NewDiagramRenderer()
	if err != nil {
		t.Fatalf("failed to create diagram renderer: %v", err)
	}

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

			// Check that it contains SVG tag somewhere (D2 may add XML header)
			svgStr := string(svg)
			if !contains(svgStr, "<svg") {
				t.Error("expected SVG to contain <svg tag")
			}
		})
	}
}

func TestRenderMermaidToHTML(t *testing.T) {
	t.Parallel()

	content := "graph TD;\n    A-->B;"
	result := RenderMermaidToHTML(content)

	// Should contain mermaid class
	if !contains(result, `class="mermaid"`) {
		t.Error("expected mermaid class in output")
	}

	// Should contain the content
	if !contains(result, "graph TD") {
		t.Error("expected content in output")
	}

	// The mermaid content itself should be preserved (mermaid handles its own syntax)
	// We just verify the structure is correct
	if !contains(result, "</pre>") {
		t.Error("expected closing </pre> tag")
	}
}

func TestEscapeHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"<", "&lt;"},
		{">", "&gt;"},
		{"&", "&amp;"},
		{`"`, "&quot;"},
		{"hello", "hello"},
		{"<div>", "&lt;div&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := escapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
