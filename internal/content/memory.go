package content

import (
	"sync"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// InMemoryRepository is a Repository implementation useful for testing.
type InMemoryRepository struct {
	mu       sync.RWMutex
	nodes    map[domain.URLPath]domain.ContentNode
	root     *domain.DirectoryNode
	modified time.Time
}

// NewInMemoryRepository creates a new in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	root, _ := domain.NewDirectoryNode(domain.MustURLPath("/"), "Home", time.Now())

	return &InMemoryRepository{
		nodes:    make(map[domain.URLPath]domain.ContentNode),
		root:     root,
		modified: time.Now(),
	}
}

// Add adds a node to the repository.
func (r *InMemoryRepository) Add(node domain.ContentNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nodes[node.Path()] = node
	r.modified = time.Now()
}

// Get retrieves a node by path.
func (r *InMemoryRepository) Get(path domain.URLPath) (domain.ContentNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if path.IsRoot() {
		return r.root, nil
	}

	node, exists := r.nodes[path]
	if !exists {
		return nil, ErrContentNotFound
	}

	return node, nil
}

// Root returns the root directory.
func (r *InMemoryRepository) Root() (*domain.DirectoryNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.root, nil
}

// Refresh is a no-op for in-memory repository.
func (r *InMemoryRepository) Refresh() domain.RefreshResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.modified = time.Now()

	return domain.RefreshResult{
		Success:      true,
		LastModified: r.modified,
		TotalFiles:   len(r.nodes),
		TotalDirs:    1,
		Duration:     "0ns",
	}
}

// LastModified returns when the content was last modified.
func (r *InMemoryRepository) LastModified() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.modified
}
