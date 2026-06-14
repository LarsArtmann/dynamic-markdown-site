package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchEndpointEmptyQuery(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)
	rec := executeRequest(handler, "/search")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSearchEndpointWithQuery(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/search?q=guide",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSearchEndpointFailingRepository(t *testing.T) {
	t.Parallel()

	handler := newFailingTestHandler(t)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/search?q=anything",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// 500 from the failing search backend is acceptable; 200 means we silently
	// swallowed the error.
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSearchEndpointMethod(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	// Search only allows GET; POST should be a 405.
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/search",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 405 or 404", rec.Code)
	}
}
