// Package renderer handles markdown to HTML conversion.
package renderer

import (
	"bytes"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// GoldmarkRenderer implements markdown rendering using goldmark.
type GoldmarkRenderer struct {
	md              goldmark.Markdown
	diagramRenderer *DiagramRenderer
}

// NewGoldmarkRenderer creates a new configured goldmark renderer.
func NewGoldmarkRenderer() *GoldmarkRenderer {
	return NewGoldmarkRendererWithDiagrams(nil)
}

// NewGoldmarkRendererWithDiagrams creates a renderer with diagram support.
func NewGoldmarkRendererWithDiagrams(diagramRenderer *DiagramRenderer) *GoldmarkRenderer {
	extensions := []goldmark.Extender{
		// Standard extensions
		extension.Table,
		extension.Strikethrough,
		extension.Linkify,
		extension.TaskList,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,

		// Syntax highlighting with Chroma
		highlighting.NewHighlighting(
			highlighting.WithStyle("monokai"),
			highlighting.WithFormatOptions(
				chromahtml.WithLineNumbers(false),
			),
		),

		// Frontmatter support
		meta.Meta,
	}

	// Add diagram extension if renderer is provided
	if diagramRenderer != nil {
		extensions = append(extensions, NewDiagramExtension(diagramRenderer))
	}

	md := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return &GoldmarkRenderer{md: md, diagramRenderer: diagramRenderer}
}

// RenderResult contains the result of rendering markdown.
type RenderResult struct {
	HTML     domain.HTML
	TOC      []domain.TOCItem
	Metadata domain.Frontmatter
}

// Render converts markdown to HTML and extracts metadata.
func (r *GoldmarkRenderer) Render(source []byte) (RenderResult, error) {
	var buf bytes.Buffer

	context := parser.NewContext()

	// Parse first to get AST for TOC extraction
	doc := r.md.Parser().Parse(text.NewReader(source), parser.WithContext(context))

	// Extract TOC before rendering
	toc := extractTOCFromAST(doc, source)

	// Extract metadata
	metaData := meta.Get(context)
	metadata := extractFrontmatter(metaData)

	// Render to HTML (syntax highlighting is handled by goldmark extension)
	err := r.md.Renderer().Render(&buf, source, doc)
	if err != nil {
		return RenderResult{}, errors.Wrap(err, "render markdown")
	}

	return RenderResult{
		HTML:     domain.HTML(buf.String()),
		TOC:      toc,
		Metadata: metadata,
	}, nil
}

// extractFrontmatter converts goldmark meta to domain.Frontmatter.
func extractFrontmatter(metaData map[string]any) domain.Frontmatter {
	var fm domain.Frontmatter

	if title, ok := metaData["title"].(string); ok {
		fm.Title = title
	}

	if desc, ok := metaData["description"].(string); ok {
		fm.Description = desc
	}

	if author, ok := metaData["author"].(string); ok {
		fm.Author = author
	}

	if draft, ok := metaData["draft"].(bool); ok {
		fm.Draft = draft
	}

	if tags, ok := metaData["tags"].([]any); ok {
		fm.Tags = lo.FilterMap(tags, func(tag any, _ int) (string, bool) {
			tagStr, ok := tag.(string)

			return tagStr, ok
		})
	}

	return fm
}

// appendTOCTopLevel appends an item to the top-level items slice and updates tracking maps.
func appendTOCTopLevel(
	items *[]domain.TOCItem,
	item domain.TOCItem,
	n ast.Node,
	itemMap map[ast.Node]*domain.TOCItem,
	orderedItems *[]*domain.TOCItem,
) {
	*items = append(*items, item)
	ptr := &(*items)[len(*items)-1]
	itemMap[n] = ptr
	*orderedItems = append(*orderedItems, ptr)
}

// appendTOCChild appends an item as a child of parent and updates tracking maps.
func appendTOCChild(
	parent *domain.TOCItem,
	item domain.TOCItem,
	n ast.Node,
	itemMap map[ast.Node]*domain.TOCItem,
	orderedItems *[]*domain.TOCItem,
) {
	parent.Children = append(parent.Children, item)
	ptr := &parent.Children[len(parent.Children)-1]
	itemMap[n] = ptr
	*orderedItems = append(*orderedItems, ptr)
}

// extractTOCFromAST extracts table of contents from the parsed AST.
func extractTOCFromAST(doc ast.Node, source []byte) []domain.TOCItem {
	var items []domain.TOCItem
	// orderedItems tracks items in document order for proper parent finding
	var orderedItems []*domain.TOCItem

	itemMap := make(map[ast.Node]*domain.TOCItem)

	// Walk the AST to find headings
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		if heading.Level == 1 {
			return ast.WalkContinue, nil
		}

		text := extractHeadingText(heading, source)
		if text == "" {
			return ast.WalkContinue, nil
		}

		anchor := generateAnchorID(text)

		item := domain.TOCItem{
			Level:  uint(heading.Level),
			Title:  text,
			Anchor: anchor,
		}

		if heading.Level <= 2 || len(items) == 0 {
			appendTOCTopLevel(&items, item, n, itemMap, &orderedItems)
		} else {
			parent := findTOCParent(orderedItems, heading.Level)
			if parent != nil {
				appendTOCChild(parent, item, n, itemMap, &orderedItems)
			} else {
				appendTOCTopLevel(&items, item, n, itemMap, &orderedItems)
			}
		}

		return ast.WalkContinue, nil
	})
	if err != nil {
		return items
	}

	return items
}

// extractHeadingText extracts the text content from a heading node.
func extractHeadingText(heading *ast.Heading, source []byte) string {
	var buf bytes.Buffer

	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			buf.Write(textNode.Value(source))
		}
	}

	return strings.TrimSpace(buf.String())
}

// generateAnchorID creates a URL-friendly anchor ID from heading text.
func generateAnchorID(text string) string {
	// Convert to lowercase
	id := strings.ToLower(text)

	// Replace spaces and special chars with hyphens
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")

	// Remove non-alphanumeric chars except hyphens
	var result strings.Builder

	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// findTOCParent finds the most recent parent TOC item for a given heading level.
// It iterates backwards through ordered items to find the closest item with a lower level.
func findTOCParent(orderedItems []*domain.TOCItem, level int) *domain.TOCItem {
	// Iterate backwards to find the most recent item with a lower level
	for i := len(orderedItems) - 1; i >= 0; i-- {
		if orderedItems[i].Level < uint(level) {
			return orderedItems[i]
		}
	}

	return nil
}

// SimpleRenderer is a basic renderer without extensions, useful for previews.
type SimpleRenderer struct {
	md goldmark.Markdown
}

// NewSimpleRenderer creates a simple markdown renderer.
func NewSimpleRenderer() *SimpleRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
		),
	)

	return &SimpleRenderer{md: md}
}

// Render converts markdown to HTML without metadata extraction.
func (r *SimpleRenderer) Render(source []byte) (domain.HTML, error) {
	var buf bytes.Buffer

	err := r.md.Convert(source, &buf)
	if err != nil {
		return "", errors.Wrap(err, "render markdown")
	}

	return domain.HTML(buf.String()), nil
}
