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

func getContentType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))

	switch ext {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
