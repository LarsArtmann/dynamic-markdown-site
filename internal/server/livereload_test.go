//go:build dev

package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// syncRecorder wraps httptest.ResponseRecorder with a mutex: the SSE handler
// streams on its own goroutine while the test polls the body, and the plain
// recorder is not safe for that concurrent access.
type syncRecorder struct {
	mu     sync.Mutex
	body   strings.Builder
	header http.Header
	code   int
}

func newSyncRecorder() *syncRecorder {
	return &syncRecorder{header: make(http.Header)}
}

func (r *syncRecorder) Header() http.Header { return r.header }

func (r *syncRecorder) WriteHeader(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.code = code
}

func (r *syncRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.body.Write(p) //nolint:errcheck // strings.Builder never errors
}

// Flush marks the recorder an http.Flusher for the SSE handler; frames are
// considered delivered once written.
func (r *syncRecorder) Flush() {}

func (r *syncRecorder) snapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.body.String()
}

// TestLiveReloadSendsHeartbeats pins the F54 heartbeat: an idle SSE stream
// periodically receives an SSE comment frame (`: heartbeat`) that keeps
// proxies and browsers from reaping the dev stream. The heartbeat interval is
// shortened in-process so the test stays well under a second.
func TestLiveReloadSendsHeartbeats(t *testing.T) {
	t.Parallel()

	lr := newTestLiveReload(t)
	lr.heartbeat = 20 * time.Millisecond

	rec := newSyncRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/live-reload", nil)

	done := make(chan struct{})
	go func() {
		lr.handleSSE(rec, req)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for !strings.Contains(rec.snapshot(), ": heartbeat") {
		select {
		case <-deadline:
			t.Fatalf("no heartbeat within 2s; body so far: %q", rec.snapshot())
		case <-time.After(5 * time.Millisecond):
		}
	}

	lr.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handleSSE did not return after Close")
	}
}

// TestLiveReloadRejectsClientsBeyondCap pins the F55 cap: concurrent SSE
// streams are bounded per dev server, and overflow attempts get a 503 with an
// actionable message instead of pinning unbounded goroutines and fds.
func TestLiveReloadRejectsClientsBeyondCap(t *testing.T) {
	t.Parallel()

	lr := newTestLiveReload(t)
	lr.clientCap = 2

	handlers := make([]struct {
		rec  *httptest.ResponseRecorder
		done chan struct{}
	}, 2)

	for i := range handlers {
		handlers[i].rec = httptest.NewRecorder()
		handlers[i].done = make(chan struct{})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/live-reload", nil)

		go func(rec *httptest.ResponseRecorder, done chan struct{}) {
			lr.handleSSE(rec, req)
			close(done)
		}(handlers[i].rec, handlers[i].done)
	}

	// The cap is checked under the hub lock at registration; poll until both
	// admitted handlers appear in the client set.
	deadline := time.After(2 * time.Second)
	for {
		lr.mu.RLock()
		registered := len(lr.clients)
		lr.mu.RUnlock()

		if registered == 2 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("cap-satisfying clients never registered; have %d", registered)
		case <-time.After(5 * time.Millisecond):
		}
	}

	overflow := httptest.NewRecorder()
	overflowReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/live-reload", nil)
	lr.handleSSE(overflow, overflowReq)

	if overflow.Code != http.StatusServiceUnavailable {
		t.Errorf("overflow client status = %d, want 503", overflow.Code)
	}

	if !strings.Contains(overflow.Body.String(), "limit reached (2)") {
		t.Errorf("overflow body = %q, want it to carry the cap", overflow.Body.String())
	}

	lr.Close()

	for i := range handlers {
		select {
		case <-handlers[i].done:
		case <-time.After(time.Second):
			t.Fatalf("admitted handler %d did not return after Close", i)
		}
	}
}
