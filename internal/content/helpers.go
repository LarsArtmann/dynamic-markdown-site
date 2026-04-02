package content

import (
	"path/filepath"
	"strings"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/samber/lo"
)

// skipDirs contains directory names to skip during content discovery.
//
//nolint:gochecknoglobals
var skipDirs = []string{"node_modules", ".git", "vendor", "dist", "build", "tmp", "temp"}

func shouldSkipDir(name string) bool {
	return lo.ContainsBy(skipDirs, func(skip string) bool {
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

func isMarkdownFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))

	return ext == ".md" || ext == ".markdown"
}

// contentTypes maps file extensions to MIME types.
//
//nolint:gochecknoglobals
var contentTypes = map[string]string{
	".svg":  "image/svg+xml",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".pdf":  "application/pdf",
}

func getContentType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}
