package content

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

var (
	// ErrContentNotFound indicates requested content does not exist.
	ErrContentNotFound = errors.New("content not found")
	// ErrInvalidRoot indicates the root directory is invalid.
	ErrInvalidRoot = errors.New("invalid root directory")
)

// Repository defines the interface for content storage and retrieval.
type Repository interface {
	// Get retrieves a content node by its URL path
	Get(path domain.URLPath) (domain.ContentNode, error)
	// Root returns the root directory of the content tree
	Root() (*domain.DirectoryNode, error)
	// Refresh rebuilds the content tree from filesystem and returns statistics
	Refresh() domain.RefreshResult
	// LastModified returns when the content was last indexed
	LastModified() time.Time
	// AllPaths returns all URL paths in the repository
	AllPaths() []domain.URLPath
}
