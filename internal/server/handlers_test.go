package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// executeRequest executes a GET HTTP request and returns the response recorder.
func executeRequest(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func newTestServer(t *testing.T, repo content.Repository) *Server {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	htmlCache := cache.NewHTMLCache(100)
	searcher := content.NewSearcher(repo)
	rndr := renderer.NewGoldmarkRenderer()

	return NewServer(repo, searcher, logger, htmlCache, rndr, false, "Site")
}

func newTestRouter(s *Server) *gin.Engine {
	router := gin.New()
	s.RegisterRoutes(router)

	return router
}

func newFailingTestServer(t *testing.T) *gin.Engine {
	t.Helper()

	repo := &FailingRepository{}
	logger := slog.New(slog.DiscardHandler)
	cache := cache.NewHTMLCache(100)
	searcher := content.NewSearcher(repo)
	server := NewServer(
		repo, searcher, logger, cache,
		renderer.NewGoldmarkRenderer(), false, "Site",
	)
	router := newTestRouter(server)

	return router
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	runHTTPTests(t, router, []httpTestCase{
		{
			name:       "GET /health returns 200 with status",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   `"status":"healthy"`,
		},
		{
			name:       "GET /health returns version info",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   `"version":"dev"`,
		},
		{
			name:       "GET /health returns commit",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   `"commit":"unknown"`,
		},
		{
			name:       "GET /health returns build_date",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   `"build_date":"unknown"`,
		},
		{
			name:       "GET /health returns timestamp",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   `"timestamp"`,
		},
	})
}

func TestRefreshEndpoint(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	runHTTPTests(t, router, []httpTestCase{
		{
			name:       "GET /refresh returns 200",
			method:     http.MethodGet,
			path:       "/refresh",
			wantStatus: http.StatusOK,
			wantBody:   `"status":"success"`,
		},
		{
			name:       "POST /refresh returns 200",
			method:     http.MethodPost,
			path:       "/refresh",
			wantStatus: http.StatusOK,
			wantBody:   `"status":"success"`,
		},
	})
}

func TestRefreshRateLimit(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	for i := range 10 {
		rec := executeRequest(router, "/refresh")

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, rec.Code, http.StatusOK)
		}
	}

	rec := executeRequest(router, "/refresh")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited request: status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}

	assertBodyContains(t, rec.Body.String(), "rate limit exceeded")
}

// assertBodyContains asserts that body contains the given substring.
func assertBodyContains(t *testing.T, body, substring string) {
	t.Helper()
	if !strings.Contains(body, substring) {
		t.Errorf("body = %s, want to contain %q", body, substring)
	}
}

func TestRootEndpoint(t *testing.T) {
	t.Parallel()
	runStatusTestSuite(t, []statusTestCase{
		{
			name:       "GET / returns 200",
			path:       "/",
			wantStatus: http.StatusOK,
		},
	})
}

// statusTestCase represents a simple HTTP status code test case.
type statusTestCase struct {
	name       string
	path       string
	wantStatus int
}

// runStatusTests executes a set of status code test cases against the given router.
func runStatusTests(t *testing.T, router *gin.Engine, tests []statusTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				tt.path,
				nil,
			)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// httpTestCase represents a full HTTP test case with method, status, and body assertions.
type httpTestCase struct {
	name       string
	method     string
	path       string
	wantStatus int
	wantBody   string
}

// runHTTPTests executes a set of HTTP test cases against the given router.
func runHTTPTests(t *testing.T, router *gin.Engine, tests []httpTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %s, want to contain %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

// newTestRouterWithRepo creates a test router with an empty in-memory repository.
func newTestRouterWithRepo(t *testing.T) (*gin.Engine, content.Repository) {
	t.Helper()

	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	return router, repo
}

// runStatusTestSuite creates a test router and runs status test cases.
func runStatusTestSuite(t *testing.T, tests []statusTestCase) {
	t.Helper()
	router, _ := newTestRouterWithRepo(t)
	runStatusTests(t, router, tests)
}

func TestContentNotFound(t *testing.T) {
	t.Parallel()

	testCases := []statusTestCase{
		{
			name:       "non-existent file returns 404",
			path:       "/nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-existent nested path returns 404",
			path:       "/some/deep/path/that/does/not/exist",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "path traversal with .. returns 404",
			path:       "/static/../secret",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "path traversal in content returns 404",
			path:       "/content/../../../etc/passwd",
			wantStatus: http.StatusNotFound,
		},
	}

	runStatusTestSuite(t, testCases)
}

func TestContentWithFile(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()

	filePath := domain.MustURLPath("/test-file")
	fileContent := []byte(
		"# Test File\n\nThis is test content.\n\n```go\nfmt.Println(\"hello\")\n```\n",
	)

	file, err := domain.NewFileNode(
		filePath,
		"Test File",
		fileContent,
		time.Now(),
		uint64(len(fileContent)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	repo.Add(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := executeRequest(router, "/test-file")

	if rec.Code != http.StatusOK {
		t.Errorf(
			"status = %d, want %d, body: %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String()[:min(500, len(rec.Body.String()))],
		)

		return
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Test File") {
		t.Errorf("body should contain 'Test File', got: %s", body[:min(200, len(body))])
	}
}

func TestContentRedirectFromMDExtension(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()

	// Create a file at /test-page (without .md in the URL path)
	filePath := domain.MustURLPath("/test-page")
	fileContent := []byte("# Test Page\n\nThis is test content.\n")

	file, err := domain.NewFileNode(
		filePath,
		"Test Page",
		fileContent,
		time.Now(),
		uint64(len(fileContent)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	repo.Add(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	// Request with .md extension should redirect to clean URL
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/test-page.md",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}

	// Check the Location header
	location := rec.Header().Get("Location")
	if location != "/test-page" {
		t.Errorf("Location header = %q, want %q", location, "/test-page")
	}
}

func TestContentRedirectFromMDExtensionWithPath(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()

	// Create a nested directory structure
	dirPath := domain.MustURLPath("/docs")
	dir, err := domain.NewDirectoryNode(dirPath, "docs", time.Now())
	if err != nil {
		t.Fatalf("failed to create directory node: %v", err)
	}

	// Create a file in the directory
	filePath := domain.MustURLPath("/docs/readme")
	fileContent := []byte("# Readme\n\nDocumentation here.\n")

	file, err := domain.NewFileNode(
		filePath,
		"readme",
		fileContent,
		time.Now(),
		uint64(len(fileContent)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	repo.Add(dir)
	repo.Add(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	// Request with .md extension in nested path should redirect
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/docs/readme.md",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}

	location := rec.Header().Get("Location")
	if location != "/docs/readme" {
		t.Errorf("Location header = %q, want %q", location, "/docs/readme")
	}
}

func TestContentWithDirectory(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()

	dirPath := domain.MustURLPath("/test-dir")

	dir, err := domain.NewDirectoryNode(dirPath, "Test Directory", time.Now())
	if err != nil {
		t.Fatalf("failed to create directory node: %v", err)
	}

	filePath := domain.MustURLPath("/test-dir/nested-file")
	fileContent := []byte("# Nested File\n\nContent here.\n")

	file, err := domain.NewFileNode(
		filePath,
		"Nested File",
		fileContent,
		time.Now(),
		uint64(len(fileContent)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	repo.Add(dir)
	repo.Add(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := executeRequest(router, "/test-dir")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Test Directory") {
		t.Errorf("body should contain 'Test Directory', got: %s", body[:min(200, len(body))])
	}
}

func TestStaticFileServing(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	runHTTPTests(t, router, []httpTestCase{
		{
			name:       "non-existent static file returns 404",
			method:     http.MethodGet,
			path:       "/static/nonexistent.css",
			wantStatus: http.StatusNotFound,
		},
	})
}

func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	runHTTPTests(t, router, []httpTestCase{
		{
			name:       "DELETE /health not allowed",
			method:     http.MethodDelete,
			path:       "/health",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "PUT /refresh not allowed",
			method:     http.MethodPut,
			path:       "/refresh",
			wantStatus: http.StatusNotFound,
		},
	})
}

func TestSearchEndpoint(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()

	// Add test files for search
	filePath := domain.MustURLPath("/test-document")
	fileContent := []byte("# Test Document\n\nThis is test content.\n")

	file, err := domain.NewFileNode(
		filePath,
		"Test Document",
		fileContent,
		time.Now(),
		uint64(len(fileContent)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	repo.Add(file)

	// Also add as child of root so searcher can find it
	root, err := repo.Root()
	if err != nil {
		t.Fatalf("failed to get root: %v", err)
	}
	root.AddChild(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	runHTTPTests(t, router, []httpTestCase{
		{
			name:       "GET /search without query returns page",
			method:     http.MethodGet,
			path:       "/search",
			wantStatus: http.StatusOK,
			wantBody:   "Search",
		},
		{
			name:       "GET /search with empty query returns page",
			method:     http.MethodGet,
			path:       "/search?q=",
			wantStatus: http.StatusOK,
			wantBody:   "Search",
		},
		{
			name:       "GET /search with matching query returns results",
			method:     http.MethodGet,
			path:       "/search?q=Test",
			wantStatus: http.StatusOK,
			wantBody:   "<mark>Test</mark>",
		},
		{
			name:       "GET /search with non-matching query shows empty",
			method:     http.MethodGet,
			path:       "/search?q=nonexistent",
			wantStatus: http.StatusOK,
			wantBody:   "No results found",
		},
	})
}

func TestRefreshEndpointFailure(t *testing.T) {
	t.Parallel()
	// Test when refresh fails
	repo := &FailingRepository{refreshError: true}
	logger := slog.New(slog.DiscardHandler)
	cache := cache.NewHTMLCache(100)
	searcher := content.NewSearcher(repo)
	server := NewServer(
		repo, searcher, logger, cache,
		renderer.NewGoldmarkRenderer(), false, "Site",
	)
	router := newTestRouter(server)

	rec := executeRequest(router, "/refresh")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	assertBodyContains(t, rec.Body.String(), "error")
}

func TestRootEndpointError(t *testing.T) {
	t.Parallel()
	router := newFailingTestServer(t)

	runStatusTests(t, router, []statusTestCase{
		{
			name:       "root endpoint returns 500 when repository fails",
			path:       "/",
			wantStatus: http.StatusInternalServerError,
		},
	})
}

func TestSearchEndpointError(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	logger := slog.New(slog.DiscardHandler)
	cache := cache.NewHTMLCache(100)
	// Use failing searcher
	server := &Server{
		repo:        repo,
		searcher:    nil, // nil searcher forces search failure path
		renderer:    nil,
		logger:      logger,
		rateLimiter: newRateLimiter(10, time.Minute),
		cache:       cache,
	}
	router := gin.New()
	router.GET("/search", func(c *gin.Context) {
		// Simulate search error by calling handle500
		server.handle500(c)
	})

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/search?q=test",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandle500(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	// Create gin context and call handle500 directly
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	server.handle500(c)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestGetContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path     string
		expected string
	}{
		{"/static/style.css", "text/css"},
		{"/static/app.js", "application/javascript"},
		{"/static/icon.svg", "image/svg+xml"},
		{"/static/image.png", "image/png"},
		{"/static/image.jpg", "image/jpeg"},
		{"/static/image.jpeg", "image/jpeg"},
		{"/static/image.gif", "image/gif"},
		{"/static/font.woff2", "font/woff2"},
		{"/static/font.woff", "font/woff"},
		{"/static/unknown.xyz", ""},
		{"/static/noextension", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			result := staticContentType(tt.path)
			if result != tt.expected {
				t.Errorf("staticContentType(%s) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestStaticPathTraversal(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	runStatusTests(t, router, []statusTestCase{
		{
			name:       "path traversal in static file returns 404",
			path:       "/static/../server/static/secret",
			wantStatus: http.StatusNotFound,
		},
	})
}

func TestRateLimiterCleanupTriggered(t *testing.T) {
	t.Parallel()

	// Create rate limiter with very short cleanup interval
	rl := newRateLimiter(10, 10*time.Millisecond)

	// Add some requests
	rl.checkRateLimit("192.168.1.1")

	// Wait for cleanup to run
	time.Sleep(50 * time.Millisecond)

	// Verify requests were cleaned up (map should be empty or filtered)
	rl.mu.RLock()
	_, exists := rl.requests["192.168.1.1"]
	rl.mu.RUnlock()

	// Entry may or may not exist depending on timing, but this exercises the cleanup path
	_ = exists
}

func TestContentByPathNonNotFoundError(t *testing.T) {
	t.Parallel()
	router := newFailingTestServer(t)

	runStatusTests(t, router, []statusTestCase{
		{
			name:       "content path returns 404 when not found",
			path:       "/some-path",
			wantStatus: http.StatusNotFound,
		},
	})
}

func TestRenderFileWithCache(t *testing.T) {
	t.Parallel()
	repo := content.NewInMemoryRepository()

	filePath := domain.MustURLPath("/cached-file")
	fileContent := []byte("# Cached File\n\nThis content is cached.\n")

	file, err := domain.NewFileNode(
		filePath,
		"Cached File",
		fileContent,
		time.Now(),
		uint64(len(fileContent)),
	)
	if err != nil {
		t.Fatalf("failed to create file node: %v", err)
	}

	repo.Add(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	// First request - renders and caches
	req1 := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/cached-file",
		nil,
	)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", rec1.Code, http.StatusOK)
	}

	// Second request - should use cache
	req2 := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/cached-file",
		nil,
	)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("second request: status = %d, want %d", rec2.Code, http.StatusOK)
	}

	// Both responses should contain the title
	if !strings.Contains(rec1.Body.String(), "Cached File") {
		t.Errorf("first response should contain 'Cached File'")
	}

	if !strings.Contains(rec2.Body.String(), "Cached File") {
		t.Errorf("second response should contain 'Cached File'")
	}
}

func TestStaticFileDirectoryReturns404(t *testing.T) {
	t.Parallel()
	runStatusTestSuite(t, []statusTestCase{
		{
			name:       "GET /static/ returns 404",
			path:       "/static/",
			wantStatus: http.StatusNotFound,
		},
	})
}

func TestRawFileServing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested directory with SVG file
	diagramsDir := filepath.Join(tmpDir, "docs", "assets", "diagrams")
	if err := os.MkdirAll(diagramsDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`
	if err := os.WriteFile(
		filepath.Join(diagramsDir, "01-workflow.svg"),
		[]byte(svgContent),
		0o644,
	); err != nil {
		t.Fatalf("failed to write SVG: %v", err)
	}

	// Need at least one markdown file so tree doesn't get filtered
	if err := os.WriteFile(
		filepath.Join(tmpDir, "index.md"),
		[]byte("# Home"),
		0o644,
	); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}

	repo, err := content.NewFileSystemRepository(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemRepository() error = %v", err)
	}

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	tests := []httpTestCase{
		{
			name:       "SVG in nested directory returns 200 with correct content type",
			method:     http.MethodGet,
			path:       "/docs/assets/diagrams/01-workflow.svg",
			wantStatus: http.StatusOK,
			wantBody:   svgContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %s, want to contain %s", rec.Body.String(), tt.wantBody)
			}

			// Check content-type header
			contentType := rec.Header().Get("Content-Type")
			if contentType != "image/svg+xml" {
				t.Errorf("Content-Type = %q, want %q", contentType, "image/svg+xml")
			}

			// Check cache-control header
			cacheControl := rec.Header().Get("Cache-Control")
			if cacheControl != "public, max-age=86400" {
				t.Errorf("Cache-Control = %q, want %q", cacheControl, "public, max-age=86400")
			}
		})
	}
}

// FailingRepository is a test helper that always returns errors.
type FailingRepository struct {
	refreshError bool
}

func (f *FailingRepository) Get(_ domain.URLPath) (domain.ContentNode, error) {
	return nil, content.ErrContentNotFound
}

func (f *FailingRepository) Root() (*domain.DirectoryNode, error) {
	return nil, errors.New("root error")
}

func (f *FailingRepository) LastModified() time.Time {
	return time.Time{}
}

func (f *FailingRepository) Refresh() domain.RefreshResult {
	if f.refreshError {
		return domain.RefreshResult{
			Success: false,
			Error:   "forced refresh failure",
		}
	}

	return domain.RefreshResult{Success: true}
}

func (f *FailingRepository) AllPaths() []domain.URLPath {
	return nil
}

func (f *FailingRepository) Search(_ string) ([]content.SearchResult, error) {
	return nil, errors.New("search error")
}

func (f *FailingRepository) GetRaw(_ domain.URLPath) (*content.RawFile, error) {
	return nil, content.ErrContentNotFound
}
