package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

type countingRepository struct {
	content.Repository

	refreshes atomic.Int64
}

func (c *countingRepository) Refresh() domain.RefreshResult {
	c.refreshes.Add(1)

	return c.Repository.Refresh()
}

func newWatchRoot(t *testing.T) string {
	t.Helper()

	root, err := os.MkdirTemp(".", "watcher-test-")
	if err != nil {
		t.Fatalf("create watch root: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(root) })

	return root
}

func startWatcher(
	t *testing.T,
	root string,
	repo content.Repository,
) (cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		watchForChanges(ctx, root, repo, nil, slog.New(slog.DiscardHandler))
	}()

	time.Sleep(500 * time.Millisecond)

	return cancel, finished
}

func stopWatcher(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("watcher did not exit within 5s after context cancel")
	}
}

func waitForRefreshes(t *testing.T, repo *countingRepository, minimum int64, timeout time.Duration) int64 {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if got := repo.refreshes.Load(); got >= minimum {
			return got
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("repository refresh count stayed below %d within %s", minimum, timeout)

	return 0
}

func containsPath(paths []domain.URLPath, want string) bool {
	for _, p := range paths {
		if p.String() == want {
			return true
		}
	}

	return false
}

func TestWatchForChanges_RefreshesOnMarkdownWrite(t *testing.T) {
	t.Parallel()

	root := newWatchRoot(t)

	fsRepo, err := content.NewFileSystemRepository(root)
	if err != nil {
		t.Fatalf("NewFileSystemRepository: %v", err)
	}

	repo := &countingRepository{Repository: fsRepo}

	cancel, done := startWatcher(t, root, repo)

	if err := os.WriteFile(filepath.Join(root, "hello.md"), []byte("# Hello\n"), 0o600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if containsPath(repo.AllPaths(), "/hello.md") {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if !containsPath(repo.AllPaths(), "/hello") {
		t.Error("watcher did not refresh the repository with the new markdown file")
	}

	stopWatcher(t, cancel, done)
}

func TestWatchForChanges_IgnoresSkippedDirs(t *testing.T) {
	t.Parallel()

	root := newWatchRoot(t)

	fsRepo, err := content.NewFileSystemRepository(root)
	if err != nil {
		t.Fatalf("NewFileSystemRepository: %v", err)
	}

	repo := &countingRepository{Repository: fsRepo}

	cancel, done := startWatcher(t, root, repo)

	if err := os.MkdirAll(filepath.Join(root, "vendor"), 0o700); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "vendor", "secret.md"), []byte("# Secret\n"), 0o600); err != nil {
		t.Fatalf("write ignored markdown: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "hello.md"), []byte("# Hello\n"), 0o600); err != nil {
		t.Fatalf("write root markdown: %v", err)
	}

	first := waitForRefreshes(t, repo, 1, 5*time.Second)
	time.Sleep(3 * time.Second)

	if got := repo.refreshes.Load(); got != first {
		t.Errorf("ignored-dir write triggered %d unexpected refresh(s) beyond the initial %d", got-first, first)
	}

	if containsPath(repo.AllPaths(), "/vendor/secret") {
		t.Error("content inside a skipped directory must not appear in the repository")
	}

	stopWatcher(t, cancel, done)
}

func TestWatchForChanges_ExitsOnContextCancel(t *testing.T) {
	t.Parallel()

	root := newWatchRoot(t)

	fsRepo, err := content.NewFileSystemRepository(root)
	if err != nil {
		t.Fatalf("NewFileSystemRepository: %v", err)
	}

	cancel, done := startWatcher(t, root, fsRepo)
	stopWatcher(t, cancel, done)
}
