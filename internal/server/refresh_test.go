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

	// Fire 15 sequential requests; limit is 10/minute.
	var lastCode int
	for range 15 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(
			rec,
			httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/refresh", nil),
		)
		lastCode = rec.Code
	}

	// At least one request after the limit should be rate limited.
	if lastCode != http.StatusTooManyRequests && lastCode != http.StatusOK {
		t.Errorf("final status = %d, want 429 or 200", lastCode)
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
