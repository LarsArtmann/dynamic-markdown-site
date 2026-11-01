//go:build dev

package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

func newTestLiveReload(t *testing.T) *LiveReload {
	t.Helper()

	return NewLiveReload(slog.New(slog.DiscardHandler))
}

// TestLiveReloadEndpointRunsInDevBuilds verifies a dev-tagged binary with
// -dev serves the SSE endpoint (the prod stub build must 404 instead, which
// TestLiveReloadEndpointDisabledOutsideDevMode asserts in both variants).
func TestLiveReloadEndpointRunsInDevBuilds(t *testing.T) {
	t.Parallel()

	srv := newTestServerWithDevMode(t, content.NewInMemoryRepository(), true)
	handler := newTestHandler(srv)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/live-reload", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait briefly for the SSE handler to start, then cancel to unblock it.
	// The SSE handler blocks on ctx.Done() or the hub's closed channel.
	cancel()
	<-done

	if rec.Code == http.StatusNotFound {
		t.Error("live-reload endpoint should be registered in dev builds, got 404")
	}
}

// TestLiveReloadCloseReleasesHandler proves the Close exit path: the SSE
// handler blocks forever on httptest.ResponseRecorder (its request context is
// never cancelled, and the recorder implements Flusher so the early return is
// not taken), so only LiveReload.Close can make the handler return.
func TestLiveReloadCloseReleasesHandler(t *testing.T) {
	t.Parallel()

	lr := newTestLiveReload(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/live-reload",
		nil,
	)

	done := make(chan struct{})
	go func() {
		lr.handleSSE(rec, req)
		close(done)
	}()

	lr.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handleSSE did not return after Close")
	}
}

func TestLiveReloadCloseIsIdempotentAndSafeAfterClose(t *testing.T) {
	t.Parallel()

	lr := newTestLiveReload(t)

	lr.Close()
	lr.Close()
	lr.Notify("/changed")
}
