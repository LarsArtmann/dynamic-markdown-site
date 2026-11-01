package server

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/larsartmann/go-health"
)

// HealthMetrics holds the latest health-probe exposition for /metrics. The
// probe's evaluation hook (WithEvaluationHook) calls Observe after every
// sweep; handleMetrics appends the snapshot, keeping the endpoint
// dependency-free per ADR-0004 — the same stdlib exposition shape go-health's
// prometheus-exposition design prescribes (sorted series, byte-stable
// scrapes, health_ prefix convention).
type HealthMetrics struct {
	latest atomic.Value // string

	// checked is the set of do service names whose types implement a real
	// health check (do.HealthcheckerWithContext). When non-nil, Observe
	// renders only these services: go-health's injector sweep reports a
	// synthetic "pass" for every invoked service that lacks a check, and
	// exposing those as health_check{service=...} 1 would claim verdicts
	// that were never evaluated. Nil (NewHealthMetrics) renders every
	// sweep entry, which is correct for NewChecks-shaped probes whose
	// entries are all real.
	checked map[string]struct{}
}

// NewHealthMetrics creates an exposition holder that renders every probe
// response entry as-is. Use NewHealthMetricsWithChecks for injector-sweep
// probes whose response includes services without real checks.
func NewHealthMetrics() *HealthMetrics {
	return &HealthMetrics{latest: atomic.Value{}, checked: nil}
}

// NewHealthMetricsWithChecks creates an exposition holder scoped to the
// named services — the ones whose types implement a real health check.
func NewHealthMetricsWithChecks(services []string) *HealthMetrics {
	checked := make(map[string]struct{}, len(services))
	for _, name := range services {
		checked[name] = struct{}{}
	}

	return &HealthMetrics{latest: atomic.Value{}, checked: checked}
}

// Observe renders the classified response and publishes it atomically.
// Responses from injector-sweep probes carry synthetic pass entries for
// services without real checks; a checked-scoped holder drops those.
func (h *HealthMetrics) Observe(resp health.Response) {
	h.latest.Store(renderHealthExposition(h.filterChecks(resp)))
}

// filterChecks returns resp unchanged unless the holder is scoped to real
// checks, in which case synthetic sweep entries are dropped from Checks.
// The roll-up status is kept: go-health classifies it from the real checks'
// verdicts, so synthetic entries do not influence it.
func (h *HealthMetrics) filterChecks(resp health.Response) health.Response {
	if h.checked == nil {
		return resp
	}

	filtered := resp
	filtered.Checks = make(map[string]health.Check, len(h.checked))

	for name, check := range resp.Checks {
		if _, isReal := h.checked[name]; isReal {
			filtered.Checks[name] = check
		}
	}

	return filtered
}

// Exposition returns the latest probe metrics, or "" before the first sweep.
func (h *HealthMetrics) Exposition() string {
	if v, ok := h.latest.Load().(string); ok {
		return v
	}

	return ""
}

// renderHealthExposition maps the three-state status onto Prometheus
// conventions: 1 = pass, 0 = fail; warn degrades to 1 (up) — the alerting
// distinction is carried by the per-service health_check series.
func renderHealthExposition(resp health.Response) string {
	var b strings.Builder

	b.WriteString("# HELP health_up Whether this instance reports itself healthy.\n")
	b.WriteString("# TYPE health_up gauge\n")
	fmt.Fprintf(&b, "health_up %d\n", statusGauge(resp.Status))

	b.WriteString("# HELP health_check Whether the named service passes its health check.\n")
	b.WriteString("# TYPE health_check gauge\n")

	names := make([]string, 0, len(resp.Checks))
	for name := range resp.Checks {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(&b, "health_check{service=%q} %d\n",
			name,
			statusGauge(resp.Checks[name].Status))
	}

	return b.String()
}

// statusGauge maps health.Status onto 1 (pass/warn) and 0 (fail/other).
func statusGauge(s health.Status) int {
	if s == health.StatusPass || s == health.StatusWarn {
		return 1
	}

	return 0
}
