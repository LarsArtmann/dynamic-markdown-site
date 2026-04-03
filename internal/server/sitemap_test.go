package server

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHost = "example.com"

func TestSitemapXMLEmptyRepo(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/xml")
}

func TestSitemapXMLWithFiles(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	file, err := domain.NewFileNode(
		domain.MustURLPath("/guide"),
		"Guide",
		[]byte("# Guide"),
		time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		100,
	)
	require.NoError(t, err)
	repo.Add(file)

	root, err := repo.Root()
	require.NoError(t, err)
	root.AddChild(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "http://example.com/guide")
	assert.Contains(t, body, "2026-03-15")

	var urlset URLSet
	require.NoError(t, xml.Unmarshal([]byte(body), &urlset))
	require.Len(t, urlset.URLs, 1)
	assert.Equal(t, "http://example.com/guide", urlset.URLs[0].Loc)
}

func TestSitemapXMLWithDirectories(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	dir, err := domain.NewDirectoryNode(
		domain.MustURLPath("/docs"),
		"Docs",
		time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	file, err := domain.NewFileNode(
		domain.MustURLPath("/docs/guide"),
		"Guide",
		[]byte("# Guide"),
		time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		100,
	)
	require.NoError(t, err)

	dir.AddChild(file)

	root, err := repo.Root()
	require.NoError(t, err)
	root.AddChild(dir)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "http://example.com/docs")
	assert.Contains(t, body, "http://example.com/docs/guide")

	var urlset URLSet
	require.NoError(t, xml.Unmarshal([]byte(body), &urlset))
	assert.Len(t, urlset.URLs, 2)
}

func TestSitemapXMLSkipsRootPath(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var urlset URLSet
	require.NoError(t, xml.Unmarshal(rec.Body.Bytes(), &urlset))
	assert.Empty(t, urlset.URLs, "root path should not appear in sitemap")
}

func TestSitemapXMLHasCleanURLs(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	file, err := domain.NewFileNode(
		domain.MustURLPath("/my-page"),
		"My Page",
		[]byte("# My Page"),
		time.Now(),
		50,
	)
	require.NoError(t, err)
	repo.Add(file)

	root, err := repo.Root()
	require.NoError(t, err)
	root.AddChild(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, "http://example.com/my-page")
	assert.NotContains(t, body, ".md", "URLs should not contain .md extension")
}

func TestSitemapXMLHTTPSEnvironment(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	file, err := domain.NewFileNode(
		domain.MustURLPath("/secure"),
		"Secure",
		[]byte("# Secure"),
		time.Now(),
		50,
	)
	require.NoError(t, err)
	repo.Add(file)

	root, err := repo.Root()
	require.NoError(t, err)
	root.AddChild(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, "https://example.com/secure")
}

func TestSitemapXMLRepositoryError(t *testing.T) {
	t.Parallel()

	router := newFailingTestServer(t)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCalculatePriority(t *testing.T) {
	t.Parallel()

	server := newTestServer(t, content.NewInMemoryRepository())

	tests := []struct {
		segments int
		expected float64
	}{
		{0, 1.0},
		{1, 0.8},
		{2, 0.6},
		{3, 0.4},
		{5, 0.4},
	}

	for _, tt := range tests {
		name := "depth"
		if tt.segments > 0 {
			name = strings.Repeat("/a", tt.segments)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var path domain.URLPath
			if tt.segments > 0 {
				path = domain.MustURLPath(strings.Repeat("/a", tt.segments))
			}
			result := server.calculatePriority(path)
			assert.InEpsilon(t, tt.expected, result, 0.001)
		})
	}
}

func TestCalculateChangeFreq(t *testing.T) {
	t.Parallel()

	server := newTestServer(t, content.NewInMemoryRepository())

	dir, err := domain.NewDirectoryNode(domain.MustURLPath("/docs"), "Docs", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "daily", server.calculateChangeFreq(dir))

	oldFile, err := domain.NewFileNode(
		domain.MustURLPath("/old"),
		"Old",
		[]byte("# Old"),
		time.Now().Add(-400*24*time.Hour),
		10,
	)
	require.NoError(t, err)
	assert.Equal(t, "yearly", server.calculateChangeFreq(oldFile))
}

func TestSitemapXMLCacheHeaders(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)
	req.Host = testHost
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "public, max-age=3600", rec.Header().Get("Cache-Control"))
}
