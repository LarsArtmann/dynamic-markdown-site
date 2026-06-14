package main

import (
	"net"
	"net/http"
	"os"
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

	// Save and restore os.Args.
	origArgs := make([]string, len(os.Args))
	copy(origArgs, os.Args)
	t.Cleanup(func() {
		os.Args = origArgs
	})

	// Call the healthcheck with --addr pointing at our test server.
	os.Args = []string{"dynamic-markdown-site", "healthcheck", "--addr", addr, "--timeout", "2"}

	if err := runHealthcheck(); err != nil {
		t.Errorf("runHealthcheck returned error: %v", err)
	}
}
