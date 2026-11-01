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
func WithHealthProbes(next http.Handler, probe *health.Probe) http.Handler {
	if probe == nil {
		return next
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", probe.ReadinessHandler())
	probe.RegisterRoutes(mux, health.DefaultRoutes())
	mountHealthDashboard(mux, probe)
	mux.Handle("/", next)

	return mux
}
