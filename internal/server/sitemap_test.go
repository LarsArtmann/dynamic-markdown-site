package server

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHost = "example.com"

// newTestFileNode creates a new file node without adding it to the repository.
// It fails the test if node creation fails.
func newTestFileNode(
	t *testing.T,
	filePath, title string,
	contentBytes []byte,
	modTime time.Time,
) *domain.FileNode {
	t.Helper()

	file, err := domain.NewFileNode(
		domain.MustURLPath(filePath),
		title,
		contentBytes,
		modTime,
		uint64(len(contentBytes)),
	)
	require.NoError(t, err)

	return file
}

// addTestFile creates a file node, adds it to the repository and the repository's root.
// It fails the test if any operation fails.
func addTestFile(
	t *testing.T,
	repo *content.InMemoryRepository,
	filePath, title string,
	contentBytes []byte,
	modTime time.Time,
) {
	t.Helper()

	file := newTestFileNode(t, filePath, title, contentBytes, modTime)
	repo.Add(file)

	root, err := repo.Root()
	require.NoError(t, err)
	root.AddChild(file)
}

// addTestDir creates a directory node and adds it to the repository's root.
// It fails the test if any operation fails.
func addTestDir(
	t *testing.T,
	repo *content.InMemoryRepository,
	dirPath, title string,
	modTime time.Time,
) *domain.DirectoryNode {
	t.Helper()

	dir, err := domain.NewDirectoryNode(domain.MustURLPath(dirPath), title, modTime)
	require.NoError(t, err)

	root, err := repo.Root()
	require.NoError(t, err)
	root.AddChild(dir)

	return dir
}

func serveSitemap(router *gin.Engine) *httptest.ResponseRecorder {
	return serveSitemapWithProto(router, "")
}

func serveSitemapWithProto(router *gin.Engine, proto string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sitemap.xml", nil)

	req.Host = testHost
	if proto != "" {
		req.Header.Set("X-Forwarded-Proto", proto)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func TestSitemapXMLEmptyRepo(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()
	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := serveSitemap(router)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/xml")
}

func TestSitemapXMLWithFiles(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	addTestFile(
		t,
		repo,
		"/guide",
		"Guide",
		[]byte("# Guide"),
		time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
	)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := serveSitemap(router)

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

	dir := addTestDir(t, repo, "/docs", "Docs", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))

	file := newTestFileNode(
		t,
		"/docs/guide",
		"Guide",
		[]byte("# Guide"),
		time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
	)
	repo.Add(file)
	dir.AddChild(file)

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := serveSitemap(router)

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

	rec := serveSitemap(router)

	assert.Equal(t, http.StatusOK, rec.Code)

	var urlset URLSet
	require.NoError(t, xml.Unmarshal(rec.Body.Bytes(), &urlset))
	assert.Empty(t, urlset.URLs, "root path should not appear in sitemap")
}

func TestSitemapXMLHasCleanURLs(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	addTestFile(t, repo, "/my-page", "My Page", []byte("# My Page"), time.Now())

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := serveSitemap(router)

	body := rec.Body.String()
	assert.Contains(t, body, "http://example.com/my-page")
	assert.NotContains(t, body, ".md", "URLs should not contain .md extension")
}

func TestSitemapXMLHTTPSEnvironment(t *testing.T) {
	t.Parallel()

	repo := content.NewInMemoryRepository()

	addTestFile(t, repo, "/secure", "Secure", []byte("# Secure"), time.Now())

	server := newTestServer(t, repo)
	router := newTestRouter(server)

	rec := serveSitemapWithProto(router, "https")

	body := rec.Body.String()
	assert.Contains(t, body, "https://example.com/secure")
}

func TestSitemapXMLRepositoryError(t *testing.T) {
	t.Parallel()

	router := newFailingTestServer(t)

	rec := serveSitemap(router)

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

	rec := serveSitemap(router)

	assert.Equal(t, "public, max-age=3600", rec.Header().Get("Cache-Control"))
}
