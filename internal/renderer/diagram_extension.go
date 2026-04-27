package renderer

import (
	"bytes"
	"html"

	"github.com/cockroachdb/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const (
	diagramTypeD2      = "d2"
	diagramTypeMermaid = "mermaid"
	maxLogContentLen   = 100

	diagramNodeKind ast.NodeKind = 10000
)

// truncateForLog truncates content for safe logging.
func truncateForLog(content string) string {
	if len(content) <= maxLogContentLen {
		return content
	}
	return content[:maxLogContentLen] + "..."
}

// diagramNode is a custom AST node representing a diagram (mermaid or d2).
// It replaces FencedCodeBlock nodes during the AST transform phase,
// avoiding priority conflicts with the Chroma syntax highlighting renderer.
type diagramNode struct {
	ast.BaseBlock
	language string
	content  string
}

// Kind returns the unique node kind for diagram nodes.
func (n *diagramNode) Kind() ast.NodeKind {
	return diagramNodeKind
}

// Dump implements ast.Node for debugging.
func (n *diagramNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"language": n.language,
		"content":  truncateForLog(n.content),
	}, nil)
}

// hasMermaidKey is a parser context key for tracking mermaid diagram presence.
var hasMermaidKey = parser.NewContextKey()

// diagramTransformer converts mermaid/d2 fenced code blocks into diagramNodes
// during the parse phase, before rendering begins.
type diagramTransformer struct{}

// Transform walks the AST and replaces mermaid/d2 fenced code blocks with diagramNodes.
// Nodes are collected during the walk and replaced afterwards to avoid unsafe mutation during iteration.
// It also sets a flag on the parser context when mermaid diagrams are detected.
func (t *diagramTransformer) Transform(
	node *ast.Document,
	reader text.Reader,
	pctx parser.Context,
) {
	source := reader.Source()
	var replacements []nodeReplacement

	walkAST(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		fenced, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		lang := string(fenced.Language(source))
		if lang != diagramTypeD2 && lang != diagramTypeMermaid {
			return ast.WalkContinue, nil
		}

		var buf bytes.Buffer
		for i := range fenced.Lines().Len() {
			line := fenced.Lines().At(i)
			if _, err := buf.Write(line.Value(source)); err != nil {
				return ast.WalkContinue, errors.Wrapf(
					err,
					"read fenced code line (lang: %s)",
					lang,
				)
			}
		}

		if lang == diagramTypeMermaid {
			pctx.Set(hasMermaidKey, true)
		}

		diagram := &diagramNode{
			BaseBlock: ast.BaseBlock{},
			language:  lang,
			content:   buf.String(),
		}

		replacements = append(replacements, nodeReplacement{
			parent: fenced.Parent(),
			old:    fenced,
			new:    diagram,
		})

		return ast.WalkContinue, nil
	})

	applyReplacements(replacements)
}

// DiagramExtension is a goldmark extension for rendering diagrams.
// It uses an AST transformer to intercept mermaid/d2 code blocks before
// the Chroma syntax highlighter processes them.
type DiagramExtension struct {
	diagramRenderer *DiagramRenderer
}

// NewDiagramExtension creates a new diagram extension with the given renderer.
func NewDiagramExtension(dr *DiagramRenderer) *DiagramExtension {
	return &DiagramExtension{diagramRenderer: dr}
}

// Extend adds the diagram extension to goldmark.
func (de *DiagramExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(&diagramTransformer{}, 100),
	))

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&diagramNodeRenderer{diagramRenderer: de.diagramRenderer}, 1),
	))
}

// diagramNodeRenderer renders diagramNode AST nodes to HTML.
type diagramNodeRenderer struct {
	diagramRenderer *DiagramRenderer
}

// RegisterFuncs registers the rendering function for diagram nodes.
func (r *diagramNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(diagramNodeKind, r.renderDiagram)
}

// renderDiagram renders a diagram node to HTML.
func (r *diagramNodeRenderer) renderDiagram(
	w util.BufWriter,
	_ []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	diagram, ok := node.(*diagramNode)
	if !ok {
		return ast.WalkContinue, nil
	}

	switch diagram.language {
	case diagramTypeD2:
		return r.renderD2(w, diagram.content)
	case diagramTypeMermaid:
		return r.renderMermaid(w, diagram.content)
	default:
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
func wrapWriteError(err error, msg, content string) error {
	return errors.Wrapf(err, "%s: %s", msg, truncateForLog(content))
}

// renderD2 renders a D2 diagram to SVG.
func (r *diagramNodeRenderer) renderD2(w util.BufWriter, content string) (ast.WalkStatus, error) {
	svg, err := r.diagramRenderer.RenderD2(content)
	if err != nil {
		htmlContent := html.EscapeString(content)
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
		return ast.WalkContinue, wrapWriteError(writeErr, "write D2 div end", content)
	}

	return ast.WalkContinue, nil
}

// renderMermaid renders a Mermaid diagram to HTML for client-side processing.
func (r *diagramNodeRenderer) renderMermaid(
	w util.BufWriter,
	content string,
) (ast.WalkStatus, error) {
	html := RenderMermaidToHTML(content)
	if _, err := w.WriteString(html); err != nil {
		return ast.WalkContinue, wrapWriteError(err, "write mermaid HTML", content)
	}

	return ast.WalkContinue, nil
}
