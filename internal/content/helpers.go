package content

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/samber/lo"
)

// SkipDirs contains directory names to skip during content discovery.
//
//nolint:gochecknoglobals
var SkipDirs = []string{"node_modules", ".git", "vendor", "dist", "build", "tmp", "temp"}

// ShouldSkipDir returns true if the directory should be skipped during traversal.
func ShouldSkipDir(name string) bool {
	return lo.ContainsBy(SkipDirs, func(skip string) bool {
		return strings.EqualFold(name, skip)
	})
}

func filterEmptyDirectories(dir *domain.DirectoryNode) bool {
	children := dir.Children()

	var filteredChildren []domain.ContentNode

	hasMarkdown := false

	for _, child := range children {
		if subdir, ok := child.(*domain.DirectoryNode); ok {
			if hasChildMarkdown := filterEmptyDirectories(subdir); hasChildMarkdown {
				filteredChildren = append(filteredChildren, subdir)
				hasMarkdown = true
			}
		} else {
			filteredChildren = append(filteredChildren, child)
			hasMarkdown = true
		}
	}

	dir.SetChildren(filteredChildren)

	return hasMarkdown
}

// IsMarkdownFile returns true if the filename has a markdown extension.
func IsMarkdownFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))

	return ext == ".md" || ext == ".markdown"
}

// ContentTypes maps file extensions to MIME types.
//
//nolint:gochecknoglobals
var ContentTypes = map[string]string{
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".css":   "text/css",
	".js":    "application/javascript",
	".json":  "application/json",
	".pdf":   "application/pdf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

// GetContentType returns the MIME type for a file based on its extension.
func GetContentType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))

	if ct, ok := ContentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}

// allPaths returns all URL paths from a content tree with proper locking.
func allPaths(tree *domain.ContentTree, mu *sync.RWMutex) []domain.URLPath {
	mu.RLock()
	defer mu.RUnlock()

	if tree == nil {
		return []domain.URLPath{domain.MustURLPath("/")}
	}

	return tree.AllPaths()
}

// rootFromTree returns the root directory from a content tree with proper locking.
func rootFromTree(tree *domain.ContentTree, mu *sync.RWMutex) (*domain.DirectoryNode, error) {
	mu.RLock()
	defer mu.RUnlock()

	if tree == nil {
		return nil, errors.Wrapf(ErrContentNotFound, "tree not initialized")
	}

	return tree.Root(), nil
}

// refreshStats contains statistics collected during a repository refresh.
type refreshStats struct {
	files  int
	dirs   int
	errors []string
}

// newRefreshStats creates a new refresh stats instance.
func newRefreshStats() *refreshStats {
	return &refreshStats{}
}

// recordError records an error with the given path and operation.
func (s *refreshStats) recordError(path, operation string, err error) {
	s.errors = append(s.errors, operation+" at "+path+": "+err.Error())
}

// buildRefreshResult creates a success RefreshResult from the given stats.
func buildRefreshResult(stats *refreshStats, lastModified time.Time, start time.Time) domain.RefreshResult {
	return domain.RefreshResult{
		Success:      true,
		LastModified: lastModified,
		TotalFiles:   stats.files,
		TotalDirs:    stats.dirs,
		Duration:     time.Since(start).String(),
		Errors:       stats.errors,
	}
}

// buildFailedRefreshResult creates a failed RefreshResult with an error.
func buildFailedRefreshResult(lastModified time.Time, start time.Time, errMsg string) domain.RefreshResult {
	return domain.RefreshResult{
		Success:      false,
		LastModified: lastModified,
		Error:       errMsg,
		Duration:    time.Since(start).String(),
	}
}
