package content

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// createBenchmarkTestContent creates a temporary directory with sample markdown files.
func createBenchmarkTestContent(b *testing.B, fileCount int) string {
	b.Helper()

	dir := b.TempDir()

	// Create nested directory structure
	for i := range fileCount {
		depth := i % 5 // Distribute files across 5 directory levels

		subDir := dir
		for range depth {
			subDir = filepath.Join(subDir, "subdir")
		}

		err := os.MkdirAll(subDir, 0o750)
		if err != nil {
			b.Fatalf("failed to create subdir: %v", err)
		}

		// Create markdown file with varied content
		content := createBenchmarkMarkdown(i)

		filename := filepath.Join(subDir, "file-"+padNum(i)+".md")

		err = os.WriteFile(filename, []byte(content), 0o644)
		if err != nil {
			b.Fatalf("failed to write file: %v", err)
		}
	}

	return dir
}

func createBenchmarkMarkdown(index int) string {
	return `---
title: Benchmark File ` + padNum(index) + `
description: This is a benchmark test file for performance testing
author: Test Author
tags:
  - benchmark
  - testing
  - performance
date: 2026-02-20
---

# Benchmark File ` + padNum(index) + `

This is a benchmark test file with various markdown elements.

## Section 1

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

### Subsection 1.1

- Item 1
- Item 2
- Item 3
- Item 4
- Item 5

## Section 2: Code Examples

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, Benchmark!")
    for i := 0; i < 10; i++ {
        fmt.Printf("Iteration %d\n", i)
    }
}
` + "```" + `

## Section 3: Tables

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data 1   | Data 2   | Data 3   |
| Data 4   | Data 5   | Data 6   |
| Data 7   | Data 8   | Data 9   |

## Section 4: Links and Images

[Link to example](https://example.com)

![Alt text](https://example.com/image.png)

## Section 5: Blockquotes

> This is a blockquote.
> It can span multiple lines.
>
> And have multiple paragraphs.

## Conclusion

This file contains ` + padNum(index) + ` unique identifiers for testing.
`
}

// padNum converts a number to a 3-digit zero-padded string.
func padNum(n int) string {
	return fmt.Sprintf("%03d", n)
}

// newBenchmarkRepository creates a refreshed filesystem repository backed by
// a freshly-generated benchmark content directory of the requested size.
func newBenchmarkRepository(b *testing.B, fileCount int) *FileSystemRepository {
	b.Helper()

	dir := createBenchmarkTestContent(b, fileCount)

	repo, err := NewFileSystemRepository(dir)
	if err != nil {
		b.Fatalf("failed to create repository: %v", err)
	}

	repo.Refresh()

	return repo
}

// BenchmarkRepositoryRefresh benchmarks the content tree refresh operation.
func BenchmarkRepositoryRefresh(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files_%d", size), func(b *testing.B) {
			repo := newBenchmarkRepository(b, size)

			b.ResetTimer()

			for range b.N {
				repo.Refresh()
			}
		})
	}
}

// BenchmarkRepositoryGet benchmarks content retrieval by path.
func BenchmarkRepositoryGet(b *testing.B) {
	repo := newBenchmarkRepository(b, 100)

	paths := []domain.URLPath{
		"/",
		"/file-000",
		"/file-050",
		"/file-099",
		"/subdir/file-010",
		"/subdir/subdir/file-020",
	}

	for _, path := range paths {
		b.Run("path_"+path.String(), func(b *testing.B) {
			b.ResetTimer()

			for range b.N {
				_, _ = repo.Get(path)
			}
		})
	}
}

// BenchmarkRepositoryRoot benchmarks getting the root directory.
func BenchmarkRepositoryRoot(b *testing.B) {
	repo := newBenchmarkRepository(b, 100)

	for b.Loop() {
		_, _ = repo.Root()
	}
}

// BenchmarkRepositoryRefreshConcurrent benchmarks concurrent refresh operations.
func BenchmarkRepositoryRefreshConcurrent(b *testing.B) {
	repo := newBenchmarkRepository(b, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			repo.Refresh()
		}
	})
}

// BenchmarkRepositoryGetConcurrent benchmarks concurrent content retrieval.
func BenchmarkRepositoryGetConcurrent(b *testing.B) {
	repo := newBenchmarkRepository(b, 100)

	paths := []domain.URLPath{
		"/",
		"/file-000",
		"/file-025",
		"/file-050",
		"/file-075",
		"/file-099",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = repo.Get(paths[i%len(paths)])
			i++
		}
	})
}

// BenchmarkSearcherSearch benchmarks search performance with different content sizes.
func BenchmarkSearcherSearch(b *testing.B) {
	sizes := []int{10, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files_%d", size), func(b *testing.B) {
			dir := createBenchmarkTestContent(b, size)

			repo, err := NewFileSystemRepository(dir)
			if err != nil {
				b.Fatalf("failed to create repository: %v", err)
			}

			searcher := NewSearcher(repo)

			queries := []string{
				"benchmark",   // Title match
				"Lorem ipsum", // Content match
				"file-001",    // Specific title
				"nonexistent", // No matches
				"Go code",     // Code block content
			}

			b.ResetTimer()

			for range b.N {
				for _, query := range queries {
					_, _ = searcher.Search(query)
				}
			}
		})
	}
}

// runSearchBenchmark creates a repository with test content and benchmarks the search.
func runSearchBenchmark(b *testing.B, query string) {
	b.Helper()

	dir := createBenchmarkTestContent(b, 100)

	repo, err := NewFileSystemRepository(dir)
	if err != nil {
		b.Fatalf("failed to create repository: %v", err)
	}

	searcher := NewSearcher(repo)

	for b.Loop() {
		_, _ = searcher.Search(query)
	}
}

// BenchmarkSearcherSearchTitleOnly benchmarks title-only searches.
func BenchmarkSearcherSearchTitleOnly(b *testing.B) {
	runSearchBenchmark(b, "benchmark")
}

// BenchmarkSearcherSearchContentOnly benchmarks content-only searches.
func BenchmarkSearcherSearchContentOnly(b *testing.B) {
	runSearchBenchmark(b, "Lorem ipsum dolor sit amet")
}
