package renderer

import "github.com/yuin/goldmark/ast"

// nodeReplacement tracks a node to replace in the AST during transformation.
type nodeReplacement struct {
	parent ast.Node
	old    ast.Node
	new    ast.Node
}

// applyReplacements replaces old nodes with new nodes in the AST.
// This must be called after the walk completes to avoid unsafe mutation during iteration.
func applyReplacements(replacements []nodeReplacement) {
	for _, r := range replacements {
		r.parent.ReplaceChild(r.parent, r.old, r.new)
	}
}

// walkAST walks the AST document and silently handles any walk errors.
// This eliminates repetitive error handling boilerplate.
func walkAST(node ast.Node, fn func(ast.Node, bool) (ast.WalkStatus, error)) {
	if err := ast.Walk(node, fn); err != nil {
		return
	}
}
