package main

import (
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestRunHealthcheckReturnsNilForHealthyServer(t *testing.T) {
	t.Parallel()

	// Spin up a tiny server that always returns 200.
	listener, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test scaffolding
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = httpSrv.Serve(listener)
	})
	t.Cleanup(func() {
		_ = httpSrv.Close()
		wg.Wait()
	})

	addr := listener.Addr().String()

	// Pass args directly: mutating os.Args races parallel test goroutines.
	if err := runHealthcheckWithArgs([]string{"--addr", addr, "--timeout", "2"}); err != nil {
		t.Errorf("runHealthcheckWithArgs returned error: %v", err)
	}
}

// TestRunHealthcheckFailsOnUnhealthyServer pins the orchestrator contract
// (ADR-0005): a non-200 /health verdict — the readiness alias returning 503
// for a failed critical dependency — makes runHealthcheck return an error,
// which main turns into exit 1.
func TestRunHealthcheckFailsOnUnhealthyServer(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test scaffolding
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = httpSrv.Serve(listener)
	})
	t.Cleanup(func() {
		_ = httpSrv.Close()
		wg.Wait()
	})

	addr := listener.Addr().String()

	if err := runHealthcheckWithArgs([]string{"--addr", addr, "--timeout", "2"}); err == nil {
		t.Error("runHealthcheckWithArgs returned nil, want error for a 503 /health verdict")
	}
}
