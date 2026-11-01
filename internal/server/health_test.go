package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
	"github.com/larsartmann/go-health"
)

// probeCheck mirrors one entry of the go-health readiness payload.
type probeCheck struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// probeHealthPayload mirrors the go-health readiness response served at
// GET /health (the readiness alias for /readyz).
type probeHealthPayload struct {
	Status  string                `json:"status"`
	Version string                `json:"version"`
	Uptime  string                `json:"uptime"`
	Checks  map[string]probeCheck `json:"checks"`
}

// newTestProbe builds a started go-health probe over explicit check funcs.
// The refresh interval is frozen at an hour so tests observe the
// deterministic first evaluation only. critical names the subset whose
// failure flips readiness to 503; the rest degrade to warn.
func newTestProbe(tb testing.TB, critical []string, checks map[string]health.CheckFunc) *health.Probe {
	tb.Helper()

	opts := []health.Option{
		health.WithRefreshInterval(time.Hour),
		health.WithVersion(version.Version),
	}
	if len(critical) > 0 {
		opts = append(opts, health.WithCriticalServices(critical...))
	}

	probe := health.NewChecks(checks, opts...)
	if err := probe.Start(context.Background()); err != nil {
		tb.Fatalf("probe.Start() error: %v", err)
	}

	tb.Cleanup(probe.Shutdown)

	return probe
}

// healthCheckingRepo is the slice of the repository contract the probe
// invokes; test doubles and production repositories both satisfy it.
type healthCheckingRepo interface {
	HealthCheck(ctx context.Context) error
}

func repositoryCheck(repo healthCheckingRepo) health.CheckFunc {
	return func(ctx context.Context) error {
		return repo.HealthCheck(ctx)
	}
}

func newHealthyTestProbe(tb testing.TB, repo healthCheckingRepo) *health.Probe {
	tb.Helper()

	return newTestProbe(tb, []string{"repository"}, map[string]health.CheckFunc{
		"repository": repositoryCheck(repo),
	})
}

func fetchHealth(t *testing.T, srv *Server, probe *health.Probe, wantCode int) probeHealthPayload {
	t.Helper()

	handler := WithHealthProbes(newTestHandler(srv), probe)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil),
	)

	if rec.Code != wantCode {
		t.Fatalf("status = %d, want %d", rec.Code, wantCode)
	}

	var payload probeHealthPayload
	decodeJSON(t, rec, &payload)

	return payload
}

func TestHealthAliasServesReadiness(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newHealthyTestProbe(t, repo)

	payload := fetchHealth(t, srv, probe, http.StatusOK)

	if payload.Status != "pass" {
		t.Errorf("status = %q, want pass", payload.Status)
	}

	repoCheck, ok := payload.Checks["repository"]
	if !ok {
		t.Fatal("expected repository check in payload")
	}

	if repoCheck.Status != "pass" {
		t.Errorf("repository check status = %q, want pass", repoCheck.Status)
	}
}

func TestHealthEndpointReportsUptime(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newHealthyTestProbe(t, repo)

	payload := fetchHealth(t, srv, probe, http.StatusOK)

	if payload.Uptime == "" {
		t.Fatal("expected uptime in response, got empty")
	}

	if _, err := time.ParseDuration(payload.Uptime); err != nil {
		t.Errorf("uptime %q is not a duration: %v", payload.Uptime, err)
	}
}

func TestHealthEndpointReportsVersion(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newHealthyTestProbe(t, repo)

	payload := fetchHealth(t, srv, probe, http.StatusOK)

	if payload.Version != version.Version {
		t.Errorf("version = %q, want %q", payload.Version, version.Version)
	}
}

func TestHealthEndpointFailsOnCriticalRepository(t *testing.T) {
	t.Parallel()

	repo := &FailingRepository{}
	srv := newUnhealthyRepositoryServer(t)
	probe := newTestProbe(t, []string{"repository"}, map[string]health.CheckFunc{
		"repository": repositoryCheck(repo),
	})

	payload := fetchHealth(t, srv, probe, http.StatusServiceUnavailable)

	if payload.Status != "fail" {
		t.Errorf("status = %q, want fail", payload.Status)
	}

	repoCheck, ok := payload.Checks["repository"]
	if !ok {
		t.Fatal("expected repository check in payload")
	}

	if repoCheck.Status != "fail" {
		t.Errorf("repository check status = %q, want fail", repoCheck.Status)
	}

	if repoCheck.Error == "" {
		t.Error("expected repository check error detail, got empty")
	}
}

func TestRendererDegradationWarnsWithoutFailingReadiness(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newTestProbe(t, []string{"repository"}, map[string]health.CheckFunc{
		"repository": repositoryCheck(repo),
		"renderer": func(_ context.Context) error {
			return context.DeadlineExceeded // any non-critical failure
		},
	})

	payload := fetchHealth(t, srv, probe, http.StatusOK)

	if payload.Status != "warn" {
		t.Errorf("status = %q, want warn (degraded but serving)", payload.Status)
	}

	if payload.Checks["renderer"].Status == "pass" {
		t.Error("renderer check should not report pass while degraded")
	}
}

func TestLivenessNeverChecksDependencies(t *testing.T) {
	t.Parallel()

	repo := &FailingRepository{}
	srv := newUnhealthyRepositoryServer(t)
	probe := newTestProbe(t, []string{"repository"}, map[string]health.CheckFunc{
		"repository": repositoryCheck(repo),
	})

	handler := WithHealthProbes(newTestHandler(srv), probe)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("liveness status = %d, want 200 even with a dead critical dependency", rec.Code)
	}

	var payload probeHealthPayload
	decodeJSON(t, rec, &payload)

	if payload.Status != "pass" {
		t.Errorf("liveness status = %q, want pass", payload.Status)
	}
}

func newUnhealthyRepositoryServer(t *testing.T) *Server {
	t.Helper()

	htmlCache := cache.NewHTMLCache(10)
	t.Cleanup(htmlCache.Close)

	return newTestServerWithCache(t, &FailingRepository{}, htmlCache)
}

func TestWithHealthProbesNilProbeReturnsNext(t *testing.T) {
	t.Parallel()

	next := http.NewServeMux()

	if got := WithHealthProbes(next, nil); got != http.Handler(next) {
		t.Error("WithHealthProbes with a nil probe must return the wrapped handler unchanged")
	}
}
