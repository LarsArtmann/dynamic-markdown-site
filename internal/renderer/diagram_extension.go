// Package renderer provides diagram support for goldmark markdown parser.
package renderer

import (
	"bytes"

	"github.com/cockroachdb/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

const (
	diagramTypeD2      = "d2"
	diagramTypeMermaid = "mermaid"
	maxLogContentLen   = 100
)

// truncateForLog truncates content for safe logging.
func truncateForLog(content string) string {
	if len(content) <= maxLogContentLen {
		return content
	}
	return content[:maxLogContentLen] + "..."
}

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
func (r *DiagramRendererNode) renderFencedCodeBlock(
	w util.BufWriter,
	source []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n, ok := node.(*ast.FencedCodeBlock)
	if !ok {
		return ast.WalkContinue, nil
	}

	// Get the language
	lang := string(n.Language(source))

	// Get the content
	var content bytes.Buffer
	for i := range n.Lines().Len() {
		line := n.Lines().At(i)
		if _, err := content.Write(line.Value(source)); err != nil {
			return ast.WalkContinue, errors.Wrapf(
				err,
				"write fence code line (lang: %s, partial: %s)",
				lang,
				truncateForLog(content.String()),
			)
		}
	}

	switch lang {
	case diagramTypeD2:
		return r.renderD2(w, content.String())
	case diagramTypeMermaid:
		return r.renderMermaid(w, content.String())
	default:
		// Not a diagram, let default renderer handle it
		return ast.WalkContinue, nil
	}
}

// writeStrings writes multiple strings to the writer and returns an error if any write fails.
func writeStrings(w util.BufWriter, errMsg string, parts ...string) error {
	for _, p := range parts {
		if _, err := w.WriteString(p); err != nil {
			return errors.Wrapf(err, "%s (writing %d parts)", errMsg, len(parts))
		}
	}
	return nil
}

// wrapWriteError wraps a write error with context including truncated content.
func wrapWriteError(err error, msg string, content string) error {
	return errors.Wrapf(err, "%s: %s", msg, truncateForLog(content))
}

// renderD2 renders a D2 diagram to SVG.
func (r *DiagramRendererNode) renderD2(w util.BufWriter, content string) (ast.WalkStatus, error) {
	svg, err := r.diagramRenderer.RenderD2(content)
	if err != nil {
		htmlContent := escapeHTML(content)
		if writeErr := writeStrings(
			w,
			"write D2 fallback",
			"<pre><code class=\"language-d2\">",
			htmlContent,
			"</code></pre>",
		); writeErr != nil {
			return ast.WalkContinue, wrapWriteError(
				writeErr,
				"write D2 fallback for diagram",
				content,
			)
		}
		return ast.WalkContinue, nil
	}

	if writeErr := writeStrings(
		w,
		"write D2 output",
		`<div class="diagram d2-diagram">`,
	); writeErr != nil {
		return ast.WalkStop, wrapWriteError(writeErr, "write D2 div start", content)
	}
	if _, writeErr := w.Write(svg); writeErr != nil {
		return ast.WalkStop, wrapWriteError(writeErr, "write D2 SVG", content)
	}
	if _, writeErr := w.WriteString("</div>"); writeErr != nil {
		return ast.WalkStop, wrapWriteError(writeErr, "write D2 div end", content)
	}

	return ast.WalkStop, nil
}

// renderMermaid renders a Mermaid diagram to HTML for client-side processing.
func (r *DiagramRendererNode) renderMermaid(
	w util.BufWriter,
	content string,
) (ast.WalkStatus, error) {
	html := RenderMermaidToHTML(content)
	if _, err := w.WriteString(html); err != nil {
		return ast.WalkStop, wrapWriteError(err, "write mermaid HTML", content)
	}

	return ast.WalkStop, nil
}
