package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
	"github.com/larsartmann/go-health"
)

// TestMetricsExposeProbeVerdict pins the evaluation-hook seam: after a probe
// sweep, /metrics carries the health_up and health_check gauges (ADR-0004 —
// stdlib exposition, no client_golang).
func TestMetricsExposeProbeVerdict(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	htmlCache := cache.NewHTMLCache(10)
	t.Cleanup(htmlCache.Close)

	healthMetrics := NewHealthMetrics()
	srv := NewServer(
		repo,
		content.NewSearcher(repo),
		slog.New(slog.DiscardHandler),
		htmlCache,
		renderer.NewGoldmarkRendererWithDiagrams(&renderer.DiagramRenderer{}),
		healthMetrics,
		false,
		"Site",
	)
	t.Cleanup(srv.Shutdown)

	probe := health.NewChecks(
		map[string]health.CheckFunc{
			"repository": repositoryCheck(repo),
		},
		health.WithEvaluationHook(healthMetrics.Observe),
		health.WithRefreshInterval(time.Hour),
	)
	if err := probe.Start(context.Background()); err != nil {
		t.Fatalf("probe.Start() error: %v", err)
	}

	t.Cleanup(probe.Shutdown)

	rec := executeRequest(newTestHandler(srv), "/metrics")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()

	for _, want := range []string{
		"# TYPE health_up gauge",
		"health_up 1",
		`health_check{service="repository"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics body missing %q", want)
		}
	}
}
