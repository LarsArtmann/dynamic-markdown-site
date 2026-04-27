package cache

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// newTestContent creates a RenderedContent for testing.
func newTestContent(html string) domain.RenderedContent {
	return domain.RenderedContent{
		HTML:     domain.HTML(html),
		TOC:      []domain.TOCItem{{Level: 1, Title: "Test", Anchor: "test"}},
		Metadata: domain.Frontmatter{Title: "Test Title"},
	}
}

// assertContentEqual compares two RenderedContent values for equality.
func assertContentEqual(t *testing.T, got, want domain.RenderedContent) {
	t.Helper()

	if string(got.HTML) != string(want.HTML) {
		t.Errorf("Expected HTML %q, got %q", want.HTML, got.HTML)
	}
}

// assertCacheSize asserts the expected cache size.
func assertCacheSize(t *testing.T, cache *HTMLCache, expected int) {
	t.Helper()

	if size := cache.EstimatedSize(); size != expected {
		t.Errorf("Expected cache size %d, got %d", expected, size)
	}
}

// statsTestCase represents a test case for Stats methods.
type statsTestCase struct {
	name     string
	hits     uint64
	misses   uint64
	expected any
}

// runStatsTests executes table-driven tests for Stats methods.
func runStatsTests(
	t *testing.T,
	tests []statsTestCase,
	getResult func(Stats) any,
	errFormat string,
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := Stats{Hits: tt.hits, Misses: tt.misses}

			result := getResult(stats)
			if result != tt.expected {
				t.Errorf(errFormat, tt.expected, result)
			}
		})
	}
}

func TestNewHTMLCache(t *testing.T) {
	t.Parallel()
	t.Run("creates empty cache", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		if cache == nil {
			t.Fatal("NewHTMLCache returned nil")
		}

		assertCacheSize(t, cache, 0)
	})
}

func TestHTMLCache_Get(t *testing.T) {
	t.Parallel()
	t.Run("returns nil for missing key", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		result := cache.Get("/nonexistent")
		if result != nil {
			t.Errorf("Expected nil for missing key, got %+v", result)
		}
	})

	t.Run("returns stored value after Set", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		content := newTestContent("<h1>Hello</h1>")

		cache.Set("/test", content)
		result := cache.Get("/test")

		if result == nil {
			t.Fatal("Expected non-nil result after Set")
		}

		assertContentEqual(t, *result, content)
	})

	t.Run("returns nil after InvalidateAll", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		content := newTestContent("<h1>Hello</h1>")

		cache.Set("/test", content)
		cache.InvalidateAll()
		result := cache.Get("/test")

		if result != nil {
			t.Errorf("Expected nil after InvalidateAll, got %+v", result)
		}
	})
}

func TestHTMLCache_Set(t *testing.T) {
	t.Parallel()
	t.Run("stores value correctly", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		content := newTestContent("<h1>Test</h1>")

		cache.Set("/path/to/file", content)
		assertCacheSize(t, cache, 1)
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		cache.Set("/test", newTestContent("<h1>First</h1>"))
		cache.Set("/test", newTestContent("<h1>Second</h1>"))

		result := cache.Get("/test")
		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if string(result.HTML) != "<h1>Second</h1>" {
			t.Errorf("Expected 'Second', got %q", result.HTML)
		}
	})

	t.Run("stores multiple entries", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		cache.Set("/a", newTestContent("A"))
		cache.Set("/b", newTestContent("B"))
		cache.Set("/c", newTestContent("C"))

		assertCacheSize(t, cache, 3)
	})
}

func TestHTMLCache_GetOrCompute(t *testing.T) {
	t.Parallel()
	t.Run("computes on miss", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		content := newTestContent("<h1>Computed</h1>")
		computeCalled := false

		result, err := cache.GetOrCompute(
			context.Background(),
			"/test",
			func() (domain.RenderedContent, error) {
				computeCalled = true

				return content, nil
			},
		)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !computeCalled {
			t.Error("Expected compute function to be called")
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		assertContentEqual(t, *result, content)
	})

	t.Run("returns cached on hit", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		originalContent := newTestContent("<h1>Original</h1>")
		cache.Set("/test", originalContent)

		computeCalled := false

		result, err := cache.GetOrCompute(
			context.Background(),
			"/test",
			func() (domain.RenderedContent, error) {
				computeCalled = true

				return newTestContent("<h1>Computed</h1>"), nil
			},
		)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if computeCalled {
			t.Error("Expected compute function NOT to be called on cache hit")
		}

		if string(result.HTML) != string(originalContent.HTML) {
			t.Errorf("Expected original content, got %q", result.HTML)
		}
	})

	t.Run("returns error from compute function", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		expectedErr := assertError("compute failed")

		result, err := cache.GetOrCompute(
			context.Background(),
			"/test",
			func() (domain.RenderedContent, error) {
				return domain.RenderedContent{}, expectedErr
			},
		)
		if err == nil {
			t.Fatal("Expected error from compute function")
		}

		if result != nil {
			t.Errorf("Expected nil result on error, got %+v", result)
		}
	})
}

func TestHTMLCache_Stats(t *testing.T) {
	t.Parallel()
	t.Run("returns zero stats initially", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		stats := cache.Stats()

		if stats.Hits != 0 || stats.Misses != 0 || stats.Evictions != 0 {
			t.Errorf("Expected zero stats, got %+v", stats)
		}
	})

	t.Run("tracks misses", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		cache.Get("/missing1")
		cache.Get("/missing2")

		stats := cache.Stats()
		if stats.Misses != 2 {
			t.Errorf("Expected 2 misses, got %d", stats.Misses)
		}
	})

	t.Run("tracks hits", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)
		cache.Set("/test", newTestContent("test"))

		cache.Get("/test")
		cache.Get("/test")

		stats := cache.Stats()
		if stats.Hits != 2 {
			t.Errorf("Expected 2 hits, got %d", stats.Hits)
		}
	})
}

func TestStats_HitRatio(t *testing.T) {
	t.Parallel()

	tests := []statsTestCase{
		{"zero requests", 0, 0, float64(0)},
		{"all hits", 10, 0, 1.0},
		{"all misses", 0, 10, float64(0)},
		{"50/50", 5, 5, 0.5},
		{"75% hit rate", 75, 25, 0.75},
	}

	runStatsTests(
		t,
		tests,
		func(s Stats) any { return s.HitRatio() },
		"Expected hit ratio %v, got %v",
	)
}

func TestStats_Requests(t *testing.T) {
	t.Parallel()

	tests := []statsTestCase{
		{"zero", 0, 0, uint64(0)},
		{"only hits", 10, 0, uint64(10)},
		{"only misses", 0, 10, uint64(10)},
		{"mixed", 7, 3, uint64(10)},
	}

	runStatsTests(
		t,
		tests,
		func(s Stats) any { return s.Requests() },
		"Expected %v requests, got %v",
	)
}

func TestHTMLCache_EstimatedSize(t *testing.T) {
	t.Parallel()
	t.Run("returns correct size", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		assertCacheSize(t, cache, 0)

		cache.Set("/a", newTestContent("A"))
		assertCacheSize(t, cache, 1)

		cache.Set("/b", newTestContent("B"))
		cache.Set("/c", newTestContent("C"))
		assertCacheSize(t, cache, 3)

		cache.InvalidateAll()
		assertCacheSize(t, cache, 0)
	})
}

func TestHTMLCache_InvalidateAll(t *testing.T) {
	t.Parallel()
	t.Run("clears all entries", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		for i := range 10 {
			cache.Set("/test"+string(rune('0'+i)), newTestContent("content"))
		}

		assertCacheSize(t, cache, 10)

		cache.InvalidateAll()
		assertCacheSize(t, cache, 0)
	})
}

func TestHTMLCache_Concurrent(t *testing.T) {
	t.Parallel()
	t.Run("concurrent Get/Set operations", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(1000)

		var wg sync.WaitGroup

		iterations := 100

		// Concurrent writes
		for i := range iterations {
			wg.Add(1)

			go func(n int) {
				defer wg.Done()

				path := "/test" + string(rune('0'+n%10))
				cache.Set(path, newTestContent("content"))
			}(i)
		}

		// Concurrent reads
		for i := range iterations {
			wg.Add(1)

			go func(n int) {
				defer wg.Done()

				path := "/test" + string(rune('0'+n%10))
				_ = cache.Get(path)
			}(i)
		}

		wg.Wait()

		// Cache should still be functional
		cache.Set("/final", newTestContent("final"))

		result := cache.Get("/final")
		if result == nil {
			t.Error("Cache not functional after concurrent operations")
		}
	})

	t.Run("concurrent GetOrCompute operations", func(t *testing.T) {
		t.Parallel()

		cache := NewHTMLCache(100)

		var (
			wg           sync.WaitGroup
			computeCount int
			mu           sync.Mutex
		)

		for range 10 {
			wg.Go(func() {
				_, _ = cache.GetOrCompute(
					context.Background(),
					"/shared",
					func() (domain.RenderedContent, error) {
						mu.Lock()
						computeCount++
						mu.Unlock()

						return newTestContent("shared"), nil
					},
				)
			})
		}

		wg.Wait()

		// All goroutines should have gotten a result
		result := cache.Get("/shared")
		if result == nil {
			t.Error("Expected cached result after concurrent GetOrCompute")
		}
	})
}

// assertError is a simple error type for testing.
type assertError string

func (e assertError) Error() string {
	return string(e)
}
