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
}

// NewHealthMetrics creates an empty exposition holder.
func NewHealthMetrics() *HealthMetrics {
	return &HealthMetrics{latest: atomic.Value{}}
}

// Observe renders the classified response and publishes it atomically.
func (h *HealthMetrics) Observe(resp health.Response) {
	h.latest.Store(renderHealthExposition(resp))
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
	b.WriteString(fmt.Sprintf("health_up %d\n", statusGauge(resp.Status)))

	b.WriteString("# HELP health_check Whether the named service passes its health check.\n")
	b.WriteString("# TYPE health_check gauge\n")

	names := make([]string, 0, len(resp.Checks))
	for name := range resp.Checks {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		b.WriteString(fmt.Sprintf(
			"health_check{service=%q} %d\n",
			name,
			statusGauge(resp.Checks[name].Status),
		))
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
