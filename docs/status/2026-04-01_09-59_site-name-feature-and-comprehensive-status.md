# Dynamic Markdown Site — Comprehensive Status Report

**Date:** 2026-04-01 09:59 UTC  
**Branch:** master  
**Working Tree:** CLEAN (zero uncommitted changes)  
**Test Suite:** ALL 7 PACKAGES PASSING  
**Codebase:** ~10,759 lines of Go (excluding generated `_templ.go`)

---

## A) FULLY DONE ✅

### Core Application
- **Go web server** — Gin-based HTTP routing with graceful shutdown
- **Markdown rendering** — Goldmark + Chroma syntax highlighting, YAML frontmatter
- **Templ templates** — Type-safe HTML with `LayoutProps`, `DirectoryView`, `FileView`, `SearchView`, `ErrorView`
- **Dependency injection** — `samber/do/v2` container wiring all services
- **Caching** — Otter-based HTML response cache with invalidation
- **File watching** — Dev mode live reload via SSE (`-dev` flag)
- **Content search** — Full-text search across all markdown files
- **Custom 404** — Fuzzy path suggestions with similarity scoring
- **Breadcrumb navigation** — Built from URL path hierarchy
- **Table of contents** — Auto-generated from markdown headings
- **Reading time** — Estimated reading time per file
- **Mermaid diagrams** — Client-side rendering via Mermaid.js CDN
- **D2 diagram support** — Custom AST node types and transformer pipeline for D2 diagrams
- **Rate limiting** — 10 req/min per IP on `/refresh` endpoint
- **Security headers** — Request ID middleware, security headers middleware

### Configuration
- **CLI flags** — `-port`, `-root`, `-log-level`, `-cache`, `-dev`, `-timeout`, `-storage-url`
- **Environment variables** — `DYNAMIC_MARKDOWN_*` prefix for all config
- **Site name** — `DYNAMIC_MARKDOWN_SITE_NAME` (env) / defaults to `"Site"` — threads through `Config → Container → Server → LayoutProps → Header template` and all error views

### Storage
- **Filesystem repository** — Reads markdown from disk, ignores hidden files
- **Blob storage** — OCI-compatible via `gocloud.dev` (S3, GCS, file://, etc.) with timeout context
- **In-memory repository** — For testing

### CI/CD
- **Docker build** — Multi-stage Dockerfile, OCI-compliant, nonroot user, arm64/amd64
- **GitHub Actions** — `docker.yml` workflow for push/tags, pin-locked `templ` version
- **golangci-lint** — ~75 linters configured in `.golangci.yml`

### Testing
- **7 test packages** — All passing: `cache`, `config`, `container`, `content`, `domain`, `renderer`, `server`
- **Parallel tests** — `t.Parallel()` enforced by `paralleltest` linter
- **Benchmarks** — Rendering and server benchmarks
- **Test utilities** — `internal/testutil/http.go` shared fixtures
- **Mock repositories** — `FailingRepository` for error path testing

### Documentation
- **30 status reports** in `docs/status/` spanning Mar 28 – Apr 1
- **Project AGENTS.md** — Comprehensive agent guidelines with commands, patterns, gotchas
- **.editorconfig** — Consistent editor settings

---

## B) PARTIALLY DONE 🟡

### Blob Storage
- **Status:** Wired and functional, but limited real-world testing
- **What works:** S3, GCS, file:// scheme support; timeout context; tree building; filter logic
- **What's missing:** Integration tests against real cloud storage; ETag/If-Modified-Since caching; streaming large files

### D2 Diagram Rendering
- **Status:** Custom AST node types and transformer pipeline committed (`diagram_extension.go`)
- **What works:** AST node types, transformer registration
- **What's unclear:** Whether server-side D2 rendering is fully end-to-end working or falls back to client-side only

### CSS/Styling
- **Status:** Functional but potentially needs polish
- **What works:** Site theme, cards, breadcrumbs, search, error pages, TOC sidebar, code highlighting
- **What's missing:** Dark mode toggle, mobile responsiveness audit, design polish pass

---

## C) NOT STARTED ⬜

1. **Metrics/Observability** — No Prometheus metrics, OpenTelemetry traces, or structured metrics export
2. **Authentication/Authorization** — No auth layer; all endpoints are public
3. **HTTPS/TLS** — Server is HTTP-only; no TLS termination
4. **Pagination** — Directory listings have no pagination for large directories
5. **RSS/Atom feed** — No feed generation for content
6. **Sitemap.xml** — No sitemap generation for SEO
7. **robots.txt** — No robots.txt serving
8. **Multi-theme support** — No theme switching (only `site-theme`)
9. **i18n/Localization** — All UI strings are English-only, hardcoded
10. **API versioning** — `/health` and `/refresh` are unversioned
11. **Webhook support** — No webhook for content refresh notifications
12. **Access logging** — No structured HTTP access logs (request ID exists but no access log format)
13. **Content drafts** — Frontmatter has `draft` field but no filtering logic
14. **Image optimization** — No image processing, thumbnailing, or lazy loading
15. **Content ordering** — No configurable sort order for directory listings
16. **Plugin system** — No extensibility mechanism for custom renderers
17. **Multi-root support** — Single root directory only
18. **Configurable favicon/logo** — Favicon and logo icon (◈) are hardcoded
19. **OpenGraph/Twitter meta** — No social media meta tags beyond basic description
20. **Content permissions** — No per-path access control

---

## D) TOTALLY FUCKED UP 💥

1. **Go version mismatch** — `go.mod` says `go 1.26.1` but installed toolchain is `go1.26.0`. Every `go test` run spews ~30 `compile: version "go1.26.1" does not match go tool version "go1.26.0"` warnings. Tests still pass but it's noisy and fragile.
2. **Missing CI lint job** — The `docker.yml` workflow only builds Docker. There's no separate `lint` or `test` CI job. Linting only happens locally.
3. **Disk was 100% full** — Go build cache (2.8GB) + tool caches filled the 229GB disk to 131MB free. Cleared ~4GB during this session.

---

## E) WHAT WE SHOULD IMPROVE 🔧

1. **Fix Go version mismatch** — Either update `go.mod` to `go 1.26.0` or install Go 1.26.1 locally.
2. **Add CI test job** — GitHub Actions should run `go test ./...` before Docker build.
3. **Add CI lint job** — Run `golangci-lint` in CI. Currently only runs locally.
4. **Config tests for SiteName** — Add explicit test that `DYNAMIC_MARKDOWN_SITE_NAME` env var is read correctly.
5. **Blob storage integration tests** — At minimum, test against a localstack/minio container.
6. **Access logging middleware** — Add structured access logging with request ID, method, path, status, duration.
7. **Configurable favicon/logo icon** — The `◈` is hardcoded. Make it configurable.
8. **OpenGraph meta tags** — Add `og:title`, `og:description`, `og:type`, `twitter:card` to layout.
9. **Content draft filtering** — The `draft: true` frontmatter field exists but isn't checked.
10. **Reduce status report sprawl** — 30 status docs in 4 days. Consider a summary index.

---

## F) TOP 25 THINGS TO DO NEXT (Prioritized)

| # | Priority | Task | Effort |
|---|----------|------|--------|
| 1 | 🔴 Critical | Fix Go version mismatch (go.mod vs toolchain) | 5min |
| 2 | 🔴 Critical | Add CI `test` job to GitHub Actions | 30min |
| 3 | 🔴 Critical | Add CI `lint` job to GitHub Actions | 30min |
| 4 | 🟠 High | Add `DYNAMIC_MARKDOWN_SITE_NAME` config test | 15min |
| 5 | 🟠 High | Add access logging middleware | 1h |
| 6 | 🟠 High | Add OpenGraph/Twitter meta tags to layout | 30min |
| 7 | 🟠 High | Implement `draft: true` frontmatter filtering | 30min |
| 8 | 🟠 High | Add configurable favicon (env var or file-based) | 30min |
| 9 | 🟡 Medium | Add `sitemap.xml` generation | 1h |
| 10 | 🟡 Medium | Add `robots.txt` serving | 15min |
| 11 | 🟡 Medium | Add pagination for directory listings | 2h |
| 12 | 🟡 Medium | Add HTTPS/TLS support (auto or manual) | 2h |
| 13 | 🟡 Medium | Add RSS/Atom feed generation | 1h |
| 14 | 🟡 Medium | Add Prometheus metrics endpoint | 1h |
| 15 | 🟡 Medium | Add configurable logo icon (replace ◈) | 30min |
| 16 | 🟡 Medium | Dark mode toggle | 2h |
| 17 | 🟡 Medium | Mobile responsiveness audit | 2h |
| 18 | 🟡 Medium | Add blob storage integration tests (minio) | 2h |
| 19 | 🟢 Low | Add API versioning (`/api/v1/health`) | 1h |
| 20 | 🟢 Low | Add webhook endpoint for content refresh | 1h |
| 21 | 🟢 Low | Multi-root directory support | 3h |
| 22 | 🟢 Low | Image optimization (resize, lazy load) | 3h |
| 23 | 🟢 Low | i18n/localization framework | 3h |
| 24 | 🟢 Low | Content ordering (sort by name/date/size) | 1h |
| 25 | 🟢 Low | Plugin/extensibility system | 5h |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF ❓

**What is the deployment target?**

Is this meant to be:
- A **single-binary deploy** to a VPS/VM?
- A **containerized microservice** on Kubernetes?
- A **serverless/edge** function (Cloud Run, Fly.io, etc.)?
- A **desktop/local tool** for previewing markdown?

This fundamentally affects priorities like TLS (handled by load balancer vs. app), auth (needed if public-facing), observability (Prometheus vs. cloud-native metrics), and whether blob storage is the primary use case or a nice-to-have.

---

## Session Summary (2026-04-01 ~09:04–09:59)

### Completed this session
1. **Implemented `DYNAMIC_MARKDOWN_SITE_NAME`** — Full end-to-end wiring from env var → Config → Container → Server → LayoutProps → Header template → ErrorViewProps → all error pages
2. **Modified 8 source files + 3 test files** — config, handlers, container, render, errors, layout.templ, handlers_test, benchmark_test, testutil
3. **All 7 test packages pass** — Zero regressions
4. **Cleared ~4GB disk space** — Go build cache (2.8GB) + tool caches

### Files changed in last 5 commits (HEAD~5..HEAD)
| File | Changes |
|------|---------|
| `internal/renderer/diagram_extension.go` | +98/-54 (D2 AST refactor) |
| `internal/renderer/diagrams_test.go` | +138 (new D2 tests) |
| `internal/config/config.go` | +SiteName field, env var, defaults |
| `internal/server/handlers.go` | +siteName field, NewServer param |
| `internal/server/render.go` | +SiteName in all LayoutProps |
| `internal/server/errors.go` | +SiteName in error views |
| `internal/container/container.go` | +cfg.SiteName wiring |
| `templates/layout.templ` | +SiteName in LayoutProps, ErrorViewProps, Header |
| `internal/server/handlers_test.go` | +`"Site"` param in all NewServer calls |
| `internal/server/benchmark_test.go` | +`"Site"` param |
| `internal/testutil/http.go` | +`"Site"` param |
| `Dockerfile` | UID fix for arm64 |
