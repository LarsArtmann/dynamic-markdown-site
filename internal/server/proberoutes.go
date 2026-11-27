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
// Every route mounted here is recorded in the site handler's route registry
// (handlerRoutes), so the site's GET /endpoints listing covers the probe mux
// too: a request for the listing falls through the "/" catch-all to the site
// handler, which serves it from inside the site middleware chain.
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
	routes := handlerRoutes(next)

	// The declarations below ARE the probe mux's route table: each one mounts
	// its handler AND records it for the GET /endpoints listing, exactly like
	// the site routes in Server.Handler.
	mount(routes, mux, "GET /health", probe.ReadinessHandler(), "Readiness alias of /readyz (JSON; Docker HEALTHCHECK target)")

	probeRoutes := health.DefaultRoutes()
	probe.RegisterRoutes(mux, probeRoutes)
	record(routes, probeRoutes.Liveness, "Liveness probe (go-health JSON): always 200 while serving")
	record(routes, probeRoutes.Readiness, "Readiness probe (go-health JSON): 503 when a critical service fails")
	record(routes, probeRoutes.Startup, "Startup probe (go-health JSON): 503 until the first passing sweep")
	record(routes, probeRoutes.Version, "Build version (go-health JSON)")

	dashboardMux := http.NewServeMux()
	mountHealthDashboard(dashboardMux, probe)
	dashboardHandler := securityHeadersMiddleware()(dashboardMux)
	mustMount(routes.Handle(
		mux,
		"GET /debug/health-dashboard",
		dashboardHandler,
		"Health dashboard HTML page (dev builds only; 404 in release builds)",
	))
	mustMount(routes.Handle(
		mux,
		"GET /debug/health-dashboard/",
		dashboardHandler,
		"Health dashboard subtree (dev builds only; 404 in release builds): sse, favicon.svg, datastar.js, healthz, readyz, startupz",
	))

	mustMount(routes.Handle(mux, "/", next, "Catch-all: every other request goes to the site handler (routes listed here too)"))

	return mux
}
