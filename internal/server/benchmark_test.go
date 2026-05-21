package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
)

func newBenchmarkHandler(b *testing.B) http.Handler {
	b.Helper()

	repo := content.NewInMemoryRepository()

	const fileCount = 50
	for i := range fileCount {
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
	srv := NewServer(
		repo, searcher, logger, htmlCache,
		renderer.NewGoldmarkRenderer(), false, "Site",
	)

	return srv.Handler()
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

func BenchmarkServerRootRequest(b *testing.B) {
	handler := newBenchmarkHandler(b)
	runBenchmarkRequest(b, handler, "/")
}

func BenchmarkServerFileRequest(b *testing.B) {
	handler := newBenchmarkHandler(b)

	paths := []string{"/file-0", "/file-1", "/file-2", "/file-3", "/file-4"}

	for _, path := range paths {
		b.Run("path_"+path, func(b *testing.B) {
			runBenchmarkRequest(b, handler, path)
		})
	}
}

func BenchmarkServerCachedRequest(b *testing.B) {
	handler := newBenchmarkHandler(b)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/file-0", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	runBenchmarkRequest(b, handler, "/file-0")
}

func runBenchmarkRequest(b *testing.B, handler http.Handler, path string) {
	b.Helper()
	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkServerNotFound(b *testing.B) {
	handler := newBenchmarkHandler(b)
	runBenchmarkRequest(b, handler, "/nonexistent-file")
}

func BenchmarkServerHealthCheck(b *testing.B) {
	handler := newBenchmarkHandler(b)
	runBenchmarkRequest(b, handler, "/health")
}

func BenchmarkServerConcurrent(b *testing.B) {
	handler := newBenchmarkHandler(b)

	paths := []string{"/", "/file-0", "/file-1", "/file-2", "/health"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				paths[i%len(paths)],
				nil,
			)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			i++
		}
	})
}

func BenchmarkServerMixedWorkload(b *testing.B) {
	handler := newBenchmarkHandler(b)

	requests := []struct {
		path   string
		weight int
	}{
		{"/", 10},
		{"/file-0", 30},
		{"/file-1", 25},
		{"/file-2", 20},
		{"/health", 10},
		{"/nonexistent", 5},
	}

	b.ResetTimer()

	for i := range b.N {
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

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
