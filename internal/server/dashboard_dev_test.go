//go:build dev

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

// TestHealthDashboardMountsUnderDebugPrefixInDevBuilds pins the M8 spike:
// dev builds serve the go-health-dashboard HTML page at /debug/health-dashboard
// without disturbing the orchestrator probe routes. The /health assertion is
// the collision guard — the dashboard's default route set claims /health as
// its HTML page, which would shadow the readiness alias the healthcheck
// subcommand probes (ADR-0005).
func TestHealthDashboardMountsUnderDebugPrefixInDevBuilds(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newHealthyTestProbe(t, repo)

	handler := WithHealthProbes(newTestHandler(srv), probe)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/health-dashboard", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "data-star") && !strings.Contains(body, "datastar") {
		t.Error("dashboard HTML should reference the Datastar runtime it streams updates with")
	}

	if strings.Contains(body, "\"status\": \"pass\"") {
		t.Error("dashboard route served JSON probe payload; want the HTML page")
	}
}

// TestHealthAliasStaysReadinessJSONBesideDashboard pins that mounting the
// dashboard does not shadow GET /health: the alias must keep serving the
// readiness JSON document the healthcheck subcommand parses for its exit code.
func TestHealthAliasStaysReadinessJSONBesideDashboard(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newHealthyTestProbe(t, repo)

	payload := fetchHealth(t, srv, probe, http.StatusOK)

	if payload.Status != "pass" {
		t.Errorf("status = %q, want pass", payload.Status)
	}
}

// TestHealthDashboardCSPGrantsUnsafeEval pins the dashboard CSP lane: the
// Datastar SDK compiles its data-* expressions with the Function
// constructor, and without 'unsafe-eval' it throws Error: GenerateExpression
// at init and the SSE connection never opens (page renders, never updates).
// The grant is scoped to the dashboard prefix; the site policy elsewhere
// must stay unchanged.
func TestHealthDashboardCSPGrantsUnsafeEval(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	probe := newHealthyTestProbe(t, repo)
	handler := WithHealthProbes(newTestHandler(srv), probe)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/health-dashboard", nil),
	)
	dashCSP := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(dashCSP, "'unsafe-eval'") {
		t.Errorf("dashboard CSP = %q, want 'unsafe-eval' (the Datastar SDK cannot initialize without it)", dashCSP)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/nonexistent-page", nil),
	)
	siteCSP := rec.Header().Get("Content-Security-Policy")
	if strings.Contains(siteCSP, "'unsafe-eval'") {
		t.Errorf("site CSP = %q, want no 'unsafe-eval' outside the dashboard prefix", siteCSP)
	}
}

// TestMountHealthDashboardNilProbeIsNoOp pins the nil guard: containers
// without a probe must not panic on dashboard mounting.
func TestMountHealthDashboardNilProbeIsNoOp(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mountHealthDashboard(mux, nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/health-dashboard", nil),
	)

	if rec.Code != http.StatusNotFound {
		t.Errorf("dashboard status with nil probe = %d, want 404 (nothing mounted)", rec.Code)
	}
}
