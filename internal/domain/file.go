package domain

import (
	"html/template"
	"strings"
	"time"
)

// TOCItem represents a table of contents entry.
type TOCItem struct {
	Level    uint
	Title    string
	Anchor   string
	Children []TOCItem
}

// Frontmatter represents metadata extracted from markdown files.
type Frontmatter struct {
	Title       string
	Description string
	Author      string
	Date        *time.Time
	Tags        []string
	Draft       bool
}

// wordsPerMinute is the average reading speed for estimating reading time.
const wordsPerMinute = 200

// FileNode represents a markdown file that can be rendered.
//
// TODO: Refactor to immutable pattern. Currently has mutable setters (SetHTML,
// SetTOC, SetMetadata) that are called during rendering. This breaks immutability
// and could cause race conditions. Future refactoring should:
//  1. Remove setters from FileNode
//  2. Create RenderedFile struct combining FileNode + rendered content
//  3. Pass RenderedFile to templates instead of mutating FileNode
//
// This is a medium-priority architectural improvement tracked in the roadmap.
type FileNode struct {
	path     URLPath
	title    string
	content  []byte
	html     template.HTML
	toc      []TOCItem
	metadata Frontmatter
	modified time.Time
	size     uint64
}

// NewFileNode creates a new FileNode with validation.
func NewFileNode(
	path URLPath,
	title string,
	content []byte,
	modified time.Time,
	size uint64,
) (*FileNode, error) {
	if path == "" {
		return nil, ErrEmptyPath
	}

	if title == "" {
		title = path.Filename()
	}

	return &FileNode{
		path:     path,
		title:    title,
		content:  content,
		modified: modified,
		size:     size,
	}, nil
}

// Kind returns the node kind (always File for FileNode).
func (f *FileNode) Kind() NodeKind { return NodeKindFile }

// Path returns the URL path to this file.
func (f *FileNode) Path() URLPath { return f.path }

// Title returns the file title.
func (f *FileNode) Title() string { return f.title }

// Modified returns when this file was last modified.
func (f *FileNode) Modified() time.Time { return f.modified }

// Size returns the file size in bytes.
func (f *FileNode) Size() uint64 { return f.size }

// Content returns the raw file content.
func (f *FileNode) Content() []byte { return f.content }

// HTML returns the rendered HTML.
func (f *FileNode) HTML() template.HTML {
	return f.html
}

// SetHTML sets the rendered HTML.
//
// Deprecated: Mutates the FileNode. Will be replaced by immutable RenderedFile pattern.
func (f *FileNode) SetHTML(html template.HTML) {
	f.html = html
}

// TOC returns the table of contents.
func (f *FileNode) TOC() []TOCItem {
	return f.toc
}

// SetTOC sets the table of contents.
//
// Deprecated: Mutates the FileNode. Will be replaced by immutable RenderedFile pattern.
func (f *FileNode) SetTOC(toc []TOCItem) {
	f.toc = toc
}

// Metadata returns the extracted frontmatter.
func (f *FileNode) Metadata() Frontmatter {
	return f.metadata
}

// SetMetadata sets the frontmatter.
//
// Deprecated: Mutates the FileNode. Will be replaced by immutable RenderedFile pattern.
func (f *FileNode) SetMetadata(meta Frontmatter) {
	f.metadata = meta
	if meta.Title != "" {
		f.title = meta.Title
	}
}

// ReadingTime returns estimated reading time in minutes.
func (f *FileNode) ReadingTime() uint {
	wordCount := uint(len(strings.Fields(string(f.content))))

	minutes := wordCount / wordsPerMinute
	if minutes == 0 {
		minutes = 1
	}

	return minutes
}
