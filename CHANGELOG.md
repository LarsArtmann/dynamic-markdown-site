# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

> **Tag metadata note (2026-09-27):** the existing tags `v0.1.0`, `v0.2.0`, and `v0.3.0` are all annotated tags pointing at the same commit (`67f7632`, 2026-05-17), so their names do not correspond to the CHANGELOG entries below — `v0.2.0`/`v0.3.0` carry no distinct release state. They are left in place rather than rewritten (published refs are history); the next properly cut tag supersedes them.

### Added

- Request timing middleware — `X-Response-Time` header on every response (`internal/server/responsetime.go`)
- Dependabot config with weekly grouped minor/patch updates (Go modules, Actions, pnpm)
- dprint formatter config for JSON/YAML/Config files
- Astro + Starlight documentation website in `website/` (hosted at dynamicmarkdown.lars.software)
- Admonition/Alert blocks — GitHub-style `> [!TYPE]` blockquote syntax with 6 types (NOTE, TIP, IMPORTANT, WARNING, CAUTION, CRITICAL) and themed CSS styling
- Custom Goldmark AST transformer for parsing alert markers across split text nodes
- `/sitemap.xml` endpoint for search engine crawlers with priority and changefreq metadata
- `/metrics` endpoint exposing Prometheus-format metrics (dependency-free)
- `/cache/stats` endpoint returning cache hit/miss/eviction statistics (JSON)
- Raw asset serving for non-markdown files (images, PDFs, JSON, etc.) alongside markdown content
- URL fallback handling: `.md` extension redirects, case-insensitive path matching, trailing slash normalization
- AST-based Mermaid detection via Goldmark parser context (replaces regex-based approach)
- Comprehensive sitemap tests covering directories, files, HTTPS detection, and priority calculation
- HEALTHCHECK directive re-added to Dockerfile (uses binary's `healthcheck` subcommand)
- Compression middleware via `httputil.Compression` for gzip-encoded responses

### Changed

- **Migrated server JSON to `encoding/json/v2`** — requires `GOEXPERIMENT=jsonv2` and a `go 1.27+` toolchain (`go.mod` now declares `go 1.27.1`); supersedes the earlier "stable encoding/json only" policy
- **Dependency injection hardened** — all `do.MustInvoke` call sites replaced with error-returning `do.Invoke`; container accessors now return `(T, error)` (Cache/Renderer/Searcher accessors removed as dead code)
- **File watching rewritten on `go-filewatcher/v2`** — 181 lines of hand-rolled fsnotify boilerplate replaced with 85 lines; watcher now shuts down cleanly via the SIGINT/SIGTERM context
- Upgraded to `charm.land/log/v2`, `otter/v2`, `httputil` v1.2.0, `go-filewatcher/v2` v2.3.0
- Website flake apps switched from npm to pnpm
- Refactored frontmatter draft parsing to use `yaml.v3` for proper boolean handling
- Simplified static file embedding pattern using `//go:embed`
- Refactored `getContentType` from switch statement to map lookup
- Improved sitemap test quality: `NewRequestWithContext`, extracted test host constant, `InEpsilon` for float comparisons
- Added godoc comments on exported admonition extension types
- Silence `fmt.Fprintf` return value warnings in admonition renderer
- Add linter exclusions for exhaustruct and gochecknoglobals in Goldmark extensions

### Fixed

- Flaky `TestRateLimiter_Concurrent` — exact-count assertions now use the `newBurstOnlyLimiter` helper (1-hour window, negligible refill); sibling tests hardened the same way
- `TestRefreshRateLimit` no-op assertion — now asserts exactly 10× `200` + 5× `429` across 15 sequential requests
- Pre-existing build break from `httputil` v0.6.0 importing `encoding/json/v2` (resolved by the json/v2 adoption)
- `TestHealthEndpoint` brittleness under ldflags injection — reads `version.Version`/`version.Commit` at runtime instead of hardcoding `dev`/`unknown`
- `nix-build` `vendorHash` ↔ `go.sum` drift (multiple fixes; latest `b4d12bb`)
- golangci-lint findings: `makezero` in Levenshtein DP, `unparam` in sitemap test helper, misplaced `SkipDirs` nolint directive
- Website: HTML cache rule now matches cleanUrls pages (`f55589a`)
- Fixed panic on double `Stop()` call in rate limiter
- Fixed `hasMermaid` not propagating through `NewRenderedFile` constructor
- Removed dead regex-based diagram detection code
- Stripped `.md` extension from URL paths for clean URLs
- Reverted accidental `encoding/json/v2` migration that broke compilation (`GOEXPERIMENT=jsonv2` not enabled; superseded by the json/v2 adoption above)
- Fixed metrics endpoint test by setting `Accept-Encoding: identity` to bypass compression middleware

### Security

- Pinned `anchore/sbom-action/download-syft` to a full commit SHA in `release.yml`
- Skipped root-level `pnpm audit` in BuildFlow (wrong directory — see AGENTS.md gotcha #15); audit `website/` manually via `cd website && pnpm audit`

### Removed

- Removed Gin web framework — migrated to standard `net/http` with Go 1.22+ method-based routing
- Removed `justfile` — build/task automation now handled by `flake.nix`

## [0.1.0] - 2026-04-01

### Added

- Go web server that converts markdown files into a navigable website
- Gin HTTP framework for routing with middleware support
- Goldmark + Chroma markdown rendering with syntax highlighting
- Templ type-safe HTML templates for directory views, file views, and search
- Browser live reload via Server-Sent Events (SSE) in dev mode
- D2 diagram support with server-side SVG rendering
- Mermaid diagram support with client-side rendering
- Diagram CSS styling for embedded SVG and Mermaid output
- Custom AST node types and transformers for the diagram rendering pipeline
- Custom 404 page with Levenshtein distance path suggestions and case-insensitive matching
- Request ID middleware for distributed request tracing
- Blob storage support via go-cloud (S3, GCS, Azure Blob, file, memory backends)
- Site name configuration via `-site-name` flag and `DYNAMIC_MARKDOWN_SITE_NAME` env var
- Draft filtering from YAML frontmatter (`draft: true` hides content)
- Robots.txt endpoint with sitemap reference
- Security headers middleware (X-Content-Type-Options, X-Frame-Options, CSP, HSTS)
- Open Graph meta tag support in templates
- Access logging middleware with structured request logging
- HTML response caching via otter cache (10K entries, 1h TTL)
- Content search across all markdown files
- Rate limiting on `/refresh` endpoint (10 req/min per IP)
- File system watcher in dev mode (auto-refresh on markdown changes)
- Graceful shutdown handling (SIGINT/SIGTERM, 30s drain timeout)
- YAML frontmatter parsing (title, description, author, tags, draft)
- Health check endpoint (`/health`)
- RenderedContent immutable type for the render pipeline
- `RenderedContent` type in domain package for type-safe HTML passing
- Centralized testutil package for shared HTTP test fixtures
- `DefaultConfig()` function for consistent test configuration
- Renderer benchmarks for performance tracking
- Request timeout configuration (`-timeout` flag)
- Containerized multi-arch Docker builds (linux/amd64, linux/arm64)
- GitHub Actions CI with Docker build, lint, test, and smoke-test pipeline
- Dedicated linter CI job to unblock Docker builds
- SBOM generation and Trivy security scanning in CI pipeline
- GHCR (GitHub Container Registry) push on master/tag
- CI triggers on master push, version tags, and pull requests
- `.editorconfig` for consistent formatting across contributors
- `justfile` with test, lint, fix, pre-push, gen-build, and cover recipes
- Binary version injection via ldflags (version, commit, build date)
- Configuration validation with descriptive error messages

### Changed

- Migrated static asset serving from disk to embedded filesystem
- Upgraded Docker base image to distroless/static-debian13
- Decomposed `config.Load()` into focused methods (`defineAndParseFlags`, `applyEnvironmentOverrides`, `applyDerivedSettings`) to reduce cyclomatic complexity
- Replaced `html/template` dependency with `domain.HTML` type alias
- Extracted `renderComponent` method for unified component rendering
- Migrated error recording to structured `stats.recordError` in filesystem repository
- Removed deprecated `FileNode` setters, completing immutable render pipeline
- Footer now links "Lars Artmann" to personal website and brand to GitHub repo
- Normalized JSON and CSS indentation across project files
- Reduced blob repository initialization timeout from 30s to 10s

### Fixed

- Fixed panic in config loading — changed to proper error return
- Fixed `exhaustruct` linter error by initializing `BaseBlock` in diagram AST node
- Fixed `noctx` errors by replacing `httptest.NewRequest` with `NewRequestWithContext`
- Fixed Dockerfile build failures (dynamic ARG commands, numeric UID for nonroot, missing COPY)
- Fixed duplicate request logger middleware
- Fixed undefined `RenderedContent` type references in cache tests
- Fixed blob tree building by moving filter earlier and removing dead code
- Fixed CI build and lint failures across multiple iterations
- Fixed long lines for golines compliance across test and server files
- Fixed missing context import in request ID tests

### Removed

- Removed unused `pkg/errors` package (conflicted with stdlib `errors`)
- Removed 33 stale status reports from `docs/status/`
- Removed duplicate container smoke test from CI workflow
- Removed broken COPY directive for non-existent `internal/static/`
- Removed dynamic git/date commands from Dockerfile ARG builds (incompatible with Docker build args)
- Removed HEALTHCHECK from Dockerfile (incompatible with distroless)

### Security

- Added security headers middleware (X-Content-Type-Options, X-Frame-Options, CSP, HSTS)
- Added URLPath validation preventing directory traversal attacks
- Added SBOM and Trivy vulnerability scanning in CI pipeline
- Added rate limiting on refresh endpoint
- Bumped `golang.org/x/crypto` dependency
- Supply chain security improvements for Docker image builds
