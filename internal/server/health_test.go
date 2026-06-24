package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
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

func fetchHealth(t *testing.T, srv *Server) healthPayload {
	t.Helper()

	rec := httptest.NewRecorder()
	newTestHandler(srv).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload healthPayload
	decodeJSON(t, rec, &payload)

	return payload
}

func TestHealthEndpointReportsUptime(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	srv.startedAt = time.Now().Add(-2 * time.Second)

	payload := fetchHealth(t, srv)

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

	payload := fetchHealth(t, srv)

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

	srv := newUnhealthyRepositoryServer(t)

	payload := fetchHealth(t, srv)

	repoHealth, ok := payload.Dependencies["repository"].(map[string]any)
	if !ok {
		t.Fatalf("expected repository dependency report")
	}
	if repoHealth["status"] != "error" {
		t.Errorf("expected repository error, got %v", repoHealth["status"])
	}
}

func newUnhealthyRepositoryServer(t *testing.T) *Server {
	t.Helper()

	htmlCache := cache.NewHTMLCache(10)
	t.Cleanup(htmlCache.Close)

	return newTestServerWithCache(t, &FailingRepository{}, htmlCache)
}
