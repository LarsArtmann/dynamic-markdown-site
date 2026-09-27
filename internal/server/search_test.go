package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

func TestSearchEndpointEmptyQuery(t *testing.T) {
	t.Parallel()

	assertEndpointOK(t, "/search")
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

func newSeededSearchServer(t *testing.T, fileCount int) http.Handler {
	t.Helper()

	repo := content.NewInMemoryRepository()

	for i := range fileCount {
		addTestFile(
			t,
			repo,
			"/doc"+string(rune('a'+i)),
			"Paginated Doc",
			[]byte("# Paginated Doc\n\nfindme "+string(rune('a'+i))),
			time.Now(),
		)
	}

	return newTestHandler(newTestServer(t, repo))
}

func TestSearchRateLimitDeniedAfterBudget(t *testing.T) {
	t.Parallel()

	handler := newSeededSearchServer(t, 1)

	for i := range searchRateLimit {
		rec := executeRequest(handler, "/search?q=findme")

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d (body=%s)", i+1, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	rec := executeRequest(handler, "/search?q=findme")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("request beyond budget: status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestSearchPaginationSlicesResults(t *testing.T) {
	t.Parallel()

	handler := newSeededSearchServer(t, 5)

	rec := executeRequest(handler, "/search?q=findme&page=2&pageSize=2")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := strings.Count(rec.Body.String(), "search-result-card"); got != 2 {
		t.Errorf("page 2 result cards = %d, want 2", got)
	}

	if !strings.Contains(rec.Body.String(), "Page 2 of 3") {
		t.Error("page 2 of 3 indicator missing from response")
	}
}

func TestSearchPaginationClampsOutOfRangePages(t *testing.T) {
	t.Parallel()

	handler := newSeededSearchServer(t, 5)

	rec := executeRequest(handler, "/search?q=findme&page=99&pageSize=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("deep page status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), "Page 3 of 3") {
		t.Error("page 99 should clamp to the last page (3 of 3)")
	}

	rec = executeRequest(handler, "/search?q=findme&page=0&pageSize=2")
	if !strings.Contains(rec.Body.String(), "Page 1 of 3") {
		t.Error("page 0 should clamp to the first page (1 of 3)")
	}
}

func TestSearchPaginationEmptyQueryIgnoresParams(t *testing.T) {
	t.Parallel()

	handler := newSeededSearchServer(t, 2)

	rec := executeRequest(handler, "/search?page=7&pageSize=999")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
