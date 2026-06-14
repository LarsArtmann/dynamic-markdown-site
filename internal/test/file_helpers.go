// Package test provides shared test helpers for the dynamic-markdown-site project.
package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

const (
	testFilePerm  = 0o644
	testDirectory = 0o755
)

// WriteTestFile creates a test file with the given content.
func WriteTestFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(
		filepath.Join(dir, name),
		[]byte(content),
		testFilePerm,
	)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
}

// WriteTestFiles writes multiple test files to a directory.
func WriteTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		fullPath := filepath.Join(dir, name)

		err := os.MkdirAll(filepath.Dir(fullPath), testDirectory)
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		err = os.WriteFile(fullPath, []byte(content), testFilePerm)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}
}

// FileNode describes a file to add to a test repository.
type FileNode struct {
	Path    string
	Title   string
	Content string
	ModTime time.Time
}

// NewFileNode creates a domain.FileNode for testing. Prefer this helper over
// local re-implementations so all tests share the same construction logic.
func (f FileNode) NewFileNode(t *testing.T) *domain.FileNode {
	t.Helper()

	node, err := domain.NewFileNode(
		domain.MustURLPath(f.Path),
		f.Title,
		[]byte(f.Content),
		f.ModTime,
		uint64(len(f.Content)),
	)
	if err != nil {
		t.Fatalf("failed to create file node %q: %v", f.Path, err)
	}

	return node
}
