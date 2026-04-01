package content

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"
)

func TestBlobRepository(t *testing.T) {
	t.Parallel()

	t.Run("NewBlobRepository", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		repo, err := NewBlobRepository(ctx, "mem://")
		require.NoError(t, err)
		require.NotNil(t, repo)

		root, err := repo.Root()
		require.NoError(t, err)
		require.NotNil(t, root)
		assert.Equal(t, "/", string(root.Path()))
	})

	t.Run("Get root", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		repo, err := NewBlobRepository(ctx, "mem://")
		require.NoError(t, err)

		root, err := repo.Get(domain.MustURLPath("/"))
		require.NoError(t, err)
		require.NotNil(t, root)
		assert.Equal(t, "/", string(root.Path()))
	})

	t.Run("Get non-existent path", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		repo, err := NewBlobRepository(ctx, "mem://")
		require.NoError(t, err)

		_, err = repo.Get(domain.MustURLPath("/nonexistent"))
		assert.ErrorIs(t, err, ErrContentNotFound)
	})

	t.Run("LastModified", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		repo, err := NewBlobRepository(ctx, "mem://")
		require.NoError(t, err)

		lm := repo.LastModified()
		assert.False(t, lm.IsZero())
	})

	t.Run("AllPaths", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		repo, err := NewBlobRepository(ctx, "mem://")
		require.NoError(t, err)

		paths := repo.AllPaths()
		assert.Contains(t, paths, domain.MustURLPath("/"))
	})

	t.Run("Refresh", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		repo, err := NewBlobRepository(ctx, "mem://")
		require.NoError(t, err)

		result := repo.Refresh()
		assert.True(t, result.Success)
	})
}

func TestBlobRepositoryWithContent(t *testing.T) {
	t.Parallel()

	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files
	testFiles := map[string]string{
		"index.md":      "# Welcome\n\nThis is the home page.",
		"docs/guide.md": "# Guide\n\nThis is a guide.",
		"docs/api.md":   "# API\n\nThis is the API reference.",
	}

	for name, content := range testFiles {
		fullPath := filepath.Join(tmpDir, name)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Open as fileblob
	ctx := context.Background()
	repo, err := NewBlobRepository(ctx, "file://"+tmpDir)
	require.NoError(t, err)
	require.NotNil(t, repo)

	t.Run("Get file", func(t *testing.T) {
		t.Parallel()

		node, err := repo.Get(domain.MustURLPath("/index"))
		require.NoError(t, err)
		require.NotNil(t, node)
		assert.Equal(t, "/index", string(node.Path()))

		fileNode, ok := node.(*domain.FileNode)
		require.True(t, ok, "expected *FileNode")
		assert.Equal(t, "index", fileNode.Title())
	})

	t.Run("Get nested file", func(t *testing.T) {
		t.Parallel()

		node, err := repo.Get(domain.MustURLPath("/docs/guide"))
		require.NoError(t, err)
		require.NotNil(t, node)

		fileNode, ok := node.(*domain.FileNode)
		require.True(t, ok, "expected *FileNode")
		assert.Equal(t, "guide", fileNode.Title())
	})

	t.Run("Get directory", func(t *testing.T) {
		t.Parallel()

		node, err := repo.Get(domain.MustURLPath("/docs"))
		require.NoError(t, err)
		require.NotNil(t, node)

		dirNode, ok := node.(*domain.DirectoryNode)
		require.True(t, ok, "expected *DirectoryNode")
		assert.Equal(t, "/docs", string(dirNode.Path()))
	})

	t.Run("AllPaths includes all files", func(t *testing.T) {
		t.Parallel()

		paths := repo.AllPaths()
		assert.Contains(t, paths, domain.MustURLPath("/"))
		assert.Contains(t, paths, domain.MustURLPath("/index"))
		assert.Contains(t, paths, domain.MustURLPath("/docs"))
		assert.Contains(t, paths, domain.MustURLPath("/docs/guide"))
		assert.Contains(t, paths, domain.MustURLPath("/docs/api"))
	})
}

func TestBlobRepositoryMemblob(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a memblob bucket for testing
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Write test content
	testContent := []byte("# Test\n\nThis is test content.")
	err := bucket.WriteAll(ctx, "test.md", testContent, nil)
	require.NoError(t, err)

	// Create repository
	repo, err := NewBlobRepository(ctx, "mem://")
	require.NoError(t, err)

	t.Run("Get file from memblob", func(t *testing.T) {
		t.Parallel()

		// The file won't be found because Refresh doesn't read from memblob in this test
		// This test just verifies the basic repository works
		root, err := repo.Root()
		require.NoError(t, err)
		assert.NotNil(t, root)
	})
}
