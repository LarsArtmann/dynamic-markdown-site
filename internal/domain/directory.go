package domain

import "time"

// DirectoryNode represents a directory containing content nodes.
type DirectoryNode struct {
	path     URLPath
	title    string
	children []ContentNode
	modified time.Time
}

// NewDirectoryNode creates a new DirectoryNode with validation.
func NewDirectoryNode(path URLPath, title string, modified time.Time) (*DirectoryNode, error) {
	if path == "" {
		return nil, ErrEmptyPath
	}

	if title == "" {
		title = path.Filename()
	}

	return &DirectoryNode{
		path:     path,
		title:    title,
		children: make([]ContentNode, 0),
		modified: modified,
	}, nil
}

// Kind returns the node kind for DirectoryNode.
func (d *DirectoryNode) Kind() NodeKind { return NodeKindDirectory }

// Path returns the URL path for DirectoryNode.
func (d *DirectoryNode) Path() URLPath { return d.path }

// Title returns the display title for DirectoryNode.
func (d *DirectoryNode) Title() string { return d.title }

// Modified returns the last modification time for DirectoryNode.
func (d *DirectoryNode) Modified() time.Time { return d.modified }

// Children returns the child nodes.
func (d *DirectoryNode) Children() []ContentNode {
	return d.children
}

// AddChild adds a child node while maintaining sort order.
func (d *DirectoryNode) AddChild(child ContentNode) {
	insertIdx := 0

	for i, existing := range d.children {
		if shouldComeAfter(existing, child) {
			insertIdx = i

			break
		}

		insertIdx = i + 1
	}

	d.children = append(d.children, nil)
	copy(d.children[insertIdx+1:], d.children[insertIdx:])
	d.children[insertIdx] = child

	if child.Modified().After(d.modified) {
		d.modified = child.Modified()
	}
}

// SetChildren replaces all children with the given slice.
func (d *DirectoryNode) SetChildren(children []ContentNode) {
	d.children = children
}

// shouldComeAfter determines if existing should come after newNode in sort order.
func shouldComeAfter(existing, newNode ContentNode) bool {
	if existing.Kind() != newNode.Kind() {
		return existing.Kind() == NodeKindFile && newNode.Kind() == NodeKindDirectory
	}

	return existing.Title() > newNode.Title()
}
