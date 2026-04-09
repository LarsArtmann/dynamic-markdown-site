package content

import (
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

var _ Repository = (*FileSystemRepository)(nil)

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
	return rootFromTree(r.tree, &r.mu)
}

// LastModified returns when the content was last indexed.
func (r *FileSystemRepository) LastModified() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lastModified
}

// AllPaths returns all URL paths in the repository.
func (r *FileSystemRepository) AllPaths() []domain.URLPath {
	return allPaths(r.tree, &r.mu)
}

// GetRaw retrieves a non-markdown file directly from the filesystem.
func (r *FileSystemRepository) GetRaw(urlPath domain.URLPath) (*RawFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert URL path to filesystem path
	fsPath := filepath.Join(r.rootDir, filepath.FromSlash(urlPath.String()))

	// Security check: ensure the path is within rootDir
	absPath, err := filepath.Abs(fsPath)
	if err != nil {
		return nil, errors.Wrapf(ErrContentNotFound, "invalid path: %s", urlPath)
	}

	absRoot, err := filepath.Abs(r.rootDir)
	if err != nil {
		return nil, errors.Wrapf(ErrContentNotFound, "invalid root: %s", r.rootDir)
	}

	if !strings.HasPrefix(absPath, absRoot) {
		return nil, errors.Wrapf(ErrContentNotFound, "path outside root: %s", urlPath)
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrapf(ErrContentNotFound, "file not found: %s", urlPath)
		}

		return nil, errors.Wrapf(err, "stat failed: %s", urlPath)
	}

	if info.IsDir() {
		return nil, errors.Wrapf(ErrContentNotFound, "path is directory: %s", urlPath)
	}

	// Skip markdown files - they should be served via Get
	if IsMarkdownFile(info.Name()) {
		return nil, errors.Wrapf(ErrContentNotFound, "markdown files served via Get: %s", urlPath)
	}

	// Skip hidden files
	if strings.HasPrefix(info.Name(), ".") {
		return nil, errors.Wrapf(ErrContentNotFound, "hidden file: %s", urlPath)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read failed: %s", urlPath)
	}

	return &RawFile{
		Content:     content,
		ContentType: GetContentType(info.Name()),
		ModTime:     info.ModTime(),
		Size:        uint64(info.Size()),
	}, nil
}

// Refresh rebuilds the content tree from the filesystem and returns statistics.
func (r *FileSystemRepository) Refresh() domain.RefreshResult {
	return doRefresh(&r.mu, &r.lastModified, &r.tree, r.buildTree)
}

func (r *FileSystemRepository) buildTree(stats *refreshStats) (*domain.DirectoryNode, error) {
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
	dirNodes map[string]*domain.DirectoryNode, stats *refreshStats,
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

	if d.IsDir() && ShouldSkipDir(d.Name()) {
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
	dirNodes map[string]*domain.DirectoryNode, stats *refreshStats,
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
	urlPath domain.URLPath, parentNode *domain.DirectoryNode, stats *refreshStats,
) {
	if !IsMarkdownFile(d.Name()) {
		return
	}

	content, err := os.ReadFile(fsPath)
	if err != nil {
		return
	}

	if isDraft(content) {
		return
	}

	title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

	// Strip .md/.markdown extension from URL path for clean URLs
	// (e.g., "/docs/readme.md" → "/docs/readme")
	cleanPath := strings.TrimSuffix(urlPath.String(), filepath.Ext(d.Name()))
	cleanURLPath, err := domain.NewURLPath(cleanPath)
	if err != nil {
		return
	}

	fileNode, err := domain.NewFileNode(
		cleanURLPath,
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
