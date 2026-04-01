package content

import (
	"context"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"gocloud.dev/blob"
)

// BlobRepository implements Repository using go-cloud blob storage.
type BlobRepository struct {
	mu           sync.RWMutex
	bucket       *blob.Bucket
	prefix       string
	tree         *domain.ContentTree
	lastModified time.Time
}

// NewBlobRepository creates a new repository using a go-cloud blob.Bucket URL.
// The URL can be:
//   - file:///path/to/dir (local filesystem)
//   - s3://bucket-name/path/prefix
//   - gs://bucket-name/path/prefix
//   - azblob://container/path/prefix
//   - mem:// (in-memory)
func NewBlobRepository(ctx context.Context, bucketURL string) (*BlobRepository, error) {
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open blob bucket: %s", bucketURL)
	}

	repo := &BlobRepository{
		bucket: bucket,
		prefix: "",
	}

	if result := repo.Refresh(); !result.Success { //nolint:contextcheck
		if result.Error != "" {
			return nil, errors.Wrapf(errRefreshFailed, "refresh %s: %s", bucketURL, result.Error)
		}

		return nil, errors.Wrapf(errRefreshFailed, "refresh %s: unknown error", bucketURL)
	}

	return repo, nil
}

// Get retrieves a content node by URL path.
func (r *BlobRepository) Get(p domain.URLPath) (domain.ContentNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tree == nil {
		return nil, errors.Wrapf(ErrContentNotFound, "path: %s", p)
	}

	if p.IsRoot() {
		return r.tree.Root(), nil
	}

	node, found := r.tree.Find(p)
	if !found {
		return nil, errors.Wrapf(ErrContentNotFound, "path: %s", p)
	}

	return node, nil
}

// Root returns the root directory.
func (r *BlobRepository) Root() (*domain.DirectoryNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tree == nil {
		return nil, errors.Wrapf(ErrContentNotFound, "tree not initialized")
	}

	return r.tree.Root(), nil
}

// LastModified returns when the content was last indexed.
func (r *BlobRepository) LastModified() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lastModified
}

// AllPaths returns all URL paths in the repository.
func (r *BlobRepository) AllPaths() []domain.URLPath {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tree == nil {
		return []domain.URLPath{domain.MustURLPath("/")}
	}

	return r.tree.AllPaths()
}

// Refresh rebuilds the content tree from blob storage and returns statistics.
func (r *BlobRepository) Refresh() domain.RefreshResult {
	start := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	stats := &blobTreeStats{}

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

type blobTreeStats struct {
	files  int
	dirs   int
	errors []string
}

func (s *blobTreeStats) recordError(blobPath, operation string, err error) {
	s.errors = append(s.errors, operation+" at "+blobPath+": "+err.Error())
}

func (r *BlobRepository) buildTree(stats *blobTreeStats) (*domain.DirectoryNode, error) {
	rootPath := domain.MustURLPath("/")

	root, err := domain.NewDirectoryNode(rootPath, "Home", time.Now())
	if err != nil {
		return nil, errors.Wrapf(err, "create root node")
	}

	dirNodes := make(map[string]*domain.DirectoryNode)
	dirNodes["/"] = root

	// List all blobs under the prefix
	iter := r.bucket.List(&blob.ListOptions{Prefix: r.prefix})
	for {
		obj, err := iter.Next(context.Background())
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			stats.recordError(r.prefix, "list error", err)
			continue
		}

		blobPath := obj.Key
		if r.prefix != "" && strings.HasPrefix(blobPath, r.prefix) {
			blobPath = strings.TrimPrefix(blobPath, r.prefix)
		}
		blobPath = strings.TrimPrefix(blobPath, "/")

		if blobPath == "" || strings.HasSuffix(blobPath, "/") {
			continue
		}

		if strings.HasPrefix(path.Base(blobPath), ".") {
			continue
		}

		parentPath := path.Dir(blobPath)
		if parentPath == "." {
			parentPath = "/"
		} else {
			parentPath = "/" + parentPath
		}

		parentNode, exists := dirNodes[parentPath]
		if !exists {
			parentNode = r.findOrCreateParentDirs(
				parentPath, root, dirNodes, stats,
			)
		}

		if isMarkdownFile(blobPath) {
			urlPath, err := domain.NewURLPath(
				"/" + strings.TrimSuffix(blobPath, path.Ext(blobPath)),
			)
			if err != nil {
				stats.recordError(blobPath, "invalid path", err)
				continue
			}

			content, err := r.readBlobContent(obj.Key)
			if err != nil {
				stats.recordError(obj.Key, "read error", err)
				continue
			}

			if isDraft(content) {
				continue
			}

			title := strings.TrimSuffix(path.Base(obj.Key), path.Ext(obj.Key))

			fileNode, err := domain.NewFileNode(
				urlPath,
				title,
				content,
				obj.ModTime,
				uint64(obj.Size),
			)
			if err != nil {
				stats.recordError(obj.Key, "create node error", err)
				continue
			}

			parentNode.AddChild(fileNode)
			stats.files++
		}
	}

	filterEmptyDirectories(root)

	return root, nil
}

func (r *BlobRepository) findOrCreateParentDirs(
	blobPath string,
	root *domain.DirectoryNode,
	dirNodes map[string]*domain.DirectoryNode,
	stats *blobTreeStats,
) *domain.DirectoryNode {
	parts := strings.Split(strings.Trim(blobPath, "/"), "/")

	currentPath := ""
	currentNode := root

	for _, part := range parts {
		if currentPath == "" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		if node, exists := dirNodes[currentPath]; exists {
			currentNode = node
			continue
		}

		urlPath := domain.MustURLPath(currentPath)
		dirNode, err := domain.NewDirectoryNode(urlPath, part, time.Now())
		if err != nil {
			stats.recordError(blobPath, "create dir node error", err)
			return currentNode
		}

		dirNodes[currentPath] = dirNode
		currentNode.AddChild(dirNode)
		currentNode = dirNode
		stats.dirs++
	}

	return currentNode
}

func (r *BlobRepository) readBlobContent(blobPath string) ([]byte, error) {
	reader, err := r.bucket.NewReader(context.Background(), blobPath, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read blob: %s", blobPath)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)

	return data, errors.Wrapf(err, "read blob content: %s", blobPath)
}

// GetRaw retrieves a non-markdown file directly from blob storage.
func (r *BlobRepository) GetRaw(urlPath domain.URLPath) (*RawFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert URL path to blob path
	blobPath := strings.TrimPrefix(urlPath.String(), "/")

	// Skip markdown files - they should be served via Get
	if isMarkdownFile(blobPath) {
		return nil, errors.Wrapf(ErrContentNotFound, "markdown files served via Get: %s", urlPath)
	}

	// Skip hidden files
	if strings.HasPrefix(path.Base(blobPath), ".") {
		return nil, errors.Wrapf(ErrContentNotFound, "hidden file: %s", urlPath)
	}

	// Check if blob exists
	exists, err := r.bucket.Exists(context.Background(), blobPath)
	if err != nil {
		return nil, errors.Wrapf(err, "check existence failed: %s", urlPath)
	}

	if !exists {
		return nil, errors.Wrapf(ErrContentNotFound, "blob not found: %s", urlPath)
	}

	// Get blob attributes
	attrs, err := r.bucket.Attributes(context.Background(), blobPath)
	if err != nil {
		return nil, errors.Wrapf(err, "get attributes failed: %s", urlPath)
	}

	// Read blob content
	content, err := r.readBlobContent(blobPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read failed: %s", urlPath)
	}

	return &RawFile{
		Content:     content,
		ContentType: getContentType(blobPath),
		ModTime:     attrs.ModTime,
		Size:        uint64(attrs.Size),
	}, nil
}

// Close closes the underlying blob bucket.
func (r *BlobRepository) Close() error {
	return errors.Wrap(r.bucket.Close(), "close blob bucket")
}
