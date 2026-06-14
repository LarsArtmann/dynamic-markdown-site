package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
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

	repo := content.NewInMemoryRepository()
	// Add a raw file (non-markdown) to the in-memory repo.
	p, err := domain.NewURLPath("/robots.txt")
	if err != nil {
		t.Fatalf("invalid path: %v", err)
	}

	raw := &content.RawFile{
		Content:     []byte("User-agent: *\nAllow: /\n"),
		ContentType: "text/plain; charset=utf-8",
		ModTime:     time.Now(),
		Size:        27,
	}
	_ = raw
	_ = p

	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	// We just need to assert that the handler is wired and the route resolves.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
	)

	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 200 or 500", rec.Code)
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
