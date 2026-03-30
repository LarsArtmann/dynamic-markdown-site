// Package renderer handles diagram rendering for D2 and Mermaid.
package renderer

import (
	"bytes"
	"context"
	"fmt"
	"regexp"

	"github.com/cockroachdb/errors"
	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// DiagramRenderer handles rendering of D2 and Mermaid diagrams.
type DiagramRenderer struct {
	ruler *textmeasure.Ruler
}

// NewDiagramRenderer creates a new diagram renderer.
func NewDiagramRenderer() (*DiagramRenderer, error) {
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, errors.Wrap(err, "create text ruler for diagrams")
	}

	return &DiagramRenderer{ruler: ruler}, nil
}

// DiagramType represents the type of diagram.
type DiagramType int

const (
	// DiagramTypeD2 is a D2 diagram.
	DiagramTypeD2 DiagramType = iota
	// DiagramTypeMermaid is a Mermaid diagram.
	DiagramTypeMermaid
)

// String returns the string representation of the diagram type.
func (d DiagramType) String() string {
	switch d {
	case DiagramTypeD2:
		return "d2"
	case DiagramTypeMermaid:
		return "mermaid"
	default:
		return "unknown"
	}
}

// DetectedDiagram represents a detected diagram in markdown content.
type DetectedDiagram struct {
	Type    DiagramType
	Content string
	Start   int
	End     int
}

// CodeBlock represents a code block with its language and content.
type CodeBlock struct {
	Language string
	Content  string
	Start    int
	End      int
}

var codeBlockRegex = regexp.MustCompile("```([a-zA-Z0-9_-]*)\\s*\\n([^`]*?)\\n?```")

// DetectDiagrams finds D2 and Mermaid diagrams in markdown content.
func DetectDiagrams(content string) []DetectedDiagram {
	var diagrams []DetectedDiagram

	matches := codeBlockRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		// match[0], match[1] = full match
		// match[2], match[3] = language
		// match[4], match[5] = content
		lang := content[match[2]:match[3]]
		diagramContent := content[match[4]:match[5]]

		var diagramType DiagramType
		switch lang {
		case "d2":
			diagramType = DiagramTypeD2
		case "mermaid":
			diagramType = DiagramTypeMermaid
		default:
			continue
		}

		diagrams = append(diagrams, DetectedDiagram{
			Type:    diagramType,
			Content: diagramContent,
			Start:   match[0],
			End:     match[1],
		})
	}

	return diagrams
}

// RenderD2 renders D2 content to SVG.
func (r *DiagramRenderer) RenderD2(content string) ([]byte, error) {
	ctx := log.WithDefault(context.Background())

	layoutResolver := func(engine string) (d2graph.LayoutGraph, error) {
		return d2dagrelayout.DefaultLayout, nil
	}

	compileOpts := &d2lib.CompileOptions{
		LayoutResolver: layoutResolver,
		Ruler:          r.ruler,
	}

	pad := int64(10)
	renderOpts := &d2svg.RenderOpts{
		Pad: &pad,
	}

	diagram, _, err := d2lib.Compile(ctx, content, compileOpts, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "compile D2 diagram")
	}

	svg, err := d2svg.Render(diagram, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "render D2 diagram to SVG")
	}

	return svg, nil
}

// RenderMermaidToHTML returns HTML for client-side Mermaid rendering.
func RenderMermaidToHTML(content string) string {
	// Escape HTML entities to prevent XSS
	escaped := escapeHTML(content)
	return fmt.Sprintf(`<pre class="mermaid">%s</pre>`, escaped)
}

// escapeHTML escapes special HTML characters.
func escapeHTML(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// ProcessMarkdown detects and renders diagrams in markdown content.
// Returns the processed markdown with diagrams replaced.
func (r *DiagramRenderer) ProcessMarkdown(content string) (string, error) {
	diagrams := DetectDiagrams(content)
	if len(diagrams) == 0 {
		return content, nil
	}

	// Process from end to start to preserve positions
	result := content
	for i := len(diagrams) - 1; i >= 0; i-- {
		d := diagrams[i]
		var replacement string

		switch d.Type {
		case DiagramTypeD2:
			svg, err := r.RenderD2(d.Content)
			if err != nil {
				// If rendering fails, keep original code block
				replacement = fmt.Sprintf("```d2\n%s\n```", d.Content)
			} else {
				replacement = fmt.Sprintf("<div class=\"diagram d2-diagram\">%s</div>", string(svg))
			}
		case DiagramTypeMermaid:
			replacement = RenderMermaidToHTML(d.Content)
		}

		result = result[:d.Start] + replacement + result[d.End:]
	}

	return result, nil
}

// HasMermaidDiagrams checks if content contains Mermaid diagrams.
func HasMermaidDiagrams(content string) bool {
	diagrams := DetectDiagrams(content)
	for _, d := range diagrams {
		if d.Type == DiagramTypeMermaid {
			return true
		}
	}
	return false
}
