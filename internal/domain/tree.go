package domain

// ContentTree is the root of the content hierarchy.
type ContentTree struct {
	root *DirectoryNode
}

// NewContentTree creates a new content tree with the given root.
func NewContentTree(root *DirectoryNode) *ContentTree {
	return &ContentTree{root: root}
}

// Root returns the root directory node.
func (t *ContentTree) Root() *DirectoryNode {
	return t.root
}

// Find traverses the tree to find a node by path.
func (t *ContentTree) Find(path URLPath) (ContentNode, bool) {
	return t.findInNode(t.root, path)
}

// AllPaths returns all paths in the content tree.
func (t *ContentTree) AllPaths() []URLPath {
	return t.collectPaths(t.root)
}

func (t *ContentTree) collectPaths(node ContentNode) []URLPath {
	var paths []URLPath
	paths = append(paths, node.Path())

	if dir, ok := node.(*DirectoryNode); ok {
		for _, child := range dir.Children() {
			paths = append(paths, t.collectPaths(child)...)
		}
	}

	return paths
}

func (t *ContentTree) findInNode(node ContentNode, target URLPath) (ContentNode, bool) {
	if node.Path() == target {
		return node, true
	}

	if dir, ok := node.(*DirectoryNode); ok {
		for _, child := range dir.Children() {
			if found, ok := t.findInNode(child, target); ok {
				return found, true
			}
		}
	}

	return nil, false
}

// Breadcrumb represents a single breadcrumb item.
type Breadcrumb struct {
	Path     URLPath
	Title    string
	IsActive bool
}

// BuildBreadcrumbs creates breadcrumbs from root to the given path.
func BuildBreadcrumbs(path URLPath) []Breadcrumb {
	if path.IsRoot() {
		return []Breadcrumb{
			{Path: MustURLPath("/"), Title: "Home", IsActive: true},
		}
	}

	crumbs := make([]Breadcrumb, 0)
	segments := path.Segments()

	crumbs = append(crumbs, Breadcrumb{
		Path:     MustURLPath("/"),
		Title:    "Home",
		IsActive: false,
	})

	currentPath := ""

	for i, segment := range segments {
		if segment == "" {
			continue
		}

		currentPath = currentPath + "/" + segment
		crumbPath := MustURLPath(currentPath)

		crumbs = append(crumbs, Breadcrumb{
			Path:     crumbPath,
			Title:    segment,
			IsActive: i == len(segments)-1,
		})
	}

	return crumbs
}
