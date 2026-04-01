package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDraft(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "draft true",
			content: "---\ntitle: Test\ndraft: true\n---\n# Content",
			want:    true,
		},
		{
			name:    "draft false",
			content: "---\ntitle: Test\ndraft: false\n---\n# Content",
			want:    false,
		},
		{
			name:    "no frontmatter",
			content: "# Just content\n\nNo frontmatter here.",
			want:    false,
		},
		{
			name:    "frontmatter without draft field",
			content: "---\ntitle: Test\nauthor: Someone\n---\n# Content",
			want:    false,
		},
		{
			name:    "unclosed frontmatter",
			content: "---\ntitle: Test\ndraft: true\n\n# Missing closing",
			want:    false,
		},
		{
			name:    "empty content",
			content: "",
			want:    false,
		},
		{
			name:    "draft true with extra whitespace",
			content: "---\ntitle: Test\n  draft: true  \n---\n# Content",
			want:    false,
		},
		{
			name:    "draft true in body not frontmatter",
			content: "---\ntitle: Test\n---\ndraft: true\n# Content",
			want:    false,
		},
		{
			name:    "draft yes",
			content: "---\ntitle: Test\ndraft: yes\n---\n# Content",
			want:    true,
		},
		{
			name:    "draft True",
			content: "---\ntitle: Test\ndraft: True\n---\n# Content",
			want:    true,
		},
		{
			name:    "draft TRUE",
			content: "---\ntitle: Test\ndraft: TRUE\n---\n# Content",
			want:    true,
		},
		{
			name:    "draft on",
			content: "---\ntitle: Test\ndraft: on\n---\n# Content",
			want:    true,
		},
		{
			name:    "empty frontmatter",
			content: "---\n---\n# Content",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isDraft([]byte(tt.content))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileSystemRepositorySkipsDrafts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	writeFile(t, tmpDir, "published.md", "---\ntitle: Published\n---\n# Published")
	writeFile(t, tmpDir, "draft.md", "---\ntitle: Draft\ndraft: true\n---\n# Draft")

	repo, err := NewFileSystemRepository(tmpDir)
	require.NoError(t, err)

	_, err = repo.Get(domain.MustURLPath("/published"))
	require.NoError(t, err, "published file should be found")

	_, err = repo.Get(domain.MustURLPath("/draft"))
	require.ErrorIs(t, err, ErrContentNotFound, "draft file should be excluded")
}

func TestBlobRepositorySkipsDrafts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	testFiles := map[string]string{
		"published.md": "---\ntitle: Published\n---\n# Published",
		"draft.md":     "---\ntitle: Draft\ndraft: true\n---\n# Draft",
	}

	for name, content := range testFiles {
		fullPath := filepath.Join(tmpDir, name)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	repo, err := NewBlobRepository(t.Context(), "file://"+tmpDir)
	require.NoError(t, err)

	_, err = repo.Get(domain.MustURLPath("/published"))
	require.NoError(t, err, "published file should be found")

	_, err = repo.Get(domain.MustURLPath("/draft"))
	require.ErrorIs(t, err, ErrContentNotFound, "draft file should be excluded")
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	fullPath := filepath.Join(dir, name)
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(fullPath, []byte(content), 0o644)
	require.NoError(t, err)
}
