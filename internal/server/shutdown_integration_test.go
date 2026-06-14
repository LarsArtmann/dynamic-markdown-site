package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
)

// TestGracefulShutdownStopsInFlightRequests verifies that http.Server.Shutdown
// allows in-flight requests to complete and that subsequent requests fail.
func TestGracefulShutdownStopsInFlightRequests(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	repo := content.NewInMemoryRepository()
	htmlCache := cache.NewHTMLCache(100)
	t.Cleanup(htmlCache.Close)

	srv := NewServer(
		repo,
		content.NewSearcher(repo),
		logger,
		htmlCache,
		renderer.NewGoldmarkRenderer(),
		false,
		"Site",
	)

	// Find a free port for the test.
	listener, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test scaffolding
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	httpSrv := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server in the background.
	var wg sync.WaitGroup
	wg.Go(func() {
		_ = httpSrv.Serve(listener)
	})

	// Wait for the server to be ready.
	addr := listener.Addr().String()
	if !waitForServer(addr, time.Second) {
		t.Fatal("server did not become ready")
	}

	// Issue an in-flight request, then immediately call Shutdown.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+addr+"/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() { //nolint:bodyclose // body is closed when consumed from respCh
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			errCh <- e

			return
		}
		respCh <- resp
	}()

	// Give the request time to start.
	time.Sleep(10 * time.Millisecond)

	// Shutdown the server with a 2s grace period.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// In-flight request should still have completed.
	select {
	case resp := <-respCh:
		if resp.StatusCode != http.StatusOK {
			t.Errorf("in-flight response status = %d, want 200", resp.StatusCode)
		}
		_ = resp.Body.Close()
	case err := <-errCh:
		t.Errorf("in-flight request failed: %v", err)
	case <-time.After(time.Second):
		t.Error("in-flight request did not complete")
	}

	// Subsequent requests should fail (server is closed).
	client := &http.Client{Timeout: time.Second}
	newReq, newErr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+addr+"/health", nil)
	if newErr != nil {
		t.Fatalf("new request: %v", newErr)
	}

	resp, err := client.Do(newReq)
	if err == nil {
		_ = resp.Body.Close()
		t.Error("expected request to fail after shutdown")
	}

	wg.Wait()
	srv.Shutdown()
}

// waitForServer polls the address until it accepts a connection or times out.
func waitForServer(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	dialer := &net.Dialer{Timeout: 100 * time.Millisecond}
	for time.Now().Before(deadline) {
		conn, err := dialer.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()

			return true
		}
		time.Sleep(20 * time.Millisecond)
	}

	return false
}
