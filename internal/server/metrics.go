package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleMetrics returns a Prometheus-compatible text exposition of cache
// counters and the request counter tracked by the access log middleware.
//
// We intentionally keep this endpoint dependency-free — it does not pull in
// github.com/prometheus/client_golang — because the server's observability
// needs are small (a handful of counters) and the dependency surface would
// be larger than the value provided.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	stats := s.cache.Stats()

	w.Header().Set(headerContentType, "text/plain; version=0.0.4; charset=utf-8")

	var b strings.Builder

	// Cache hit/miss counters.
	b.WriteString("# HELP dynamic_markdown_site_cache_hits_total Number of cache hits.\n")
	b.WriteString("# TYPE dynamic_markdown_site_cache_hits_total counter\n")
	fmt.Fprintf(&b, "dynamic_markdown_site_cache_hits_total %d\n", stats.Hits)

	b.WriteString("# HELP dynamic_markdown_site_cache_misses_total Number of cache misses.\n")
	b.WriteString("# TYPE dynamic_markdown_site_cache_misses_total counter\n")
	fmt.Fprintf(&b, "dynamic_markdown_site_cache_misses_total %d\n", stats.Misses)

	b.WriteString("# HELP dynamic_markdown_site_cache_evictions_total Number of cache evictions.\n")
	b.WriteString("# TYPE dynamic_markdown_site_cache_evictions_total counter\n")
	fmt.Fprintf(&b, "dynamic_markdown_site_cache_evictions_total %d\n", stats.Evictions)

	b.WriteString("# HELP dynamic_markdown_site_cache_size Current cache entry count.\n")
	b.WriteString("# TYPE dynamic_markdown_site_cache_size gauge\n")
	fmt.Fprintf(&b, "dynamic_markdown_site_cache_size %d\n", s.cache.EstimatedSize())

	b.WriteString("# HELP dynamic_markdown_site_cache_hit_ratio Cache hit ratio in [0, 1].\n")
	b.WriteString("# TYPE dynamic_markdown_site_cache_hit_ratio gauge\n")
	fmt.Fprintf(&b, "dynamic_markdown_site_cache_hit_ratio %s\n",
		strconv.FormatFloat(stats.HitRatio(), 'f', 4, 64))

	// Uptime gauge (seconds).
	b.WriteString("# HELP dynamic_markdown_site_uptime_seconds Server uptime in seconds.\n")
	b.WriteString("# TYPE dynamic_markdown_site_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "dynamic_markdown_site_uptime_seconds %s\n",
		strconv.FormatFloat(time.Since(s.startedAt).Seconds(), 'f', 3, 64))

	_, _ = w.Write([]byte(b.String()))
}
