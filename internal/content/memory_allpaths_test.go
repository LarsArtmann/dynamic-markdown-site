package content

import (
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func TestInMemoryRepository_AllPaths(t *testing.T) {
	t.Parallel()

	now := time.Now()
	repo := NewInMemoryRepository()

	if paths := repo.AllPaths(); len(paths) != 1 {
		// Only the root path should be present.
		t.Errorf("empty repo: AllPaths returned %d paths, want 1", len(paths))
	}

	// Add three files.
	for _, p := range []string{"/a", "/b", "/c"} {
		f, err := domain.NewFileNode(
			domain.MustURLPath(p),
			"Title",
			[]byte("body"),
			now,
			4,
		)
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		repo.Add(f)
	}

	paths := repo.AllPaths()
	if len(paths) != 4 {
		t.Errorf("expected 4 paths (root + 3 files), got %d", len(paths))
	}

	// Verify all expected paths are present.
	got := make(map[string]bool, len(paths))
	for _, p := range paths {
		got[p.String()] = true
	}
	for _, expected := range []string{"/", "/a", "/b", "/c"} {
		if !got[expected] {
			t.Errorf("expected path %q in AllPaths result, missing", expected)
		}
	}
}

func TestInMemoryRepository_AllPathsWithNestedDirectories(t *testing.T) {
	t.Parallel()

	now := time.Now()
	repo := NewInMemoryRepository()

	docsDir, err := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Docs", now)
	if err != nil {
		t.Fatalf("create dir: %v", err)
	}
	repo.Add(docsDir)

	// File under /docs/intro
	intro, err := domain.NewFileNode(
		domain.MustURLPath("/docs/intro"),
		"Intro",
		[]byte("body"),
		now,
		4,
	)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	repo.Add(intro)

	paths := repo.AllPaths()
	hasRoot := false
	hasIntro := false
	for _, p := range paths {
		switch p.String() {
		case "/":
			hasRoot = true
		case "/docs/intro":
			hasIntro = true
		}
	}
	if !hasRoot {
		t.Error("AllPaths missing root path")
	}
	if !hasIntro {
		t.Error("AllPaths missing nested file path")
	}
}
