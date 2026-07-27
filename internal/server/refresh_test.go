package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRefreshEndpointGet(t *testing.T) {
	t.Parallel()

	assertEndpointOK(t, "/refresh")
}

func TestRefreshEndpointPost(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRefreshRateLimit(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerFor(t)

	var ok, limited int
	for range 15 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(
			rec,
			httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/refresh", nil),
		)

		switch rec.Code {
		case http.StatusOK:
			ok++
		case http.StatusTooManyRequests:
			limited++
		default:
			t.Fatalf("unexpected status %d from /refresh", rec.Code)
		}
	}

	if ok != 10 {
		t.Errorf("allowed refresh requests = %d, want 10", ok)
	}

	if limited != 5 {
		t.Errorf("rate-limited refresh requests = %d, want 5", limited)
	}
}

func TestRefreshEndpointFailing(t *testing.T) {
	t.Parallel()

	repo := &FailingRepository{refreshError: true}
	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/refresh", nil),
	)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
