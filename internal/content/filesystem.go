package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// FileSystemRepository implements Repository using the local filesystem.
type FileSystemRepository struct {
	mu           sync.RWMutex
	rootDir      string
	tree         *domain.ContentTree
	lastModified time.Time
}

var errRefreshFailed = errors.New("content refresh failed")

// NewFileSystemRepository creates a new repository for the given root directory.
func NewFileSystemRepository(rootDir string) (*FileSystemRepository, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidRoot, "stat %s: %w", rootDir, err)
	}

	if !info.IsDir() {
		return nil, errors.Wrapf(ErrInvalidRoot, "%s is not a directory", rootDir)
	}

	repo := &FileSystemRepository{
		rootDir: filepath.Clean(rootDir),
	}

	if result := repo.Refresh(); !result.Success {
		if result.Error != "" {
			return nil, errors.Wrapf(errRefreshFailed, "refresh %s: %s", rootDir, result.Error)
		}

		return nil, errors.Wrapf(errRefreshFailed, "refresh %s: unknown error", rootDir)
	}

	return repo, nil
}

// Get retrieves a content node by URL path.
func (r *FileSystemRepository) Get(path domain.URLPath) (domain.ContentNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tree == nil {
		return nil, errors.Wrapf(ErrContentNotFound, "path: %s", path)
	}

	if path.IsRoot() {
		return r.tree.Root(), nil
	}

	node, found := r.tree.Find(path)
	if !found {
		return nil, errors.Wrapf(ErrContentNotFound, "path: %s", path)
	}

	return node, nil
}

// Root returns the root directory.
func (r *FileSystemRepository) Root() (*domain.DirectoryNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tree == nil {
		return nil, errors.Wrapf(ErrContentNotFound, "tree not initialized")
	}

	return r.tree.Root(), nil
}

// LastModified returns when the content was last indexed.
func (r *FileSystemRepository) LastModified() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lastModified
}

// Refresh rebuilds the content tree from the filesystem and returns statistics.
func (r *FileSystemRepository) Refresh() domain.RefreshResult {
	start := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	stats := &treeStats{}

	rootNode, err := r.buildTree(stats)
	if err != nil {
		return domain.RefreshResult{
			Success:      false,
			LastModified: r.lastModified,
			Error:        err.Error(),
			Duration:     time.Since(start).String(),
		}
	}

	r.tree = domain.NewContentTree(rootNode)
	r.lastModified = time.Now()

	return domain.RefreshResult{
		Success:      true,
		LastModified: r.lastModified,
		TotalFiles:   stats.files,
		TotalDirs:    stats.dirs,
		Duration:     time.Since(start).String(),
		Errors:       stats.errors,
	}
}

type treeStats struct {
	files  int
	dirs   int
	errors []string
}

func (s *treeStats) recordError(fsPath, operation string, err error) {
	s.errors = append(s.errors, fmt.Sprintf("%s at %s: %v", operation, fsPath, err))
}

func (r *FileSystemRepository) buildTree(stats *treeStats) (*domain.DirectoryNode, error) {
	rootPath := domain.MustURLPath("/")

	rootInfo, err := os.Stat(r.rootDir)
	if err != nil {
		return nil, errors.Wrapf(err, "stat root %s: %w", r.rootDir, err)
	}

	root, err := domain.NewDirectoryNode(rootPath, "Home", rootInfo.ModTime())
	if err != nil {
		return nil, errors.Wrapf(err, "create root node for %s: %w", r.rootDir, err)
	}

	dirNodes := make(map[string]*domain.DirectoryNode)
	dirNodes[r.rootDir] = root

	err = filepath.WalkDir(r.rootDir, func(fsPath string, d fs.DirEntry, err error) error {
		return r.walkEntry(fsPath, d, err, dirNodes, stats)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "walk %s: %w", r.rootDir, err)
	}

	filterEmptyDirectories(root)

	return root, nil
}

func (r *FileSystemRepository) walkEntry(
	fsPath string, d fs.DirEntry, err error,
	dirNodes map[string]*domain.DirectoryNode, stats *treeStats,
) error {
	if err != nil {
		stats.recordError(fsPath, "walk error", err)

		return nil
	}

	if fsPath == r.rootDir {
		return nil
	}

	if strings.HasPrefix(d.Name(), ".") {
		if d.IsDir() {
			return fs.SkipDir
		}

		return nil
	}

	if d.IsDir() && shouldSkipDir(d.Name()) {
		return fs.SkipDir
	}

	info, err := d.Info()
	if err != nil {
		stats.recordError(fsPath, "failed to get info", err)

		return nil
	}

	relPath, err := filepath.Rel(r.rootDir, fsPath)
	if err != nil {
		stats.recordError(fsPath, "failed to get relative path", err)

		return nil
	}

	urlPath, err := domain.NewURLPath("/" + filepath.ToSlash(relPath))
	if err != nil {
		stats.recordError(fsPath, "invalid path", err)

		return nil
	}

	parentDir := filepath.Dir(fsPath)

	parentNode, exists := dirNodes[parentDir]
	if !exists {
		stats.recordError(fsPath, "parent not found", errors.New("parent directory not found"))

		return nil
	}

	if d.IsDir() {
		r.processDirectory(fsPath, d, info, urlPath, parentNode, dirNodes, stats)
	} else {
		r.processFile(fsPath, d, info, urlPath, parentNode, stats)
	}

	return nil
}

func (r *FileSystemRepository) processDirectory(
	fsPath string, d fs.DirEntry, info fs.FileInfo,
	urlPath domain.URLPath, parentNode *domain.DirectoryNode,
	dirNodes map[string]*domain.DirectoryNode, stats *treeStats,
) {
	dirNode, err := domain.NewDirectoryNode(urlPath, d.Name(), info.ModTime())
	if err != nil {
		return
	}

	dirNodes[fsPath] = dirNode
	parentNode.AddChild(dirNode)

	stats.dirs++
}

func (r *FileSystemRepository) processFile(
	fsPath string, d fs.DirEntry, info fs.FileInfo,
	urlPath domain.URLPath, parentNode *domain.DirectoryNode, stats *treeStats,
) {
	if !isMarkdownFile(d.Name()) {
		return
	}

	content, err := os.ReadFile(fsPath)
	if err != nil {
		return
	}

	title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

	fileNode, err := domain.NewFileNode(
		urlPath,
		title,
		content,
		info.ModTime(),
		uint64(info.Size()),
	)
	if err != nil {
		return
	}

	parentNode.AddChild(fileNode)

	stats.files++
}
