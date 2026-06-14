package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
)

type healthPayload struct {
	Status       string         `json:"status"`
	Version      string         `json:"version"`
	Commit       string         `json:"commit"`
	BuildDate    string         `json:"buildDate"`
	Timestamp    time.Time      `json:"timestamp"`
	Uptime       string         `json:"uptime"`
	Dependencies map[string]any `json:"dependencies"`
}

func TestHealthEndpointReportsUptime(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	srv.startedAt = time.Now().Add(-2 * time.Second)

	rec := httptest.NewRecorder()
	newTestHandler(srv).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload healthPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v (body=%s)", err, rec.Body.String())
	}

	if payload.Uptime == "" {
		t.Errorf("expected uptime in response, got empty")
	}

	dur, err := time.ParseDuration(payload.Uptime)
	if err != nil {
		t.Errorf("uptime %q is not a duration: %v", payload.Uptime, err)
	}
	if dur < time.Second {
		t.Errorf("uptime = %v, want >= 1s", dur)
	}
}

func TestHealthEndpointReportsDependencies(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)

	rec := httptest.NewRecorder()
	newTestHandler(srv).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload healthPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	repoHealth, ok := payload.Dependencies["repository"].(map[string]any)
	if !ok {
		t.Fatalf("expected repository dependency report, got %+v", payload.Dependencies)
	}
	if repoHealth["status"] != "healthy" {
		t.Errorf("expected repository healthy, got %v", repoHealth["status"])
	}

	cacheHealth, ok := payload.Dependencies["cache"].(map[string]any)
	if !ok {
		t.Fatalf("expected cache dependency report, got %+v", payload.Dependencies)
	}
	if cacheHealth["status"] != "healthy" {
		t.Errorf("expected cache healthy, got %v", cacheHealth["status"])
	}
}

func TestHealthEndpointReportsUnhealthyRepository(t *testing.T) {
	t.Parallel()

	repo := &FailingRepository{}
	logger := slog.New(slog.DiscardHandler)
	htmlCache := cache.NewHTMLCache(10)
	t.Cleanup(htmlCache.Close)

	srv := NewServer(
		repo,
		content.NewSearcher(repo),
		logger,
		htmlCache,
		renderer.NewGoldmarkRenderer(),
		false,
		"Site",
	)
	t.Cleanup(srv.Shutdown)

	rec := httptest.NewRecorder()
	newTestHandler(srv).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload healthPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	repoHealth, ok := payload.Dependencies["repository"].(map[string]any)
	if !ok {
		t.Fatalf("expected repository dependency report")
	}
	if repoHealth["status"] != "error" {
		t.Errorf("expected repository error, got %v", repoHealth["status"])
	}
}
