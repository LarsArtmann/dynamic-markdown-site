package renderer

import (
	"context"
	"fmt"
	"html"

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

// d2Error wraps an error with D2 rendering context.
func d2Error(err error, ctx, content string) error {
	return errors.Wrapf(err, "%s: %s", ctx, truncateForLog(content))
}

// RenderD2 renders D2 content to SVG.
func (r *DiagramRenderer) RenderD2(content string) ([]byte, error) {
	ctx := log.WithDefault(context.Background())

	layoutResolver := func(_ string) (d2graph.LayoutGraph, error) {
		return d2dagrelayout.DefaultLayout, nil
	}

	compileOpts := &d2lib.CompileOptions{
		LayoutResolver: layoutResolver,
		Ruler:          r.ruler,
		UTF16Pos:       false,
	}

	sketch := false
	center := false
	themeID := int64(0)
	darkThemeID := int64(0)
	pad := int64(10)
	scale := 1.0
	noXMLTag := false
	salt := ""
	omitVersion := false

	renderOpts := &d2svg.RenderOpts{
		Pad:         &pad,
		Sketch:      &sketch,
		Center:      &center,
		ThemeID:     &themeID,
		DarkThemeID: &darkThemeID,
		Font:        "",
		Scale:       &scale,
		MasterID:    "",
		NoXMLTag:    &noXMLTag,
		Salt:        &salt,
		OmitVersion: &omitVersion,
	}

	diagram, _, err := d2lib.Compile(ctx, content, compileOpts, renderOpts)
	if err != nil {
		return nil, d2Error(err, "compile D2 diagram", content)
	}

	svg, err := d2svg.Render(diagram, renderOpts)
	if err != nil {
		return nil, d2Error(err, "render D2 diagram to SVG", content)
	}

	return svg, nil
}

// RenderMermaidToHTML returns HTML for client-side Mermaid rendering.
func RenderMermaidToHTML(content string) string {
	escaped := html.EscapeString(content)

	return fmt.Sprintf(`<pre class="mermaid">%s</pre>`, escaped)
}
