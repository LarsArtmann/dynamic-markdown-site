package content

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// writeTestFile creates a test markdown file with the given content.
func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(
		filepath.Join(dir, name),
		[]byte(content),
		0o644,
	); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
}

// newFileSystemRepoWithFiles creates a repo with given files and asserts success.
func newFileSystemRepoWithFiles(t *testing.T, files map[string]string) *FileSystemRepository {
	t.Helper()
	tmpDir := t.TempDir()
	for name, content := range files {
		writeTestFile(t, tmpDir, name, content)
	}
	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}
	return repo
}

func TestNewFileSystemRepository(t *testing.T) {
	t.Run("valid directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		if repo == nil {
			t.Fatal("expected repository, got nil")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		repo, err := NewFileSystemRepository("/nonexistent/path")
		if err == nil {
			t.Fatal("expected error for non-existent directory")
		}

		if repo != nil {
			t.Fatal("expected nil repository for error case")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		writeTestFile(t, tmpDir, "test.txt", "test")

		repo, err := NewFileSystemRepository(tmpFile)
		if err == nil {
			t.Fatal("expected error when passing file instead of directory")
		}

		if repo != nil {
			t.Fatal("expected nil repository for error case")
		}
	})

	t.Run("trailing slash in root directory path", func(t *testing.T) {
		// Regression test for trailing slash mismatch bug
		// Root cause: filepath.Dir() returns path without trailing slash,
		// causing parent directory lookup to fail when root has trailing slash
		tmpDir := t.TempDir()

		// Create a markdown file in the root
		writeTestFile(t, tmpDir, "readme.md", "# README")

		// Add trailing slash to directory path
		rootWithSlash := tmpDir + string(filepath.Separator)

		repo, err := NewFileSystemRepository(rootWithSlash)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() with trailing slash error = %v", err)
		}

		root, err := repo.Root()
		if err != nil {
			t.Fatalf("repo.Root() error = %v", err)
		}

		// Verify the file was found (not filtered out due to path mismatch)
		children := root.Children()
		if len(children) == 0 {
			t.Fatal("expected children in root, got none - path mismatch bug regression")
		}

		if len(children) != 1 {
			t.Errorf("expected 1 child, got %d", len(children))
		}

		if children[0].Title() != "readme" {
			t.Errorf("expected child title 'readme', got %s", children[0].Title())
		}
	})
}

// assertNotFoundErr is a helper to assert an error is ErrContentNotFound.
func assertNotFoundErr(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error, got nil", msg)
	}

	if !errors.Is(err, ErrContentNotFound) {
		t.Errorf("%s: expected ErrContentNotFound, got %v", msg, err)
	}
}

// assertGetRawNotFound creates a repo and asserts that GetRaw returns not found.
func assertGetRawNotFound(t *testing.T, repo *FileSystemRepository, urlPath string) {
	t.Helper()
	_, err := repo.GetRaw(domain.MustURLPath(urlPath))
	assertNotFoundErr(t, err, "GetRaw")
}

func TestFileSystemRepository_Get(t *testing.T) {
	t.Run("get root path", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		rootPath := domain.MustURLPath("/")

		node, err := repo.Get(rootPath)
		if err != nil {
			t.Fatalf("Get(/) error = %v", err)
		}

		if node == nil {
			t.Fatal("expected root node, got nil")
		}

		if node.Kind() != domain.NodeKindDirectory {
			t.Errorf("expected directory node, got %v", node.Kind())
		}
	})

	t.Run("get non-existent path", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		path := domain.MustURLPath("/nonexistent")

		_, err = repo.Get(path)
		assertNotFoundErr(t, err, "Get")
	})

	t.Run("get file node", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeTestFile(t, tmpDir, "test.md", "# Test")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		path := domain.MustURLPath("/test")

		node, err := repo.Get(path)
		if err != nil {
			t.Fatalf("Get(/test) error = %v", err)
		}

		if node == nil {
			t.Fatal("expected file node, got nil")
		}

		if node.Kind() != domain.NodeKindFile {
			t.Errorf("expected file node, got %v", node.Kind())
		}

		if node.Title() != "test" {
			t.Errorf("expected title 'test', got %q", node.Title())
		}
	})

	t.Run("get nested file node", func(t *testing.T) {
		tmpDir := t.TempDir()

		subDir := filepath.Join(tmpDir, "docs")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		writeTestFile(t, subDir, "guide.md", "# Guide")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		path := domain.MustURLPath("/docs/guide")

		node, err := repo.Get(path)
		if err != nil {
			t.Fatalf("Get(/docs/guide) error = %v", err)
		}

		if node == nil {
			t.Fatal("expected file node, got nil")
		}
	})

	t.Run("get directory node", func(t *testing.T) {
		tmpDir := t.TempDir()

		subDir := filepath.Join(tmpDir, "docs")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		writeTestFile(t, subDir, "guide.md", "# Guide")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		path := domain.MustURLPath("/docs")

		node, err := repo.Get(path)
		if err != nil {
			t.Fatalf("Get(/docs) error = %v", err)
		}

		if node == nil {
			t.Fatal("expected directory node, got nil")
		}

		if node.Kind() != domain.NodeKindDirectory {
			t.Errorf("expected directory node, got %v", node.Kind())
		}
	})
}

func TestFileSystemRepository_Root(t *testing.T) {
	t.Run("returns root directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		root, err := repo.Root()
		if err != nil {
			t.Fatalf("Root() error = %v", err)
		}

		if root == nil {
			t.Fatal("expected root directory, got nil")
		}

		if root.Title() != "Home" {
			t.Errorf("expected title 'Home', got %q", root.Title())
		}
	})
}

func TestFileSystemRepository_LastModified(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	before := time.Now()
	modified := repo.LastModified()
	after := time.Now()

	if modified.Before(before.Add(-time.Second)) {
		t.Errorf("LastModified() = %v, expected time close to %v", modified, before)
	}

	if modified.After(after.Add(time.Second)) {
		t.Errorf("LastModified() = %v, expected time close to %v", modified, after)
	}
}

func TestFileSystemRepository_Refresh(t *testing.T) {
	t.Run("refresh with new files", func(t *testing.T) {
		tmpDir := t.TempDir()

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		initialModified := repo.LastModified()

		time.Sleep(10 * time.Millisecond)

		writeTestFile(t, tmpDir, "new.md", "# New")

		result := repo.Refresh()
		if !result.Success {
			t.Errorf("Refresh() failed: %s", result.Error)
		}

		if result.TotalFiles != 1 {
			t.Errorf("Refresh() TotalFiles = %d, want 1", result.TotalFiles)
		}

		newModified := repo.LastModified()
		if !newModified.After(initialModified) {
			t.Errorf("LastModified() = %v, expected time after %v", newModified, initialModified)
		}

		path := domain.MustURLPath("/new")

		node, err := repo.Get(path)
		if err != nil {
			t.Fatalf("Get(/new) error = %v", err)
		}

		if node == nil {
			t.Fatal("expected file node after refresh")
		}
	})

	t.Run("refresh statistics", func(t *testing.T) {
		tmpDir := t.TempDir()

		subDir := filepath.Join(tmpDir, "docs")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		writeTestFile(t, tmpDir, "index.md", "# Index")
		writeTestFile(t, subDir, "guide.md", "# Guide")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		result := repo.Refresh()
		if !result.Success {
			t.Errorf("Refresh() failed: %s", result.Error)
		}

		if result.TotalFiles != 2 {
			t.Errorf("Refresh() TotalFiles = %d, want 2", result.TotalFiles)
		}

		if result.TotalDirs != 1 {
			t.Errorf("Refresh() TotalDirs = %d, want 1 (docs subdirectory)", result.TotalDirs)
		}

		if result.Duration == "" {
			t.Error("Refresh() Duration should not be empty")
		}
	})
}

func TestFileSystemRepository_SkipsHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "visible.md", "# Visible")
	writeTestFile(t, tmpDir, ".hidden.md", "# Hidden")

	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	result := repo.Refresh()
	if !result.Success {
		t.Errorf("Refresh() failed: %s", result.Error)
	}

	if result.TotalFiles != 1 {
		t.Errorf(
			"Refresh() TotalFiles = %d, want 1 (hidden files should be skipped)",
			result.TotalFiles,
		)
	}

	_, err = repo.Get(domain.MustURLPath("/.hidden.md"))
	if err == nil {
		t.Error("expected error for hidden file")
	}
}

func TestFileSystemRepository_SkipsHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	visibleDir := filepath.Join(tmpDir, "docs")
	if err := os.Mkdir(visibleDir, 0o755); err != nil {
		t.Fatalf("failed to create visible directory: %v", err)
	}

	writeTestFile(t, visibleDir, "guide.md", "# Guide")

	hiddenDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(hiddenDir, 0o755); err != nil {
		t.Fatalf("failed to create hidden directory: %v", err)
	}

	writeTestFile(t, hiddenDir, "config.md", "# Config")

	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	result := repo.Refresh()
	if !result.Success {
		t.Errorf("Refresh() failed: %s", result.Error)
	}

	if result.TotalFiles != 1 {
		t.Errorf(
			"Refresh() TotalFiles = %d, want 1 (hidden directories should be skipped)",
			result.TotalFiles,
		)
	}
}

func TestFileSystemRepository_SkipsBlacklistedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	visibleDir := filepath.Join(tmpDir, "docs")
	if err := os.Mkdir(visibleDir, 0o755); err != nil {
		t.Fatalf("failed to create visible directory: %v", err)
	}

	writeTestFile(t, visibleDir, "guide.md", "# Guide")

	for _, skipDir := range []string{"node_modules", "vendor", "dist", "build", "tmp", "temp"} {
		dir := filepath.Join(tmpDir, skipDir)

		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("failed to create %s directory: %v", skipDir, err)
		}

		writeTestFile(t, dir, "file.md", "# File")
	}

	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	result := repo.Refresh()
	if !result.Success {
		t.Errorf("Refresh() failed: %s", result.Error)
	}

	if result.TotalFiles != 1 {
		t.Errorf(
			"Refresh() TotalFiles = %d, want 1 (blacklisted directories should be skipped)",
			result.TotalFiles,
		)
	}
}

func TestFileSystemRepository_OnlyProcessesMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, tmpDir, "doc.md", "# Doc")
	writeTestFile(t, tmpDir, "readme.markdown", "# Readme")
	writeTestFile(t, tmpDir, "script.js", "console.log('test')")
	writeTestFile(t, tmpDir, "style.css", "body {}")

	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	result := repo.Refresh()
	if !result.Success {
		t.Errorf("Refresh() failed: %s", result.Error)
	}

	if result.TotalFiles != 2 {
		t.Errorf("Refresh() TotalFiles = %d, want 2 (only .md and .markdown)", result.TotalFiles)
	}
}

func TestFileSystemRepository_FiltersEmptyDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatalf("failed to create empty directory: %v", err)
	}

	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.Mkdir(docsDir, 0o755); err != nil {
		t.Fatalf("failed to create docs directory: %v", err)
	}

	writeTestFile(t, docsDir, "guide.md", "# Guide")

	repo, err := NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root() error = %v", err)
	}

	children := root.Children()
	for _, child := range children {
		if child.Title() == "empty" {
			t.Error("empty directory should be filtered out")
		}
	}
}

func TestFileSystemRepository_GetRaw(t *testing.T) {
	t.Run("serves SVG file from nested directory", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create nested directory structure like docs/assets/diagrams/
		diagramsDir := filepath.Join(tmpDir, "docs", "assets", "diagrams")
		if err := os.MkdirAll(diagramsDir, 0o755); err != nil {
			t.Fatalf("failed to create nested directory: %v", err)
		}

		svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`)
		if err := os.WriteFile(
			filepath.Join(diagramsDir, "workflow.svg"),
			svgContent,
			0o644,
		); err != nil {
			t.Fatalf("failed to write SVG file: %v", err)
		}

		// Need at least one markdown file so the tree doesn't filter the parent dirs
		writeTestFile(t, tmpDir, "index.md", "# Index")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		urlPath := domain.MustURLPath("/docs/assets/diagrams/workflow.svg")
		raw, err := repo.GetRaw(urlPath)
		if err != nil {
			t.Fatalf("GetRaw() error = %v", err)
		}

		if raw.ContentType != "image/svg+xml" {
			t.Errorf("ContentType = %q, want %q", raw.ContentType, "image/svg+xml")
		}

		if string(raw.Content) != string(svgContent) {
			t.Errorf("Content = %q, want %q", string(raw.Content), string(svgContent))
		}

		if raw.Size == 0 {
			t.Error("Size should be > 0")
		}
	})

	t.Run("serves PNG image from root", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		pngHeader := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
		if err := os.WriteFile(
			filepath.Join(tmpDir, "logo.png"),
			pngHeader,
			0o644,
		); err != nil {
			t.Fatalf("failed to write PNG file: %v", err)
		}

		writeTestFile(t, tmpDir, "index.md", "# Index")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		urlPath := domain.MustURLPath("/logo.png")
		raw, err := repo.GetRaw(urlPath)
		if err != nil {
			t.Fatalf("GetRaw() error = %v", err)
		}

		if raw.ContentType != "image/png" {
			t.Errorf("ContentType = %q, want %q", raw.ContentType, "image/png")
		}
	})

	t.Run("returns not found for non-existent and markdown paths", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name  string
			files map[string]string
			path  string
		}{
			{
				name:  "non-existent file",
				files: map[string]string{"index.md": "# Index"},
				path:  "/nonexistent.png",
			},
			{
				name:  "markdown file",
				files: map[string]string{"doc.md": "# Doc"},
				path:  "/doc.md",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := newFileSystemRepoWithFiles(t, tt.files)
				assertGetRawNotFound(t, repo, tt.path)
			})
		}
	})

	t.Run("returns not found for hidden files", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".secret"),
			[]byte("secret"),
			0o644,
		); err != nil {
			t.Fatalf("failed to write hidden file: %v", err)
		}

		writeTestFile(t, tmpDir, "index.md", "# Index")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		urlPath := domain.MustURLPath("/.secret")
		_, err = repo.GetRaw(urlPath)
		assertNotFoundErr(t, err, "GetRaw")
	})

	t.Run("returns not found for directories", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		subDir := filepath.Join(tmpDir, "assets")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		writeTestFile(t, tmpDir, "index.md", "# Index")

		repo, err := NewFileSystemRepository(tmpDir)
		if err != nil {
			t.Fatalf("NewFileSystemRepository() error = %v", err)
		}

		urlPath := domain.MustURLPath("/assets")
		_, err = repo.GetRaw(urlPath)
		assertNotFoundErr(t, err, "GetRaw")
	})
}
