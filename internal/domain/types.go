// Package domain contains the core domain types for the dynamic markdown site.
package domain

import (
	"errors"
	"time"
)

// NodeKind represents the type of content node.
type NodeKind int

const (
	// NodeKindDirectory indicates a directory node in the content tree.
	NodeKindDirectory NodeKind = iota
	// NodeKindFile indicates a file node in the content tree.
	NodeKindFile
)

func (k NodeKind) String() string {
	switch k {
	case NodeKindDirectory:
		return "directory"
	case NodeKindFile:
		return "file"
	default:
		return "unknown"
	}
}

// ContentNode is the core interface representing either a directory or file.
type ContentNode interface {
	Kind() NodeKind
	Path() URLPath
	Title() string
	Modified() time.Time
}

var (
	_ ContentNode = (*DirectoryNode)(nil)
	_ ContentNode = (*FileNode)(nil)
	_ ContentNode = (*RenderedFile)(nil)
)

// RefreshResult contains statistics about a repository refresh operation.
type RefreshResult struct {
	Success      bool      `json:"success"`
	LastModified time.Time `json:"lastModified"`
	TotalFiles   int       `json:"totalFiles"`
	TotalDirs    int       `json:"totalDirs"`
	Duration     string    `json:"duration"`
	Error        string    `json:"error,omitempty"`
	Errors       []string  `json:"errors,omitempty"` // Non-fatal errors during refresh
}

// Sentinel errors for path validation.
var (
	// ErrInvalidPath is returned when a path contains directory traversal or invalid characters.
	ErrInvalidPath = errors.New("invalid path: contains directory traversal or invalid characters")
	// ErrEmptyPath is returned when a path is empty.
	ErrEmptyPath = errors.New("path cannot be empty")
)

// HTML represents pre-escaped HTML content.
// This type is used to mark strings that contain pre-escaped HTML,
// distinguishing them from plain text strings that should be escaped.
type HTML string

// RenderedContent holds the result of rendering markdown content.
type RenderedContent struct {
	HTML       HTML
	TOC        []TOCItem
	Metadata   Frontmatter
	HasMermaid bool
}

// Renderer converts raw markdown content into rendered HTML with metadata.
type Renderer interface {
	Render(source []byte) (RenderedContent, error)
}

// SuggestedPath represents a path suggestion with similarity score for 404 pages.
type SuggestedPath struct {
	Path  URLPath
	Title string
	Score float64
}
