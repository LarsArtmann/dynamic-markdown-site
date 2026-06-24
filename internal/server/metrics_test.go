package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestMetricsEndpointReturnsPrometheusFormat(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerFor(t)

	rec := executeRequest(handler, "/metrics")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	expected := []string{
		"# HELP dynamic_markdown_site_cache_hits_total",
		"# TYPE dynamic_markdown_site_cache_hits_total counter",
		"# HELP dynamic_markdown_site_cache_misses_total",
		"# TYPE dynamic_markdown_site_uptime_seconds gauge",
		"dynamic_markdown_site_cache_hits_total 0",
		"dynamic_markdown_site_cache_misses_total 0",
	}
	for _, want := range expected {
		if !strings.Contains(body, want) {
			t.Errorf("metrics body missing %q", want)
		}
	}

	if !strings.Contains(rec.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", rec.Header().Get("Content-Type"))
	}
}
