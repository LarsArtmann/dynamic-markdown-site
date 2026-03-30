// Package testutil provides shared test fixtures and utilities.
package testutil

import (
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// CacheFixture provides test fixtures for cache testing.
type CacheFixture struct {
	Cache *cache.HTMLCache
}

// NewCacheFixture creates a new cache fixture with the specified max entries.
func NewCacheFixture(maxEntries int) *CacheFixture {
	return &CacheFixture{
		Cache: cache.NewHTMLCache(maxEntries),
	}
}

// MustGet retrieves a value from cache or fails the test if not found.
func (f *CacheFixture) MustGet(t *testing.T, path string) *domain.RenderedContent {
	t.Helper()

	result := f.Cache.Get(path)
	if result == nil {
		t.Fatalf("Expected cached value for path %q, got nil", path)
	}

	return result
}

// AssertSize verifies the cache has the expected size.
func (f *CacheFixture) AssertSize(t *testing.T, want int) {
	t.Helper()

	if got := f.Cache.EstimatedSize(); got != want {
		t.Errorf("Expected cache size %d, got %d", want, got)
	}
}

// AssertEmpty verifies the cache is empty.
func (f *CacheFixture) AssertEmpty(t *testing.T) {
	t.Helper()
	f.AssertSize(t, 0)
}

// AssertMissing verifies no value exists at the given path.
func (f *CacheFixture) AssertMissing(t *testing.T, path string) {
	t.Helper()

	if result := f.Cache.Get(path); result != nil {
		t.Errorf("Expected no cached value for path %q, got %+v", path, result)
	}
}
