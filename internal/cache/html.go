// Package cache provides HTML rendering caching using otter.
package cache

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/maypok86/otter/v2"
	"github.com/maypok86/otter/v2/stats"
)

// HTMLCache provides caching for rendered HTML content.
type HTMLCache struct {
	cache *otter.Cache[string, domain.RenderedContent]
}

// NewHTMLCache creates a new HTML cache with the specified maximum size.
func NewHTMLCache(maxSize int) *HTMLCache {
	//nolint:exhaustruct
	cache := otter.Must(&otter.Options[string, domain.RenderedContent]{
		MaximumSize:      maxSize,
		ExpiryCalculator: otter.ExpiryAccessing[string, domain.RenderedContent](time.Hour),
		StatsRecorder:    stats.NewCounter(),
	})

	return &HTMLCache{cache: cache}
}

// Get retrieves cached content by path. Returns nil if not found.
func (c *HTMLCache) Get(path string) *domain.RenderedContent {
	val, ok := c.cache.GetIfPresent(path)
	if !ok {
		return nil
	}

	return &val
}

// Set stores rendered content at the given path.
func (c *HTMLCache) Set(path string, content domain.RenderedContent) {
	c.cache.Set(path, content)
}

// GetOrCompute returns cached content or computes it using the provided function.
func (c *HTMLCache) GetOrCompute(
	ctx context.Context,
	path string,
	compute func() (domain.RenderedContent, error),
) (*domain.RenderedContent, error) {
	val, err := c.cache.Get(
		ctx,
		path,
		otter.LoaderFunc[string, domain.RenderedContent](
			func(_ context.Context, _ string) (domain.RenderedContent, error) {
				return compute()
			},
		),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "cache get failed for path %s", path)
	}

	return &val, nil
}

// InvalidateAll removes all cached entries.
func (c *HTMLCache) InvalidateAll() {
	c.cache.InvalidateAll()
}

// Close releases all cache resources including background goroutines.
func (c *HTMLCache) Close() {
	c.cache.StopAllGoroutines()
}

// Stats returns cache statistics.
func (c *HTMLCache) Stats() Stats {
	s := c.cache.Stats()

	return Stats{
		Hits:          s.Hits,
		Misses:        s.Misses,
		Evictions:     s.Evictions,
		LoadSuccesses: s.LoadSuccesses,
		LoadFailures:  s.LoadFailures,
		TotalLoadTime: s.TotalLoadTime,
	}
}

// EstimatedSize returns the estimated number of entries in the cache.
func (c *HTMLCache) EstimatedSize() int {
	return c.cache.EstimatedSize()
}

// Stats holds cache statistics.
type Stats struct {
	Hits          uint64
	Misses        uint64
	Evictions     uint64
	LoadSuccesses uint64
	LoadFailures  uint64
	TotalLoadTime time.Duration
}

// HitRatio returns the cache hit ratio (0.0 to 1.0).
func (s Stats) HitRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}

	return float64(s.Hits) / float64(total)
}

// Requests returns total number of cache requests.
func (s Stats) Requests() uint64 {
	return s.Hits + s.Misses
}
