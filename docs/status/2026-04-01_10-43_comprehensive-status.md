# Dynamic Markdown Site — Comprehensive Status Report

**Date:** 2026-04-01 10:43 | **Branch:** master | **Head:** fd89f13 | **Ahead of origin:** 4 commits

---

## Executive Summary

The project is in **strong shape**. Over the past 5 days (since March 28), 80+ commits have transformed it from a basic markdown server into a feature-complete, production-grade web application. All 7 testable packages pass, coverage is respectable, CI is healthy, and the Docker image is reproducible with SBOM/provenance attestations.

This session added: site name configuration, OpenGraph/Twitter meta tags, access logging middleware, draft frontmatter filtering (filesystem + blob repos), robots.txt serving, and config refactoring for linter compliance.

**Remaining concerns:** duplicate request logging (`requestLogger` + `accessLogMiddleware`), `container` package at 0% coverage, `domain` types undertested, and disk space critically low at 2.1GB free.

---

## A) FULLY DONE ✅

| Feature | Details | Commits |
|---------|---------|---------|
| **Core server** | Gin-based HTTP server with graceful shutdown, health check, rate limiting | Multiple |
| **Markdown rendering** | Goldmark + Chroma syntax highlighting, frontmatter parsing, TOC generation | Multiple |
| **Content repositories** | FileSystem + Blob (S3, GCS, Azure, file://, mem://) via gocloud.dev | `b5533f5` |
| **Search** | Full-text search with relevance scoring, snippet extraction, highlighting | Multiple |
| **Docker/OCI** | Multi-stage distroless build, cross-platform (amd64+arm64), SBOM, provenance | `20b275f` |
| **CI pipeline** | Test+lint+build+push on master, Trivy security scanning, smoke test | `5a1d057` |
| **Templates** | Templ-based type-safe HTML, directory/file/search/error views | Multiple |
| **Custom 404** | Smart path suggestions with case-insensitive matching, score threshold | `a1af8c2` |
| **Live reload** | SSE-based dev mode with filesystem watcher | `d2308ea` |
| **Caching** | Otter-based HTML response cache with invalidation | Multiple |
| **Site name config** | `SiteName` field via `-site-name` flag / `DYNAMIC_MARKDOWN_SITE_NAME` env, threaded through Config → Container → Server → LayoutProps → templates | `471fe10` → `9878b99` |
| **OpenGraph/Twitter meta** | `og:title`, `og:description`, `og:type`, `og:site_name`, `twitter:card`, `twitter:title`, `twitter:description` in layout template | `9878b99` |
| **Access logging** | Structured HTTP request logging (method, path, status, duration, request_id, client_ip) via `accessLogMiddleware` | `9878b99` |
| **Draft filtering** | `isDraft()` function parses YAML frontmatter for `draft: true`, wired into both FileSystemRepository and BlobRepository tree building | `3336d44`, `0d67f63` |
| **robots.txt** | `GET /robots.txt` with `User-agent: * / Allow: /` and dynamic sitemap URL | `748d17a` |
| **Config tests** | 3 new tests: `TestDefaultSiteName`, `TestConfigStringIncludesSiteName`, `TestConfigLogValueIncludesSiteName` | `9878b99` |
| **Draft tests** | 8 unit tests for `isDraft()` + 2 integration tests (FileSystem + Blob repo draft exclusion) | `0d67f63` |
| **Diagram support** | D2 server-side SVG rendering, Mermaid.js client-side rendering, custom AST transformer | `b42fe30` → `6c0423c` |
| **CSS** | 952-line comprehensive stylesheet with dark theme, responsive breakpoints, custom properties, accessibility | Multiple |
| **Config refactor** | Decomposed `Load()` into focused methods to satisfy cyclop complexity linter | `fd89f13` |
| **Diagram AST fix** | Initialize `BaseBlock` in `diagramNode` to satisfy `ast.Node` interface | `6c0423c` |
| **Dead code removal** | Removed `pkg/errors/errors.go` (was unused after migration to cockroachdb/errors) | `21c9ecf` |

---

## B) PARTIALLY DONE 🔧

| Item | Status | What's Left |
|------|--------|-------------|
| **robots.txt** | Handler works, route registered, middleware bypass added | No dedicated test file (`robots_test.go`). No `sitemap.xml` generation (robots.txt references it but it doesn't exist). No `SitemapURL` config. |
| **Draft filtering** | Both repos filter, tests pass | `isDraft()` uses simple string matching — doesn't handle YAML quoted values (`draft: "true"`), comments after value (`draft: true # comment`), or YAML flow syntax. Could miss edge cases. Linter hint: `strings.SplitSeq` is more efficient than `strings.Split`. |
| **Test coverage** | 7/7 packages passing, overall decent | `container` at **0%**, `domain` at **75.8%**, `content` at **77.5%**, `server` at **75.9%**, `renderer` at **80.5%**, `config` at **90.8%**, `cache` at **100%** |
| **CSS** | Comprehensive with dark theme, responsive, accessibility | Missing `@media print` styles. No `prefers-color-scheme` query. Some hardcoded pixel values that could use `clamp()`. No smooth scroll behavior. |
| **OpenGraph meta** | Basic tags present | Missing `og:image`, `og:url`, `og:description` from frontmatter. No configurable site URL for absolute canonical URLs. |
| **Dockerfile** | Production-quality multi-stage build | No `HEALTHCHECK` instruction (commented that distroless lacks shell). `COPY . .` copies everything including `docs/status/` (not in `.dockerignore`). |

---

## C) NOT STARTED 📋

| Item | Priority | Notes |
|------|----------|-------|
| **Sitemap.xml generation** | High | robots.txt references `/sitemap.xml` but endpoint doesn't exist. Search engines need this. |
| **Container package tests** | High | 0% coverage. `container.go` wires the entire DI graph — critical path with no verification. |
| **Domain unit tests** | Medium | `directory.go`, `file.go`, `tree.go`, `urlpath.go` have no dedicated test files. Only `types_test.go` exists. Core domain types deserve direct testing. |
| **LiveReload tests** | Medium | `livereload.go` has exported `LiveReload`, `Notify()`, `RegisterHandler()` — no direct tests. |
| **Helpers tests** | Medium | `helpers.go`: `shouldSkipDir`, `filterEmptyDirectories`, `isMarkdownFile` — no direct tests (indirectly covered via filesystem tests). |
| **Security headers tests** | Medium | `security.go` middleware — no direct unit tests (indirectly covered via handlers). |
| **`@media print` CSS** | Low | Print-friendly layout for articles, code blocks, diagrams. |
| **Configurable base URL** | Low | Needed for absolute OG URLs, sitemap, canonical links. Would require a `-base-url` flag. |
| **README update** | Low | Missing `-site-name` flag documentation. Architecture diagram references `pkg/` but `pkg/` was deleted. |
| **RSS/Atom feed** | Low | Nice-to-have for content discoverability. |
| **E2E tests** | Low | No end-to-end integration tests (start server, hit endpoints, verify HTML output). |
| **OpenTelemetry/metrics** | Low | No distributed tracing or Prometheus metrics. |
| **Authentication** | Low | No auth for refresh endpoint (just rate limiting). |
| **Content sorting options** | Low | No configurable sort order for directory listings. |

---

## D) TOTALLY FUCKED UP 💥

| Issue | Severity | Details |
|-------|----------|---------|
| **Duplicate request logging** | 🔴 HIGH | `requestLogger()` in `main.go:144` AND `accessLogMiddleware()` in `handlers.go:62` are BOTH registered as middleware. Every request is logged TWICE — once by `requestLogger` (in `main.go`), once by `accessLogMiddleware` (in `RegisterRoutes`). The `accessLogMiddleware` is also flagged as "unused" by the linter because golangci-lint can't see it's called via method value `s.accessLogMiddleware()`. This is a real bug: double log lines in production. **Fix: Remove `requestLogger` from main.go, keep `accessLogMiddleware` (it's better — includes request_id).** |
| **Disk space critical** | 🔴 HIGH | 2.1GB free on 229GB disk (99% full). Go build caches, Docker images, or test runs could fill it. Previously hit 100% during this session. |
| **Container package 0% coverage** | 🟡 MEDIUM | The DI container that wires everything together has zero tests. If a provider is added incorrectly, there's no automated verification. `container_test.go` exists but reports 0% because DI wiring uses `do.MustInvoke` which panics on failure, and the test file may not cover the actual `New()` function properly. |
| **`diagram_extension.go` unstaged change** | 🟡 MEDIUM | There's an unstaged change to `internal/renderer/diagram_extension.go` (adding `BaseBlock: ast.BaseBlock{}` to `diagramNode` initialization). This is a REAL fix for a bug where diagrams would panic at runtime. It should have been committed but the earlier `6c0423c` commit may not have included it fully. |
| **Linter warnings (5 active)** | 🟡 MEDIUM | `cyclop` on `applyEnvironmentOverrides` (complexity 11, max 10) — supposedly fixed in `fd89f13` but still showing. `testifylint` on draft_test.go lines 85/113 (wants `require` for error assertions). `unused` on `accessLogMiddleware` (false positive). `revive` on `version` package name. |

---

## E) WHAT WE SHOULD IMPROVE 🎯

### Critical (do next)

1. **Fix duplicate request logging** — Remove `requestLogger` from `main.go`. Keep `accessLogMiddleware` in the Server (better: includes request_id, already in middleware chain). This eliminates double-logging and the linter "unused" warning.

2. **Commit the unstaged `diagram_extension.go` change** — The `BaseBlock` initialization fix is sitting unstaged. If someone runs `git checkout`, it's lost and diagrams break.

3. **Add container package tests** — Test the DI wiring with mock implementations. At minimum, verify `New()` returns a valid container without panicking.

4. **Implement sitemap.xml** — robots.txt promises it. Search engines expect it. Generate from the content tree on demand.

### Important (do soon)

5. **Add domain unit tests** — `urlpath.go`, `directory.go`, `file.go`, `tree.go` are core types with no dedicated tests. They're indirectly tested but deserve direct coverage for edge cases.

6. **Fix linter warnings** — Resolve the 5 active warnings. The `cyclop` one should already be fixed by `fd89f13` — verify. Address `testifylint` and `revive` warnings.

7. **Add robots.txt tests** — Verify it returns correct content-type, cache headers, and valid robots.txt format.

8. **Clean up disk space** — Clear Go build cache, old Docker images. 2.1GB is too tight.

### Nice to have (backlog)

9. **Print CSS** — Articles deserve print-friendly layout.

10. **Configurable base URL** — Needed for absolute OG URLs and sitemap.

11. **README update** — Document `-site-name`, remove `pkg/` references.

12. **`@media (prefers-color-scheme)` CSS** — System dark/light mode preference.

13. **Content sorting** — Configurable sort for directory listings.

14. **OpenGraph images** — `og:image` from frontmatter or default.

15. **RSS/Atom feed** — Content syndication.

---

## F) Top #25 Things We Should Get Done Next

| # | Item | Effort | Impact | Type |
|---|------|--------|--------|------|
| 1 | Fix duplicate request logging (remove `requestLogger` from main.go) | 15min | 🔴 High | Bug |
| 2 | Commit unstaged `diagram_extension.go` BaseBlock fix | 2min | 🔴 High | Bug |
| 3 | Implement `/sitemap.xml` endpoint (generate from content tree) | 2h | 🔴 High | Feature |
| 4 | Add container package tests (0% → reasonable coverage) | 1h | 🔴 High | Testing |
| 5 | Add domain unit tests (`urlpath_test.go`, `directory_test.go`, `file_test.go`, `tree_test.go`) | 3h | 🟡 Medium | Testing |
| 6 | Fix all 5 active linter warnings | 30min | 🟡 Medium | Quality |
| 7 | Add robots.txt handler tests | 30min | 🟡 Medium | Testing |
| 8 | Free disk space (clear caches, prune Docker) | 15min | 🔴 High | Ops |
| 9 | Add `@media print` CSS for articles and code blocks | 1h | 🟢 Low | UX |
| 10 | Update README (document `-site-name`, remove `pkg/` references) | 30min | 🟢 Low | Docs |
| 11 | Add LiveReload unit tests | 1h | 🟡 Medium | Testing |
| 12 | Add helpers_test.go (shouldSkipDir, filterEmptyDirectories, isMarkdownFile) | 30min | 🟡 Medium | Testing |
| 13 | Add security headers middleware tests | 30min | 🟡 Medium | Testing |
| 14 | Add configurable `-base-url` flag for absolute URLs | 1h | 🟢 Low | Feature |
| 15 | Use `strings.SplitSeq` in `isDraft()` (Go 1.26 optimization) | 5min | 🟢 Low | Perf |
| 16 | Improve draft detection (handle quoted YAML, comments) | 30min | 🟢 Low | Feature |
| 17 | Add `@media (prefers-color-scheme)` CSS | 30min | 🟢 Low | UX |
| 18 | Add `og:image`, `og:url` to OpenGraph meta | 1h | 🟢 Low | SEO |
| 19 | Add E2E integration tests (start server, hit endpoints, verify HTML) | 2h | 🟡 Medium | Testing |
| 20 | Add `.dockerignore` entries for `docs/status/` | 5min | 🟢 Low | Build |
| 21 | Add RSS/Atom feed generation | 2h | 🟢 Low | Feature |
| 22 | Add configurable directory sorting | 1h | 🟢 Low | Feature |
| 23 | Add OpenTelemetry tracing / Prometheus metrics | 3h | 🟢 Low | Observability |
| 24 | Clean up `docs/status/` — 34 status reports, most are stale | 30min | 🟢 Low | Housekeeping |
| 25 | Add version flag test | 15min | 🟢 Low | Testing |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Is the `accessLogMiddleware` in `handlers.go:62` actually redundant with `requestLogger` in `main.go:144`?**

Both are registered as middleware on the same Gin engine:
- `main.go:144`: `router.Use(requestLogger(svc.logger))` — logs method, path, status, duration, client_ip, errors
- `handlers.go:62`: `router.Use(s.accessLogMiddleware())` — logs method, path, status, duration, request_id, client_ip

I **believe** `requestLogger` should be removed (it lacks request_id and is a duplicate), but I'm not 100% sure if there's a deliberate reason both exist. The `accessLogMiddleware` is clearly better (includes request_id). Confirming this would let me safely remove the duplicate.

---

## Test Coverage Summary

| Package | Coverage | Status |
|---------|----------|--------|
| `internal/cache` | **100.0%** | ✅ Perfect |
| `internal/config` | **90.8%** | ✅ Excellent |
| `internal/renderer` | **80.5%** | ✅ Good |
| `internal/server` | **75.9%** | 🟡 Adequate |
| `internal/domain` | **75.8%** | 🟡 Adequate |
| `internal/content` | **77.5%** | 🟡 Adequate |
| `internal/container` | **0.0%** | 🔴 Critical |
| **Production Go** | **4,526 lines** | |
| **Test Go** | **6,439 lines** | 1.42:1 test:code ratio |
| **Generated (templ)** | **1,351 lines** | |

## Build & CI Health

| Check | Status |
|-------|--------|
| `go build ./...` | ✅ Clean |
| `go test ./... -count=1` | ✅ All 7 packages pass |
| `go vet ./...` | ✅ Clean |
| `templ generate` | ✅ Up to date (0 updates) |
| Docker build (CI) | ✅ Multi-platform (amd64+arm64) |
| Trivy security scan | ✅ Configured |
| Disk space | 🔴 2.1GB free (99% full) |

## Session Activity (last 10 commits)

```
fd89f13 refactor(config): decompose Load() into focused methods to fix cyclop
748d17a feat(server): add robots.txt endpoint with sitemap reference
0d67f63 feat(content): apply draft filtering to blob repository and add tests
945b194 chore(lint): add exhaustruct exclusion for diagram_extension.go
3336d44 feat(content): add isDraft helper for frontmatter detection
9878b99 feat: add site-name config, draft filtering, access logging, and Open Graph support
21c9ecf chore: improve docker workflow, refactor config, fix diagram extension, remove dead errors pkg
6c0423c fix(renderer): initialize BaseBlock field in diagramNode to satisfy ast.Node interface
2ba1c02 ci: remove redundant container smoke test from docker workflow
d26c05d ci: remove redundant container smoke test from docker workflow
```

## Files Changed This Session (16 files, +521 / -158 lines)

```
 .github/workflows/docker.yml                       |  17 +--
 .golangci.yml                                      |   3 +
 docs/status/2026-04-01_09-58_mermaid-diagram-fix-complete.md | 162 ++++++
 internal/config/config.go                          |  87 +++--
 internal/config/config_test.go                     |  48 +++
 internal/content/blob.go                           |   4 +
 internal/content/draft.go                          |  27 ++
 internal/content/draft_test.go                     | 127 ++++++
 internal/content/filesystem.go                     |   4 +
 internal/renderer/diagram_extension.go             |  14 +-
 internal/renderer/diagrams.go                      |  24 +-
 internal/renderer/diagrams_test.go                 |  75 +++--
 internal/server/accesslog.go                       |  28 ++
 internal/server/handlers.go                        |   6 +-
 internal/server/robots.go                          |  22 ++
 templates/layout.templ                             |   7 +
```

---

_Report generated at 2026-04-01 10:43 by automated audit._
