# 3. Use Otter for HTML caching

Date: 2026-06-14

## Status

Accepted.

## Context

The server renders markdown to HTML on every request unless the result
is cached. Rendering is dominated by Goldmark + Chroma and is the
single biggest contributor to per-request CPU usage. A TTL-based
in-memory cache avoids re-rendering for repeated requests and protects
the server from request spikes.

We evaluated three options:

1. **stdlib `map + sync.RWMutex`** — simplest possible, but no eviction
   and no concurrency-friendly API for our use case
2. **`hashicorp/golang-lru`** — popular, well-tested, but stores `any`
   and is no longer actively maintained
3. **`maypok86/otter/v2`** — modern, lock-free reads, supports TTL
   and size-based eviction, used in production by several large Go
   services

## Decision

We use **`maypok86/otter/v2`** for the HTML cache. The cache holds
**10,000 entries** with a **1-hour access expiry** (entry expires if
not read for 1 hour).

Key properties we rely on:

- Lock-free reads (fast path) under low contention
- Background goroutine for eviction — **must be stopped with
  `cache.Close()` on shutdown** to avoid goroutine leaks (covered by
  the `Shutdown()` method on the server and by `t.Cleanup` in tests)
- Per-entry TTL plus size-based eviction (LRU-K)
- `Stats()` accessor exposing hits, misses, evictions, and load
  durations — powers our `/cache/stats` and `/metrics` endpoints

## Consequences

Positive:

- Caches reduce p99 render latency by ~10x for repeat content
- Background eviction keeps memory bounded under high cardinality
- Stats surface makes cache effectiveness observable

Negative:

- One more external dependency (Otter has a small but non-zero
  transitive set)
- Background goroutine requires explicit `Close()` — easy to forget;
  we have a shutdown integration test that verifies in-flight
  requests complete

## Alternatives considered

- **bigcache / freecache** — byte-level caches; would require
  serializing `domain.RenderedContent` ourselves and gain little for
  this workload
- **ristretto** — also solid; Otter's `Stats()` API is more
  ergonomic for our needs
