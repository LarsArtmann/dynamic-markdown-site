// Package renderer provides diagram support for goldmark markdown parser.
package renderer

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// DiagramExtension is a goldmark extension for rendering diagrams.
type DiagramExtension struct {
	diagramRenderer *DiagramRenderer
}

// NewDiagramExtension creates a new diagram extension with the given renderer.
func NewDiagramExtension(dr *DiagramRenderer) *DiagramExtension {
	return &DiagramExtension{diagramRenderer: dr}
}

// Extend adds the diagram extension to goldmark.
func (de *DiagramExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&DiagramRendererNode{diagramRenderer: de.diagramRenderer}, 0),
	))
}

// DiagramRendererNode renders diagram nodes.
type DiagramRendererNode struct {
	diagramRenderer *DiagramRenderer
}

// RegisterFuncs registers the rendering functions.
func (r *DiagramRendererNode) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

// renderFencedCodeBlock renders fenced code blocks, handling d2 and mermaid specially.
func (r *DiagramRendererNode) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.FencedCodeBlock)

	// Get the language
	lang := string(n.Language(source))

	// Get the content
	var content bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		content.Write(line.Value(source))
	}

	switch lang {
	case "d2":
		return r.renderD2(w, content.String())
	case "mermaid":
		return r.renderMermaid(w, content.String())
	default:
		// Not a diagram, let default renderer handle it
		return ast.WalkContinue, nil
	}
}

// renderD2 renders a D2 diagram to SVG.
func (r *DiagramRendererNode) renderD2(w util.BufWriter, content string) (ast.WalkStatus, error) {
	svg, err := r.diagramRenderer.RenderD2(content)
	if err != nil {
		// If rendering fails, fall back to code block
		w.WriteString("<pre><code class=\"language-d2\">")
		w.Write([]byte(escapeHTML(content)))
		w.WriteString("</code></pre>")

		return ast.WalkContinue, nil
	}

	w.WriteString(`<div class="diagram d2-diagram">`)
	w.Write(svg)
	w.WriteString("</div>")

	return ast.WalkStop, nil
}

// renderMermaid renders a Mermaid diagram to HTML for client-side processing.
func (r *DiagramRendererNode) renderMermaid(w util.BufWriter, content string) (ast.WalkStatus, error) {
	html := RenderMermaidToHTML(content)
	w.WriteString(html)

	return ast.WalkStop, nil
}

// MermaidTransformer is a parser transformer that marks mermaid code blocks.
type MermaidTransformer struct{}

// Transform marks mermaid code blocks for special handling.
func (t *MermaidTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if cb, ok := n.(*ast.FencedCodeBlock); ok {
			lang := string(cb.Language(reader.Source()))
			if lang == "mermaid" {
				// Store info for renderer
				cb.SetBlankPreviousLines(true)
			}
		}

		return ast.WalkContinue, nil
	})
}
