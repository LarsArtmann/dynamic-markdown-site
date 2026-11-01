//go:build dev

package server

import (
	"context"
	"net/http"

	"github.com/larsartmann/go-health"
	"github.com/larsartmann/go-health-dashboard"
)

// healthDashboardRoutes are the dev-only dashboard paths. They must not
// collide with the orchestrator probe routes WithHealthProbes owns
// (/health, /healthz, /readyz, /startupz): duplicate ServeMux patterns
// panic at registration, and the dashboard's default /health HTML page
// would shadow the readiness alias that the healthcheck subcommand and
// Docker HEALTHCHECK depend on (ADR-0005). The dashboard's own kubelet
// JSON mirrors live under the debug prefix: go-health-dashboard v0.10.2
// registers them unconditionally (empty-string disabling arrived in
// v0.12.x, which would force a templ version bump - gotcha 8).
var healthDashboardRoutes = dashboard.Routes{
	Dashboard:  "/debug/health-dashboard",
	SSE:        "/debug/health-dashboard/sse",
	Favicon:    "/debug/health-dashboard/favicon.svg",
	DatastarJS: "/debug/health-dashboard/datastar.js",
	Liveness:   "/debug/health-dashboard/healthz",
	Readiness:  "/debug/health-dashboard/readyz",
	Startup:    "/debug/health-dashboard/startupz",
	Healthz:    "",
	Metrics:    "",
	Trend:      "",
	Export:     "",
	Introspect: "",
}

// mountHealthDashboard mounts the go-health-dashboard HTML page over this
// server's probe in dev builds. The SSE pusher runs for the dev process
// lifetime on a background context: dev binaries are killed by the dev loop
// or Ctrl-C and are never long-lived daemons, so explicit teardown is not
// wired (the dashboard stays dev-only; plan M8/F43-F44). A nil probe (test
// containers) mounts nothing.
func mountHealthDashboard(mux *http.ServeMux, probe *health.Probe) {
	if probe == nil {
		return
	}

	dash := dashboard.New(
		probe,
		dashboard.WithTitle("Dynamic Markdown Site - Health"),
		dashboard.WithRoutes(healthDashboardRoutes),
		dashboard.WithEmbeddedDatastarSDK(),
	)

	dash.RegisterRoutes(mux)

	_ = dash.Start(context.Background()) //nolint:errcheck // Start is fire-and-forget: it returns nil today and the pusher only logs
}
