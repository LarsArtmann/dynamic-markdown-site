package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
	"github.com/larsartmann/go-health"
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

// repositoryHealthChecker is the slice of the repository contract the probe
// invokes; production repositories and test doubles both satisfy it.
type repositoryHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// serveProbeHandler starts a real HTTP server whose handler is the production
// WithHealthProbes wiring over a go-health probe whose critical "repository"
// check delegates to the given repository. The refresh interval is frozen at
// an hour so each probe serves its synchronous first evaluation only.
func serveProbeHandler(t *testing.T, repo repositoryHealthChecker) string {
	t.Helper()

	probe := health.NewChecks(
		map[string]health.CheckFunc{
			"repository": func(ctx context.Context) error { return repo.HealthCheck(ctx) },
		},
		health.WithRefreshInterval(time.Hour),
		health.WithVersion(version.Version),
		health.WithCriticalServices("repository"),
	)
	if err := probe.Start(context.Background()); err != nil {
		t.Fatalf("probe.Start() error: %v", err)
	}
	t.Cleanup(probe.Shutdown)

	handler := server.WithHealthProbes(http.NotFoundHandler(), probe)

	listener, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test scaffolding
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	httpSrv := &http.Server{
		Handler:           handler,
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

	return listener.Addr().String()
}

// TestHealthcheckSubcommandTracksProbeReadiness pins the end-to-end seam the
// orchestrators rely on (ADR-0005 + ADR-0006): the healthcheck subcommand
// probes GET /health — the readiness alias served by WithHealthProbes — so its
// exit code mirrors the probe's verdict, not merely liveness. A passing
// critical repository check yields exit 0; a failing one yields an error
// wrapping errHealthcheckFailed, which main turns into exit 1.
func TestHealthcheckSubcommandTracksProbeReadiness(t *testing.T) {
	t.Parallel()

	t.Run("healthy repository exits nil", func(t *testing.T) {
		t.Parallel()

		addr := serveProbeHandler(t, content.NewInMemoryRepository())

		if err := runHealthcheckWithArgs([]string{"--addr", addr, "--timeout", "2"}); err != nil {
			t.Errorf("runHealthcheckWithArgs returned error: %v", err)
		}
	})

	t.Run("failing critical repository errors", func(t *testing.T) {
		t.Parallel()

		failing := failingHealthRepo{}
		addr := serveProbeHandler(t, failing)

		err := runHealthcheckWithArgs([]string{"--addr", addr, "--timeout", "2"})
		if !errors.Is(err, errHealthcheckFailed) {
			t.Errorf("runHealthcheckWithArgs error = %v, want errHealthcheckFailed", err)
		}
	})
}

// errStorageUnreachable backs failingHealthRepo; a static sentinel keeps the
// test free of dynamic errors (err113) and lets the drain path stay greppable.
var errStorageUnreachable = errors.New("storage unreachable")

// failingHealthRepo fails HealthCheck to simulate dead storage behind a
// server that still boots and serves traffic.
type failingHealthRepo struct{}

func (failingHealthRepo) HealthCheck(_ context.Context) error {
	return errStorageUnreachable
}
