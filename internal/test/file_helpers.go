// Package test provides shared test helpers for the dynamic-markdown-site project.
package test

import (
	"os"
	"path/filepath"
	"testing"
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
