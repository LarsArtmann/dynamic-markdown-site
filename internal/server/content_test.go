package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

func TestStaticFileServing(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/static/favicon.svg", nil),
	)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRawFileServing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	rawBody := []byte("sample raw payload")

	if err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), rawBody, 0o600); err != nil {
		t.Fatalf("write raw file: %v", err)
	}

	repo, err := content.NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	rec := executeRequest(handler, "/data.txt")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Body.String(); got != string(rawBody) {
		t.Errorf("body = %q, want %q", got, string(rawBody))
	}

	if ct := rec.Header().Get("Content-Type"); ct != content.GetContentType("data.txt") {
		t.Errorf("Content-Type = %q, want %q", ct, content.GetContentType("data.txt"))
	}
}

func TestMDExtensionRedirect(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	rec := executeRequest(handler, "/guide.md")

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
}
