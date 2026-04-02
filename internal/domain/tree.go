package domain

// ContentTree is the root of the content hierarchy.
type ContentTree struct {
	root  *DirectoryNode
	paths map[URLPath]ContentNode
}

// NewContentTree creates a new content tree with the given root.
func NewContentTree(root *DirectoryNode) *ContentTree {
	t := &ContentTree{
		root:  root,
		paths: make(map[URLPath]ContentNode),
	}

	t.indexNode(root)

	return t
}

func (t *ContentTree) indexNode(node ContentNode) {
	t.paths[node.Path()] = node

	if dir, ok := node.(*DirectoryNode); ok {
		for _, child := range dir.Children() {
			t.indexNode(child)
		}
	}
}

// Root returns the root directory node.
func (t *ContentTree) Root() *DirectoryNode {
	return t.root
}

// Find looks up a node by path using the pre-built index.
func (t *ContentTree) Find(path URLPath) (ContentNode, bool) {
	node, ok := t.paths[path]

	return node, ok
}

// AllPaths returns all paths in the content tree.
func (t *ContentTree) AllPaths() []URLPath {
	paths := make([]URLPath, 0, len(t.paths))

	for p := range t.paths {
		paths = append(paths, p)
	}

	return paths
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
