package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

type cacheStatsPayload struct {
	Status     string  `json:"status"`
	Hits       uint64  `json:"hits"`
	Misses     uint64  `json:"misses"`
	Evictions  uint64  `json:"evictions"`
	HitRatio   float64 `json:"hitRatio"`
	Requests   uint64  `json:"requests"`
	Size       uint64  `json:"size"`
	LoadDur    string  `json:"loadDuration"`
	LoadSucc   uint64  `json:"loadSuccess"`
	LoadFailed uint64  `json:"loadFailures"`
}

func TestCacheStatsEndpointReturnsJSON(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)

	rec := httptest.NewRecorder()
	newTestHandler(srv).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/cache/stats", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload cacheStatsPayload
	decodeJSON(t, rec, &payload)

	if payload.Status != "success" {
		t.Errorf("status = %q, want success", payload.Status)
	}
}

func TestResponseTimeHeaderIsSet(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)

	rec := httptest.NewRecorder()
	newTestHandler(srv).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil),
	)

	header := rec.Header().Get("X-Response-Time")
	if header == "" {
		t.Errorf("expected X-Response-Time header to be set")
	}
}
