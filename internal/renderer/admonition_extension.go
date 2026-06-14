package renderer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var alertPattern = regexp.MustCompile(`\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION|CRITICAL)\]\s*`)

type alertKind string

const (
	alertNote      alertKind = "note"
	alertTip       alertKind = "tip"
	alertImportant alertKind = "important"
	alertWarning   alertKind = "warning"
	alertCaution   alertKind = "caution"
	alertCritical  alertKind = "critical"
)

const (
	alertTitleNote      = "Note"
	alertTitleTip       = "Tip"
	alertTitleImportant = "Important"
	alertTitleWarning   = "Warning"
	alertTitleCaution   = "Caution"
	alertTitleCritical  = "Critical"
)

var alertTitles = map[alertKind]string{
	alertNote:      alertTitleNote,
	alertTip:       alertTitleTip,
	alertImportant: alertTitleImportant,
	alertWarning:   alertTitleWarning,
	alertCaution:   alertTitleCaution,
	alertCritical:  alertTitleCritical,
}

// AlertTitles returns the display title for an alert kind.
func (k alertKind) Title() string {
	return alertTitles[k]
}

func parseAlertKind(raw string) (alertKind, bool) {
	switch strings.ToUpper(raw) {
	case "NOTE":
		return alertNote, true
	case "TIP":
		return alertTip, true
	case "IMPORTANT":
		return alertImportant, true
	case "WARNING":
		return alertWarning, true
	case "CAUTION":
		return alertCaution, true
	case "CRITICAL":
		return alertCritical, true
	default:
		return "", false
	}
}

const (
	admonitionNodeKind ast.NodeKind = 10001
)

type admonitionNode struct {
	ast.BaseBlock

	kind alertKind
}

func (n *admonitionNode) Kind() ast.NodeKind {
	return admonitionNodeKind
}

func (n *admonitionNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"kind": string(n.kind),
	}, nil)
}

type admonitionTransformer struct{}

func (t *admonitionTransformer) Transform(
	node *ast.Document,
	reader text.Reader,
	_ parser.Context,
) {
	source := reader.Source()

	var replacements []nodeReplacement

	walkAST(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		blockquote, ok := n.(*ast.Blockquote)
		if !ok {
			return ast.WalkContinue, nil
		}

		firstParagraph := findFirstParagraph(blockquote)
		if firstParagraph == nil {
			return ast.WalkContinue, nil
		}

		alertType, consumed := detectAlertInParagraph(firstParagraph, source)
		if alertType == "" {
			return ast.WalkContinue, nil
		}

		kind, ok := parseAlertKind(alertType)
		if !ok {
			return ast.WalkContinue, nil
		}

		stripAlertNodes(firstParagraph, source, consumed)

		admonition := &admonitionNode{
			BaseBlock: ast.BaseBlock{},
			kind:      kind,
		}

		child := blockquote.FirstChild()
		for child != nil {
			next := child.NextSibling()
			blockquote.RemoveChild(blockquote, child)
			admonition.AppendChild(admonition, child)
			child = next
		}

		replacements = append(replacements, nodeReplacement{
			parent: blockquote.Parent(),
			old:    blockquote,
			new:    admonition,
		})

		return ast.WalkContinue, nil
	})

	applyReplacements(replacements)
}

func findFirstParagraph(blockquote *ast.Blockquote) *ast.Paragraph {
	for child := blockquote.FirstChild(); child != nil; child = child.NextSibling() {
		if p, ok := child.(*ast.Paragraph); ok {
			return p
		}
	}

	return nil
}

type textSpan struct {
	node  ast.Node
	value string
}

func collectTextSpans(para *ast.Paragraph, source []byte) []textSpan {
	var spans []textSpan

	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			spans = append(spans, textSpan{node: n, value: string(n.Value(source))})
		case *ast.String:
			spans = append(spans, textSpan{node: n, value: string(n.Value)})
		}
	}

	return spans
}

func detectAlertInParagraph(para *ast.Paragraph, source []byte) (string, int) {
	spans := collectTextSpans(para, source)
	if len(spans) == 0 {
		return "", 0
	}

	var fullText strings.Builder
	for _, s := range spans {
		fullText.WriteString(s.value)
	}

	matches := alertPattern.FindStringSubmatchIndex(fullText.String())
	if matches == nil {
		return "", 0
	}

	alertType := fullText.String()[matches[2]:matches[3]]
	consumed := matches[1]

	return alertType, consumed
}

func stripAlertNodes(para *ast.Paragraph, source []byte, consumed int) {
	spans := collectTextSpans(para, source)

	remaining := consumed

	var (
		toRemove     []ast.Node
		toReplace    ast.Node
		replaceValue string
	)

	for _, s := range spans {
		if remaining <= 0 {
			break
		}

		if remaining >= len(s.value) {
			toRemove = append(toRemove, s.node)
			remaining -= len(s.value)
		} else {
			trimmed := s.value[remaining:]
			toReplace = s.node
			replaceValue = trimmed
			remaining = 0
		}
	}

	for _, n := range toRemove {
		para.RemoveChild(para, n)
	}

	if toReplace != nil {
		newNode := ast.NewString([]byte(replaceValue))
		para.ReplaceChild(para, toReplace, newNode)
	}

	if para.FirstChild() == nil {
		para.Parent().RemoveChild(para.Parent(), para)
	}
}

// AdmonitionExtension converts GitHub-style alert blocks ([!NOTE], [!TIP], etc.)
// into styled HTML admonition components.
type AdmonitionExtension struct{}

// NewAdmonitionExtension creates a new AdmonitionExtension.
func NewAdmonitionExtension() *AdmonitionExtension {
	return &AdmonitionExtension{}
}

// Extend registers the admonition transformer and renderer with the Goldmark parser.
func (ae *AdmonitionExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(&admonitionTransformer{}, 99),
	))

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&admonitionNodeRenderer{}, 1),
	))
}

type admonitionNodeRenderer struct{}

func (r *admonitionNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(admonitionNodeKind, r.renderAdmonition)
}

func (r *admonitionNodeRenderer) renderAdmonition(
	w util.BufWriter,
	_ []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("</div>\n</div>\n")

		return ast.WalkContinue, nil
	}

	admonition, ok := node.(*admonitionNode)
	if !ok {
		return ast.WalkContinue, nil
	}

	kind := string(admonition.kind)
	title := alertTitles[admonition.kind]

	_, _ = fmt.Fprintf(w, "<div class=\"admonition admonition-%s\">\n", kind)
	_, _ = fmt.Fprintf(w, "<div class=\"admonition-title\">%s</div>\n", title)
	_, _ = w.WriteString("<div class=\"admonition-content\">\n")

	return ast.WalkContinue, nil
}
