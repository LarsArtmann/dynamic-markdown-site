package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newBenchmarkServer creates a server with in-memory repository for benchmarking.
func newBenchmarkServer(b *testing.B, fileCount int) (*Server, *gin.Engine) {
	b.Helper()

	repo := content.NewInMemoryRepository()

	// Add some files for realistic benchmarking
	for i := range fileCount {
		_ = fileCount // Suppress unused variable warning (always 50 in practice)
		path := domain.MustURLPath("/file-" + string(rune('0'+i%10)))
		contentBytes := []byte(createBenchmarkFileContent(i))

		fileNode, err := domain.NewFileNode(
			path,
			"File "+string(rune('0'+i%10)),
			contentBytes,
			time.Now(),
			uint64(len(contentBytes)),
		)
		if err != nil {
			b.Fatalf("failed to create file node: %v", err)
		}

		repo.Add(fileNode)
	}

	logger := slog.New(slog.DiscardHandler)
	htmlCache := cache.NewHTMLCache(1000)
	searcher := content.NewSearcher(repo)
	srv := NewServer(repo, searcher, logger, htmlCache)

	router := gin.New()
	srv.RegisterRoutes(router)

	return srv, router
}

func createBenchmarkFileContent(index int) string {
	return `---
title: Benchmark File ` + string(rune('0'+index%10)) + `
description: Benchmark test file
---

# Benchmark File

This is benchmark file number ` + string(rune('0'+index%10)) + `.

## Section 1

Some content here with *italic* and **bold** text.

## Section 2: Code

` + "```go" + `
package main

func main() {
    println("Hello, Benchmark!")
}
` + "```" + `

## Section 3: Table

| A | B | C |
|---|---|---|
| 1 | 2 | 3 |

## Conclusion

End of file.
`
}

// BenchmarkServerRootRequest benchmarks requests to the root path.
func BenchmarkServerRootRequest(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)
	runBenchmarkRequest(b, router, "/")
}

// BenchmarkServerFileRequest benchmarks requests for individual files.
func BenchmarkServerFileRequest(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)

	paths := []string{"/file-0", "/file-1", "/file-2", "/file-3", "/file-4"}

	for _, path := range paths {
		b.Run("path_"+path, func(b *testing.B) {
			b.ResetTimer()

			for range b.N {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkServerCachedRequest benchmarks requests that hit the HTML cache.
func BenchmarkServerCachedRequest(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)

	// Prime the cache with a request
	req := httptest.NewRequest(http.MethodGet, "/file-0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/file-0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// runBenchmarkRequest executes a benchmark for a single HTTP GET request to the given path.
func runBenchmarkRequest(b *testing.B, router *gin.Engine, path string) {
	b.Helper()
	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkServerNotFound benchmarks 404 responses.
func BenchmarkServerNotFound(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)
	runBenchmarkRequest(b, router, "/nonexistent-file")
}

// BenchmarkServerHealthCheck benchmarks the health endpoint.
func BenchmarkServerHealthCheck(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)
	runBenchmarkRequest(b, router, "/health")
}

// BenchmarkServerConcurrent benchmarks concurrent requests.
func BenchmarkServerConcurrent(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)

	paths := []string{"/", "/file-0", "/file-1", "/file-2", "/health"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, paths[i%len(paths)], nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			i++
		}
	})
}

// BenchmarkServerMixedWorkload simulates realistic traffic mix.
func BenchmarkServerMixedWorkload(b *testing.B) {
	_, router := newBenchmarkServer(b, 50)

	// Mix of different request types with weights
	requests := []struct {
		path   string
		weight int
	}{
		{"/", 10},       // Root directory listing - frequent
		{"/file-0", 30}, // File viewing - most common
		{"/file-1", 25},
		{"/file-2", 20},
		{"/health", 10},     // Health checks - periodic
		{"/nonexistent", 5}, // 404s - occasional
	}

	b.ResetTimer()

	for i := range b.N {
		// Weighted selection based on realistic traffic distribution
		var path string

		r := i % 100

		cumulative := 0
		for _, req := range requests {
			cumulative += req.weight
			if r < cumulative {
				path = req.path

				break
			}
		}

		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
