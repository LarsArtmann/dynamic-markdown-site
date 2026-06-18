package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
)

var errFailingRoot = errors.New("root error")

func executeRequest(handler http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// decodeJSON unmarshals rec.Body into target or fails the test with the
// response body for context. Use it instead of inlining json.Unmarshal +
// t.Fatalf at every endpoint test that decodes a JSON payload.
func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder, target T) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode payload: %v (body=%s)", err, rec.Body.String())
	}
}

// assertEndpointOK runs a GET against the shared endpoint-test handler and
// fails the test if the response is not 200. Use for the trivially happy-path
// endpoint tests; anything with body assertions should use executeRequest
// directly.
func assertEndpointOK(t *testing.T, path string) {
	t.Helper()

	handler := newTestHandlerForEndpointTests(t)
	rec := executeRequest(handler, path)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func newTestServer(t *testing.T, repo content.Repository) *Server {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	htmlCache := cache.NewHTMLCache(100)
	t.Cleanup(htmlCache.Close)
	searcher := content.NewSearcher(repo)
	rndr := renderer.NewGoldmarkRenderer()

	srv := NewServer(repo, searcher, logger, htmlCache, rndr, false, "Site")
	t.Cleanup(srv.Shutdown)

	return srv
}

func newTestHandler(s *Server) http.Handler {
	return s.Handler()
}

func newFailingTestHandler(t *testing.T) http.Handler {
	t.Helper()

	repo := &FailingRepository{}
	logger := slog.New(slog.DiscardHandler)
	htmlCache := cache.NewHTMLCache(100)
	t.Cleanup(htmlCache.Close)
	searcher := content.NewSearcher(repo)
	srv := NewServer(
		repo, searcher, logger, htmlCache,
		renderer.NewGoldmarkRenderer(), false, "Site",
	)
	t.Cleanup(srv.Shutdown)

	return newTestHandler(srv)
}

type httpTestCase struct {
	name       string
	method     string
	path       string
	wantStatus int
	wantBody   string
}

var sharedHTTPTestCases = []httpTestCase{
	{
		name:       "status",
		method:     http.MethodGet,
		path:       "/health",
		wantStatus: http.StatusOK,
		wantBody:   `"status":"healthy"`,
	},
	{
		name:       "version",
		method:     http.MethodGet,
		path:       "/health",
		wantStatus: http.StatusOK,
		wantBody:   `"version":"` + version.Version + `"`,
	},
	{
		name:       "commit",
		method:     http.MethodGet,
		path:       "/health",
		wantStatus: http.StatusOK,
		wantBody:   `"commit":"` + version.Commit + `"`,
	},
	{
		name:       "build_date",
		method:     http.MethodGet,
		path:       "/health",
		wantStatus: http.StatusOK,
		wantBody:   `"build_date"`,
	},
	{
		name:       "timestamp",
		method:     http.MethodGet,
		path:       "/health",
		wantStatus: http.StatusOK,
		wantBody:   `"timestamp"`,
	},
}

func runHTTPTests(t *testing.T, handler http.Handler, tests []httpTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body should contain %q, got: %s", tt.wantBody, rec.Body.String())
			}
		})
	}
}

func newTestHandlerWithRepo(t *testing.T) (http.Handler, *content.InMemoryRepository) {
	t.Helper()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)

	return newTestHandler(srv), repo
}

func newTestHandlerForEndpointTests(t *testing.T) http.Handler {
	t.Helper()

	repo := content.NewInMemoryRepository()
	addTestFile(t, repo, "/guide", "Guide", []byte("# Guide\n\nHello world"), time.Now())
	addTestFile(t, repo, "/about", "About", []byte("# About\n\npage"), time.Now())
	addTestFile(t, repo, "/docs/intro", "Intro", []byte("# Introduction\n\nWelcome"), time.Now())

	dir, err := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Docs", time.Now())
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("failed to get root: %v", err)
	}

	root.AddChild(dir)
	repo.Add(dir)

	srv := newTestServer(t, repo)

	return newTestHandler(srv)
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	handler := newFailingTestHandler(t)
	runHTTPTests(t, handler, sharedHTTPTestCases)
}

func TestHealthEndpointStructure(t *testing.T) {
	t.Parallel()

	handler := newFailingTestHandler(t)
	rec := executeRequest(handler, "/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"healthy"`) {
		t.Errorf("body should contain healthy status, got: %s", body)
	}
}

func TestRootEndpointWithContent(t *testing.T) {
	t.Parallel()

	handler, repo := newTestHandlerWithRepo(t)

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("failed to get root: %v", err)
	}

	addTestFile(t, repo, "/index", "Home", []byte("# Welcome"), time.Now())
	_ = root

	rec := executeRequest(handler, "/")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRootEndpointRepositoryError(t *testing.T) {
	t.Parallel()

	handler := newFailingTestHandler(t)
	rec := executeRequest(handler, "/")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestContentByPath(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"guide", "/guide", http.StatusOK},
		{"about", "/about", http.StatusOK},
		{"docs intro", "/docs/intro", http.StatusOK},
		{"nonexistent", "/nonexistent", http.StatusNotFound},
		{"dotmd redirect", "/guide.md", http.StatusMovedPermanently},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := executeRequest(handler, tt.path)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, path = %s", rec.Code, tt.wantStatus, tt.path)
			}
		})
	}
}

func TestHealthEndpointContentType(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)
	rec := executeRequest(handler, "/health")

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestDirectoryListing(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	dir, err := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Docs", time.Now())
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("failed to get root: %v", err)
	}

	root.AddChild(dir)
	repo.Add(dir)

	file := newTestFileNode(t, "/docs/guide", "Guide", []byte("# Guide"), time.Now())
	repo.Add(file)
	dir.AddChild(file)

	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	rec := executeRequest(handler, "/docs")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestContentDirServingWithReadme(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	addTestFile(t, repo, "/docs/README", "README", []byte("# Docs README"), time.Now())

	root, err := repo.Root()
	if err != nil {
		t.Fatalf("failed to get root: %v", err)
	}

	dir, err := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Docs", time.Now())
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	root.AddChild(dir)
	repo.Add(dir)

	file := newTestFileNode(t, "/docs/README", "README", []byte("# Docs README"), time.Now())
	repo.Add(file)
	dir.AddChild(file)

	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	rec := executeRequest(handler, "/docs")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNotFoundSuggestions(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	addTestFile(t, repo, "/blog", "Blog", []byte("# Blog"), time.Now())
	addTestFile(t, repo, "/about", "About", []byte("# About"), time.Now())

	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	rec := executeRequest(handler, "/blg")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleContentByPathRootNode(t *testing.T) {
	t.Parallel()

	assertEndpointOK(t, "/")
}

func TestHandleContentByPathInvalidPath(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)
	rec := executeRequest(handler, "/valid-path")

	_ = rec
}

func TestHandleContentByPathRawFile(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)
	handler := newTestHandler(srv)

	rec := executeRequest(handler, "/some-image.png")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLiveReloadEndpoint(t *testing.T) {
	t.Parallel()

	handler := newTestHandlerForEndpointTests(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	req := httptest.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"/api/live-reload",
		nil,
	)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait briefly for the SSE handler to start, then cancel to unblock it.
	// The SSE handler blocks on ctx.Done() in an infinite loop.
	cancel()
	<-done

	if rec.Code == http.StatusNotFound {
		t.Error("live-reload endpoint should be registered, got 404")
	}
}

// FailingRepository is a test helper that always returns errors.
type FailingRepository struct {
	refreshError bool
}

func (f *FailingRepository) Get(_ domain.URLPath) (domain.ContentNode, error) {
	return nil, errFailingRoot
}

func (f *FailingRepository) GetRaw(_ domain.URLPath) (*content.RawFile, error) {
	return nil, errFailingRoot
}

func (f *FailingRepository) Root() (*domain.DirectoryNode, error) {
	return nil, errFailingRoot
}

func (f *FailingRepository) LastModified() time.Time {
	return time.Time{}
}

func (f *FailingRepository) Refresh() domain.RefreshResult {
	if f.refreshError {
		return domain.RefreshResult{Error: errFailingRoot.Error()}
	}

	return domain.RefreshResult{Success: true}
}

func (f *FailingRepository) AllPaths() []domain.URLPath {
	return nil
}

// Ensure test content directory exists for benchmark tests.
func init() {
	_ = os.MkdirAll("test-content", 0o755)
}
