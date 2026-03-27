package domain

import (
	"path/filepath"
	"slices"
	"strings"
)

// URLPath is a value object representing a safe URL path.
type URLPath string

// NewURLPath creates a new URLPath with validation.
func NewURLPath(path string) (URLPath, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	path = filepath.Clean(path)

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	segments := strings.Split(path, "/")
	if slices.Contains(segments, "..") {
		return "", ErrInvalidPath
	}

	return URLPath(path), nil
}

// MustURLPath panics if the path is invalid. Use only for hardcoded paths.
func MustURLPath(path string) URLPath {
	p, err := NewURLPath(path)
	if err != nil {
		panic(err)
	}

	return p
}

// String returns the path as a string.
func (p URLPath) String() string {
	return string(p)
}

// Join creates a new URLPath by appending a segment.
func (p URLPath) Join(segment string) (URLPath, error) {
	if segment == "" {
		return p, nil
	}

	cleaned := filepath.Clean(segment)
	if strings.Contains(cleaned, "..") {
		return "", ErrInvalidPath
	}

	joined := filepath.Join(string(p), cleaned)

	return NewURLPath(joined)
}

// Parent returns the parent directory path.
func (p URLPath) Parent() URLPath {
	dir := filepath.Dir(string(p))
	if dir == "." {
		dir = "/"
	}

	return URLPath(dir)
}

// Segments returns the path split into segments.
func (p URLPath) Segments() []string {
	segments := strings.Split(string(p), "/")
	if len(segments) > 0 && segments[0] == "" {
		segments = segments[1:]
	}

	return segments
}

// Filename returns the last segment of the path.
func (p URLPath) Filename() string {
	return filepath.Base(string(p))
}

// IsRoot returns true if this is the root path.
func (p URLPath) IsRoot() bool {
	return string(p) == "/" || string(p) == ""
}
