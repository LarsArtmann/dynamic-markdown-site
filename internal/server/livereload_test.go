//go:build dev

package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestLiveReload(t *testing.T) *LiveReload {
	t.Helper()

	return NewLiveReload(slog.New(slog.DiscardHandler))
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
