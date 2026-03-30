package testutil

import (
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// ContentFixture provides test fixtures for content testing.
type ContentFixture struct{}

// NewContentFixture creates a new content fixture.
func NewContentFixture() *ContentFixture {
	return &ContentFixture{}
}

// SimpleHTML creates simple HTML content.
func (f *ContentFixture) SimpleHTML(html string) domain.RenderedContent {
	return domain.RenderedContent{
		HTML:     domain.HTML(html),
		TOC:      []domain.TOCItem{{Level: 1, Title: "Test", Anchor: "test"}},
		Metadata: domain.Frontmatter{Title: "Test Title"},
	}
}

// HTMLWithTOC creates HTML content with custom TOC.
func (f *ContentFixture) HTMLWithTOC(html string, toc []domain.TOCItem) domain.RenderedContent {
	return domain.RenderedContent{
		HTML:     domain.HTML(html),
		TOC:      toc,
		Metadata: domain.Frontmatter{Title: "Test Title"},
	}
}

// DirectoryNode creates a directory node for testing.
func (f *ContentFixture) DirectoryNode(path, title string) *domain.DirectoryNode {
	node, err := domain.NewDirectoryNode(
		domain.MustURLPath(path),
		title,
		time.Now(),
	)
	if err != nil {
		panic(err)
	}

	return node
}

// AssertContentEqual compares two RenderedContent values.
func AssertContentEqual(t *testing.T, got, want domain.RenderedContent) {
	t.Helper()

	if string(got.HTML) != string(want.HTML) {
		t.Errorf("Expected HTML %q, got %q", want.HTML, got.HTML)
	}

	if len(got.TOC) != len(want.TOC) {
		t.Errorf("Expected %d TOC items, got %d", len(want.TOC), len(got.TOC))
	}

	if got.Metadata.Title != want.Metadata.Title {
		t.Errorf("Expected title %q, got %q", want.Metadata.Title, got.Metadata.Title)
	}
}
