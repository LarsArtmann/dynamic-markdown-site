package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// WriteTestFile creates a test file with the given content.
func WriteTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(
		filepath.Join(dir, name),
		[]byte(content),
		0o644,
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
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}
}

// NewTestDir creates a new directory node for testing.
func NewTestDir(t *testing.T, path string) *domain.DirectoryNode {
	t.Helper()
	dir, err := domain.NewDirectoryNode(domain.MustURLPath(path), "", time.Time{})
	if err != nil {
		t.Fatalf("failed to create test dir %q: %v", path, err)
	}
	return dir
}

// NewTestFile creates a new file node for testing with default title.
func NewTestFile(t *testing.T, path string, content string) *domain.FileNode {
	t.Helper()
	file, err := domain.NewFileNode(
		domain.MustURLPath(path),
		"",
		[]byte(content),
		time.Time{},
		uint64(len(content)),
	)
	if err != nil {
		t.Fatalf("failed to create test file %q: %v", path, err)
	}
	return file
}

// NewTestFileWithTitle creates a new file node for testing with a specific title.
func NewTestFileWithTitle(t *testing.T, path string, title string, content string) *domain.FileNode {
	t.Helper()
	file, err := domain.NewFileNode(
		domain.MustURLPath(path),
		title,
		[]byte(content),
		time.Now(),
		uint64(len(content)),
	)
	if err != nil {
		t.Fatalf("failed to create test file %q: %v", path, err)
	}
	return file
}

// NewTestFileWithTime creates a new file node for testing with a specific modified time.
func NewTestFileWithTime(t *testing.T, path string, content string, modified time.Time) *domain.FileNode {
	t.Helper()
	file, err := domain.NewFileNode(
		domain.MustURLPath(path),
		"",
		[]byte(content),
		modified,
		uint64(len(content)),
	)
	if err != nil {
		t.Fatalf("failed to create test file %q: %v", path, err)
	}
	return file
}

// AssertChildCount asserts the number of children in a directory.
func AssertChildCount(t *testing.T, children []domain.ContentNode, want int, msg string) {
	t.Helper()
	if len(children) != want {
		t.Errorf("%s: got %d children, want %d", msg, len(children), want)
	}
}

// AssertChildTitle asserts a child's title.
func AssertChildTitle(t *testing.T, child domain.ContentNode, want string, msg string) {
	t.Helper()
	if child.Title() != want {
		t.Errorf("%s: got title %q, want %q", msg, child.Title(), want)
	}
}

// AssertChildKind asserts a child's kind.
func AssertChildKind(t *testing.T, child domain.ContentNode, want domain.NodeKind, msg string) {
	t.Helper()
	if child.Kind() != want {
		t.Errorf("%s: got kind %v, want %v", msg, child.Kind(), want)
	}
}
