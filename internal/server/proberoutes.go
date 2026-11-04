package server

import (
	"net/http"

	"github.com/larsartmann/go-health"
)

// WithHealthProbes mounts the go-health probe routes in front of the site
// handler: /healthz (liveness, never checks dependencies), /readyz and
// /startupz, plus GET /health as a readiness alias so the distroless
// `healthcheck` subcommand and Docker HEALTHCHECK (ADR-0005) keep probing one
// stable path whose verdict reflects readiness (503 only on critical failure).
//
// The probe routes intentionally bypass the site middleware chain: orchestrator
// probes need no request IDs, security headers, or compression, and keeping
// them off the access log avoids polluting it with sweep-driven checks.
// A nil probe returns next unchanged (direct-construction test containers).
//
// The dev-only dashboard (dashboard_dev.go; no-op stub in prod) is the one
// exception: it is a BROWSER surface — HTML, embedded scripts, SSE — so it
// runs behind securityHeadersMiddleware. The middleware scopes its
// Content-Security-Policy to the dashboard prefix and grants the
// 'unsafe-eval' the embedded Datastar SDK needs (Function-constructor
// expression compilation; without it the SDK dies at init and the SSE
// connection never opens). In prod builds the two registered prefix routes
// answer a bare 404 instead of falling through to the site handler — they
// are dev-only paths with no prod contract.
func WithHealthProbes(next http.Handler, probe *health.Probe) http.Handler {
	if probe == nil {
		return next
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", probe.ReadinessHandler())
	probe.RegisterRoutes(mux, health.DefaultRoutes())

	dashboardMux := http.NewServeMux()
	mountHealthDashboard(dashboardMux, probe)
	dashboardHandler := securityHeadersMiddleware()(dashboardMux)
	mux.Handle("GET /debug/health-dashboard", dashboardHandler)
	mux.Handle("GET /debug/health-dashboard/", dashboardHandler)

	mux.Handle("/", next)

	return mux
}
