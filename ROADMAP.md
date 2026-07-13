# Roadmap

**Generated:** 2026-04-05 | **Last Updated:** 2026-07-13
**Purpose:** Aspirational items without timeline

> Items here are goals and ideas. No commitment on delivery dates.

## 🚀 Features

### Search & Discovery

- [ ] Implement search autocomplete
- [ ] Implement search indexing for better search relevance
- [ ] Add full-text search with Bleve/Meilisearch
- [ ] Implement related content suggestions
- [ ] Implement content tags and filtering

### Performance & Caching

- [ ] Implement cache warming strategy
- [ ] Create cache statistics dashboard
- [ ] Add distributed cache with Redis
- [ ] Benchmark regression tracking in CI
- [ ] Create benchmark suite
- [ ] Benchmark with 1,000+ files

### Rendering & Content

- [ ] Implement content draft preview
- [ ] Add image optimization
- [ ] Implement graceful degradation for D2 rendering failures — _partially done; full fallback chain could be deeper_
- [ ] Add diagram export (PNG/SVG download buttons)
- [ ] Add diagram zoom for large diagrams
- [ ] Implement content versioning (git-based history)
- [ ] Implement live preview (WYSIWYG markdown editing)

### UI/UX

- [ ] Dark mode CSS and theme toggle
- [ ] Implement syntax highlighting themes
- [ ] Add keyboard navigation and shortcuts
- [ ] Implement table of contents with sticky positioning — _base TOC shipped; sticky UX not done_
- [ ] Add print stylesheet
- [ ] Add code copy button (one-click code copying)
- [ ] Add pagination for directories

### Content Delivery

- [ ] Add RSS/Atom feed generation
- [ ] Implement WebSocket live reload
- [ ] Add gzip/brotli compression
- [ ] Add ETag/If-None-Match support

### Observability

- [ ] Add structured logging to renderer package
- [ ] Add response time histograms
- [ ] Implement distributed tracing with OpenTelemetry
- [ ] Add pprof profiling endpoint
- [ ] Add request/response logging with correlation IDs
- [ ] Implement rate limiting per-endpoint configuration

### Admin & API

- [ ] Add admin/debug endpoints (content stats dashboard)
- [ ] Create admin dashboard
- [ ] Add API endpoint for programmatic content access
- [ ] Add analytics integration

### Architecture

- [ ] Implement plugin system
- [ ] Design plugin system for custom markdown extensions
- [ ] Split Repository interface (Reader/Refresher)
- [ ] Structured error types (Is/As/Unwrap)

### Internationalization

- [ ] Implement internationalization (multi-language support)

## 🚢 Deployment

- [ ] Add Kubernetes manifests
- [ ] CDN/edge deployment manifests (Cloud Run/Fly.io)
- [ ] Create deployment documentation

## 🔬 Quality

- [ ] Add mutation testing
- [ ] Add content preview functionality
- [ ] Add sample markdown content to content/ directory

## Resources

- See [TODO_LIST.md](./TODO_LIST.md) for actionable items
- See [CHANGELOG.md](./CHANGELOG.md) for completed items
