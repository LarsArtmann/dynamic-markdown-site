package testutil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"log/slog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// HTTPTestCase defines a single HTTP test case.
type HTTPTestCase struct {
	Name       string
	Method     string
	Path       string
	Body       string
	WantStatus int
	WantBody   string
}

// HTTPTestRunner provides utilities for running HTTP tests.
type HTTPTestRunner struct {
	Router *gin.Engine
}

// NewHTTPTestRunner creates a new HTTP test runner with the given router.
func NewHTTPTestRunner(router *gin.Engine) *HTTPTestRunner {
	return &HTTPTestRunner{Router: router}
}

// Run executes a slice of HTTP test cases.
func (r *HTTPTestRunner) Run(t *testing.T, cases []HTTPTestCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.Method, tc.Path, strings.NewReader(tc.Body))
			rec := httptest.NewRecorder()

			r.Router.ServeHTTP(rec, req)

			if rec.Code != tc.WantStatus {
				t.Errorf("Expected status %d, got %d", tc.WantStatus, rec.Code)
			}

			if tc.WantBody != "" && !strings.Contains(rec.Body.String(), tc.WantBody) {
				t.Errorf("Expected body to contain %q, got %q", tc.WantBody, rec.Body.String())
			}
		})
	}
}

// ServerFixture provides test fixtures for server testing.
type ServerFixture struct {
	Repository content.Repository
	Cache      *cache.HTMLCache
	Server     *server.Server
}

// NewServerFixture creates a new server fixture with an in-memory repository.
func NewServerFixture(t *testing.T) *ServerFixture {
	t.Helper()

	repo := content.NewInMemoryRepository()
	c := cache.NewHTMLCache(100)
	searcher := content.NewSearcher(repo)
	logger := slog.New(slog.DiscardHandler)
	s := server.NewServer(repo, searcher, logger, c, false)

	return &ServerFixture{
		Repository: repo,
		Cache:      c,
		Server:     s,
	}
}

// NewRouter creates a gin router with the server routes registered.
func (f *ServerFixture) NewRouter() *gin.Engine {
	router := gin.New()
	f.Server.RegisterRoutes(router)

	return router
}

// NewRequest creates a new HTTP request for testing.
func (f *ServerFixture) NewRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

// Execute executes a request and returns the response recorder.
func (f *ServerFixture) Execute(router *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
