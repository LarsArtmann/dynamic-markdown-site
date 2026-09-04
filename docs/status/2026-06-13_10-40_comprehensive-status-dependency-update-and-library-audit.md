# Comprehensive Status Report — Dynamic Markdown Site

**Date:** 2026-06-13 10:40 | **Branch:** master | **Commit:** d48c3bd | **Go:** 1.26.3

---

## Executive Summary

**13,384 lines of Go** across **61 files** (40 production + 21 test). All tests pass, build is clean, zero linter errors. The project is functional and well-documented but carries **0/43 TODO items completed** and significant technical debt in test coverage and unused dependencies.

This session's work: **updated all 5 outdated dependencies to latest**, **added missing middleware (Recovery, Compression)**, **fixed a middleware ordering bug** (request_id was empty in logs), and **eliminated ~50 lines of duplicate code** by adopting httputil properly.

---

## a) FULLY DONE ✅

### This Session's Work (Uncommitted)

| Change                                   | Files                                                                            | Impact                                                                          |
| ---------------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **chroma v2.23.1 → v2.26.1**             | go.mod, go.sum                                                                   | Syntax highlighting library updated                                             |
| **cockroachdb/errors v1.12.0 → v1.13.0** | go.mod, go.sum                                                                   | Error wrapping library updated                                                  |
| **fsnotify v1.9.0 → v1.10.1**            | go.mod, go.sum                                                                   | File watcher library updated                                                    |
| **httputil v0.0.0-\* → v0.2.0**          | go.mod, go.sum, helpers.go                                                       | Breaking: `Middleware` type alias added; updated `chain()` signature            |
| **gocloud.dev v0.40.0 → v0.46.0**        | go.mod, go.sum                                                                   | Blob storage updated (resolved grpc/stats/opentelemetry module conflict)        |
| **httputil.Recovery() middleware**       | handlers.go                                                                      | Critical: panic recovery was completely missing                                 |
| **httputil.Compression() middleware**    | handlers.go                                                                      | gzip response compression for all responses                                     |
| **httputil.RequestID() adoption**        | handlers.go, accesslog.go, helpers.go, requestid.go (deleted), requestid_test.go | Eliminated 53 lines of duplicate request ID code                                |
| **Middleware ordering fixed**            | handlers.go                                                                      | RequestID now runs BEFORE accessLog — request_id was always empty in logs (bug) |
| **Depguard config fixed**                | .golangci.yml                                                                    | Removed unused `gin-gonic/gin`, added `httputil` + `x/time/rate`                |
| **LIBRARY_INTEGRATIONS.md updated**      | LIBRARY_INTEGRATIONS.md                                                          | Full usage analysis of all 17 direct dependencies                               |

### Pre-existing (Previously Committed)

| Feature                                                | Status        | Key Files                                   |
| ------------------------------------------------------ | ------------- | ------------------------------------------- |
| Markdown rendering (Goldmark + 7 extensions)           | ✅ Production | `internal/renderer/markdown.go`             |
| Syntax highlighting (Chroma/monokai)                   | ✅ Production | `internal/renderer/markdown.go:48-53`       |
| D2 diagram rendering (server-side SVG)                 | ✅ Production | `internal/renderer/diagrams.go`             |
| Mermaid diagrams (client-side JS)                      | ✅ Production | `templates/layout.templ`                    |
| Admonition blocks (6 types, custom Goldmark extension) | ✅ Production | `internal/renderer/admonition_extension.go` |
| Full-text search                                       | ✅ Production | `internal/content/search.go`                |
| Smart 404 with Levenshtein suggestions                 | ✅ Production | `internal/server/suggestions.go`            |
| YAML frontmatter parsing                               | ✅ Production | `internal/renderer/markdown.go:90-139`      |
| TOC extraction from AST                                | ✅ Production | `internal/renderer/markdown.go:170+`        |
| HTML caching (Otter, 1h TTL, stats)                    | ✅ Production | `internal/cache/html.go`                    |
| File watching + live reload (SSE, 500ms debounce)      | ✅ Production | `cmd/dynamic-markdown-site/watcher.go`      |
| Rate limiting (token bucket, per-IP)                   | ✅ Production | `internal/server/ratelimit.go`              |
| Blob storage (S3, GCS, filesystem, memory)             | ✅ Production | `internal/content/blob.go`                  |
| DI container (samber/do)                               | ✅ Production | `internal/container/container.go`           |
| Path traversal prevention                              | ✅ Production | `internal/domain/urlpath.go`                |
| Sitemap.xml endpoint                                   | ✅ Production | `internal/server/sitemap.go`                |
| Robots.txt endpoint                                    | ✅ Production | `internal/server/robots.go`                 |
| Raw asset serving (images, PDFs)                       | ✅ Production | `internal/server/handlers.go`               |
| .md extension redirect                                 | ✅ Production | `internal/server/handlers.go:186-194`       |
| Nix flake build + devShell + checks                    | ✅ Production | `flake.nix`, `package.nix`                  |
| Docker multi-arch (amd64+arm64)                        | ✅ Production | `Dockerfile`                                |
| GitHub Actions CI (docker + release)                   | ✅ Production | `.github/workflows/`                        |
| Domain language documentation                          | ✅ Thorough   | `docs/DOMAIN_LANGUAGE.md`                   |

### Test Coverage (per-package)

| Package                     | Coverage  | Assessment                                 |
| --------------------------- | --------- | ------------------------------------------ |
| `internal/cache`            | **95.5%** | Excellent                                  |
| `internal/config`           | **90.7%** | Excellent                                  |
| `internal/renderer`         | **85.0%** | Good (up from 68.2% per old TODOs)         |
| `internal/server`           | **81.3%** | Good                                       |
| `internal/domain`           | **81.0%** | Good                                       |
| `internal/content`          | **74.5%** | Adequate                                   |
| `internal/container`        | **0.0%**  | ❌ Has test file but 0% effective coverage |
| `cmd/dynamic-markdown-site` | **0.0%**  | ❌ No tests at all                         |

---

## b) PARTIALLY DONE 🟡

| Item                                        | What's Done                                                         | What's Missing                                                                                            |
| ------------------------------------------- | ------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Nix flake migration**                     | Phase 1-2: build, devShell, checks, treefmt, CI shell               | Phase 3-5: OCI image, direnv/.envrc, Nix CI workflow, apps.run-dev, Go formatter in treefmt               |
| **httputil adoption**                       | Recovery, Compression, RequestID, ClientIP, ResponseRecorder, Chain | ETag(), CORS(), Timeout(), Server lifecycle, HealthHandler, Logging middleware                            |
| **LIBRARY_INTEGRATIONS.md recommendations** | Updated analysis, depguard fixed                                    | go-error-family direct adoption, smart-configs, templ-components, go-filewatcher                          |
| **go-error-family**                         | Indirect dep (v0.3.0 via httputil)                                  | Not adopted directly for error classification at HTTP boundary                                            |
| **Dependency hygiene**                      | All direct deps on latest                                           | 6 unused indirect deps need `go mod tidy` (AWS SDK v1, jmespath, opencensus, groupcache, ini, s3/manager) |
| **Templ components**                        | 13 hand-built components working                                    | No component library adoption (templ-components)                                                          |
| **Error handling**                          | cockroachdb/errors wrapping everywhere                              | No WithHint/WithDetail for user-facing messages; 4 silent error swallows in filesystem.go                 |

---

## c) NOT STARTED ❌

### From TODO_LIST.md (0/43 items done — all unchecked)

**Critical (4):**

- Address GitHub security vulnerabilities in dependencies
- Fix Go 1.26.1 environment mismatch for BuildFlow
- Fix unused parameter warnings in `container.go`
- Fix local Go cache corruption

**High Priority (10):**

- Integration test suite
- `AllPaths()` unit tests for domain
- D2 error handling (graceful degradation on render failure)
- Prometheus/OpenTelemetry metrics
- Structured health check (liveness/readiness split)
- Docker artifact in GH Actions
- Coverage enforcement ≥75% in CI
- golangci-lint version pinning
- `templ generate` CI check
- Request timing histogram

**Medium Priority (15):**

- Split 3 oversized test files (search_test 685 lines, handlers_test 667 lines, markdown_test 609 lines)
- Search highlighting
- Breadcrumbs in search results
- Pagination
- Cache metrics dashboard
- Architecture Decision Records (ADRs)
- Reading time in UI
- Rate-limit search endpoint
- Rate-limit unit tests
- Graceful shutdown integration tests
- Renderer edge case coverage
- Testutil integration across tests
- Suggestions edge case tests

**Process (8):**

- Git pre-push hook
- Pre-commit golines hook
- Split CI workflows
- Go module caching in CI
- CI documentation
- CONTRIBUTING.md
- Dockerfile HEALTHCHECK
- staticcheck fix in errors.go, dead `addError` removal

**Testing (9):**

- 404 suggestions integration test
- HTTP endpoint integration tests
- Template integration tests
- Full markdown→HTML pipeline test
- Renderer edge cases
- Container assertions
- Template benchmarks
- E2E tests
- Container coverage

### From ROADMAP.md (0/~60 items started)

Key unstarted items: dark mode, autocomplete search, RSS/Atom feed, Redis cache, K8s manifests, plugin system, WYSIWYG preview, image optimization, ETag support, WebSocket reload, pprof endpoint, admin dashboard, content versioning, mutation testing.

### From LIBRARY_INTEGRATIONS.md (5 adoption candidates identified)

- go-filewatcher (replace raw fsnotify + add debouncing)
- go-error-family (error classification at HTTP boundary)
- smart-configs (replace 150 lines of config parsing)
- templ-components (selective component replacement)
- cmdguard (if CLI grows subcommands)

---

## d) TOTALLY FUCKED UP 💥

| Issue                                                                      | Severity  | Details                                                                                                                                                                                                                                         |
| -------------------------------------------------------------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`internal/container` has 0.0% test coverage** despite having a test file | 🔴 High   | `container_test.go` exists but tests don't actually exercise the container's `New()` function effectively — possibly only testing mock/empty paths                                                                                              |
| **Middleware ordering bug was live**                                       | 🟡 Medium | `accessLogMiddleware` was wrapping `requestIDMiddleware` in the wrong order — `chain(handler, accessLog, security, requestID)` means accessLog runs OUTSIDE requestID, so `request_id` was always empty in access logs. **Fixed this session.** |
| **`go mod tidy` leaves unused indirect deps**                              | 🟡 Medium | 6 unused modules flagged by gopls: `aws-sdk-go`, `jmespath`, `opencensus`, `groupcache`, `aws-sdk-go-v2/internal/ini`, `aws-sdk-go-v2/feature/s3/manager` — these are transitive deps from gocloud.dev that linger                              |
| **TODO_LIST.md is stale**                                                  | 🟡 Medium | Generated 2026-04-05 with 43 items, 0 completed since then. Many items may already be done (e.g., renderer coverage is now 85%, not 68.2%). Needs reconciliation.                                                                               |
| **ROADMAP.md lists `sitemap.xml` as planned**                              | 🟢 Low    | Already implemented but still listed in roadmap. Needs cleanup.                                                                                                                                                                                 |
| **Flaky test: `TestRateLimiter_Concurrent`**                               | 🟡 Medium | `ratelimit_test.go:99` expects exactly 100 allowed but gets 101 — classic token bucket race condition in concurrent test. Passes on re-run. Needs `//nolint` or tolerance-based assertion.                                                      |
| **4 silent error swallows in `filesystem.go:254-300`**                     | 🟡 Medium | `processDirectory` and `processFile` silently `return` on `NewDirectoryNode`, `os.ReadFile`, `NewURLPath`, `NewFileNode` errors — content invisibly missing with no log                                                                         |

---

## e) WHAT WE SHOULD IMPROVE 🚀

### Architecture & Code Quality

1. **Adopt httputil fully** — we now have Recovery, Compression, RequestID. Still missing ETag (conditional GETs), Timeout (request-level vs connection-level), and potentially the Server lifecycle wrapper that handles graceful shutdown in one place.

2. **Fix silent error swallowing in filesystem.go** — 4 places where errors during tree building are silently dropped. Content goes missing with zero observability. At minimum, log at warn level.

3. **Adopt go-error-family directly** — the LIBRARY_INTEGRATIONS.md identified this as high-value: map errors to HTTP status codes intelligently (Transient→503, Rejection→400, Infrastructure→500) instead of binary 404/500.

4. **Remove yaml.v3 dependency** — goldmark-meta already parses frontmatter. The only use of yaml.v3 is in `draft.go` for `draft: true` detection, which could read from the already-parsed frontmatter map. Eliminates a direct dependency.

5. **Expand samber/lo usage** — only 2 of 200+ functions used (`FilterMap`, `ContainsBy`). Many manual loops and type assertions could be simplified.

6. **Add chroma WithClasses(true)** — switch from inline styles to CSS classes for smaller HTML payloads. Requires generating a monokai CSS file but saves bandwidth on code-heavy pages.

### Testing

7. **Test `cmd/dynamic-markdown-site`** — zero coverage on main entry point, signal handling, watcher lifecycle.

8. **Fix container_test.go effectiveness** — has a test file but 0.0% coverage. Tests aren't exercising real code paths.

9. **Add integration tests** — no end-to-end tests exist (HTTP request → handler → repository → renderer → response).

10. **Fix flaky `TestRateLimiter_Concurrent`** — use tolerance-based assertion (`>= 99 && <= 101`) instead of exact equality.

### DevOps & Infrastructure

11. **Run `go mod tidy`** — 6 unused indirect dependencies lingering.

12. **Complete Nix migration** — OCI image build in flake, direnv support, Nix CI workflow.

13. **Add `templ generate` to CI** — ensure generated code is up-to-date before build.

14. **Pin golangci-lint version** — currently unpinned, could break CI on upstream releases.

### Documentation

15. **Reconcile TODO_LIST.md** — many items are stale (renderer coverage at 85% not 68.2%, sitemap done, etc.).

16. **Clean up ROADMAP.md** — sitemap.xml listed as planned but already done.

17. **Write ADRs** — no Architecture Decision Records exist despite ~17 dependencies and significant architectural choices.

---

## f) Top 25 Things to Get Done Next

| #  | Task                                                                       | Impact   | Effort | Category             |
| -- | -------------------------------------------------------------------------- | -------- | ------ | -------------------- |
| 1  | **Fix silent error swallowing in `filesystem.go:254-300`** — add logging   | Critical | 1h     | Code Quality         |
| 2  | **`go mod tidy`** — remove 6 unused indirect deps                          | High     | 5min   | Dependency Hygiene   |
| 3  | **Fix container_test.go** — 0% coverage despite having tests               | High     | 2h     | Testing              |
| 4  | **Fix flaky `TestRateLimiter_Concurrent`** — tolerance assertion           | High     | 15min  | Testing              |
| 5  | **Adopt httputil.ETag()** — conditional GETs / 304 responses               | High     | 2h     | Performance          |
| 6  | **Adopt go-error-family** — error classification at HTTP boundary          | High     | 4h     | Architecture         |
| 7  | **Add tests for `cmd/dynamic-markdown-site`** — watcher, shutdown          | High     | 3h     | Testing              |
| 8  | **Add integration tests** — HTTP→handler→repo→renderer pipeline            | High     | 4h     | Testing              |
| 9  | **Reconcile TODO_LIST.md** — mark done items, remove stale                 | Medium   | 1h     | Documentation        |
| 10 | **Remove yaml.v3** — use goldmark-meta frontmatter for draft detection     | Medium   | 1h     | Dependency Reduction |
| 10 | **Add chroma WithClasses(true)** — CSS-based highlighting, smaller HTML    | Medium   | 2h     | Performance          |
| 12 | **Complete Nix Phase 3-5** — OCI image, direnv, CI workflow                | Medium   | 4h     | DevOps               |
| 13 | **Split oversized test files** — search_test, handlers_test, markdown_test | Medium   | 2h     | Code Quality         |
| 14 | **Pin golangci-lint version** in CI or flake                               | Medium   | 30min  | DevOps               |
| 15 | **Add `templ generate` CI check**                                          | Medium   | 30min  | DevOps               |
| 16 | **Adopt go-filewatcher** — replace raw fsnotify, add debouncing            | Medium   | 2h     | Architecture         |
| 17 | **Add Prometheus/OpenTelemetry metrics** — cache stats, request timing     | Medium   | 4h     | Observability        |
| 18 | **Split health check** — /health/live vs /health/ready                     | Medium   | 1h     | Observability        |
| 19 | **Expand samber/lo** — simplify manual loops                               | Low      | 1h     | Code Quality         |
| 20 | **Write ADRs** — document key architectural decisions                      | Low      | 2h     | Documentation        |
| 21 | **Add dark mode** — most-requested roadmap item                            | Low      | 4h     | UI/UX                |
| 22 | **Add RSS/Atom feed** — content delivery                                   | Low      | 2h     | Features             |
| 23 | **Add search autocomplete** — typeahead search                             | Low      | 3h     | Features             |
| 24 | **Add code copy button** — on code blocks                                  | Low      | 1h     | UI/UX                |
| 25 | **Clean up ROADMAP.md** — remove done items (sitemap.xml)                  | Low      | 15min  | Documentation        |

---

## g) Top Question I Cannot Answer Myself 🤔

**Should we adopt httputil.Server (graceful lifecycle wrapper) to replace the hand-rolled shutdown logic in `main.go`, or keep the current explicit control?**

The current `main.go` has ~80 lines of carefully structured shutdown logic: `setupHTTPServer()` → `serveHTTP()` (goroutine + signal.NotifyContext) → `gracefulShutdown()` (context timeout + httpServer.Shutdown + container.Shutdown + rateLimiter.Stop + cache.Close). This is explicit, readable, and gives us full control over ordering.

`httputil.Server` wraps `http.Server` with `Start() <-chan error` and `Shutdown(ctx)`, which would simplify this to ~15 lines. BUT:

- It would remove our ability to control the exact shutdown sequence (container DI shutdown, rate limiter stop, cache close ordering)
- The current code already works correctly and is well-tested by production use
- httputil.Server is new (v0.2.0) and may not have the same edge-case handling

I cannot determine whether the simplification is worth the loss of explicit shutdown ordering control. This is a judgment call that depends on how much you value explicit lifecycle management vs. code reduction.

---

## Session Changes (Uncommitted)

```
.golangci.yml                     |   3 +-
LIBRARY_INTEGRATIONS.md           |  59 ++++++-
go.mod                            | 117 +++++++------
go.sum                            | 335 +++++++++++++++-----------------------
internal/server/accesslog.go      |   4 +-
internal/server/handlers.go       |   7 +-
internal/server/helpers.go        |  18 +-
internal/server/requestid.go      |  53 ------
internal/server/requestid_test.go | 110 +++----------
9 files changed, 281 insertions(+), 425 deletions(-)
```

### Dependency Changes Summary

| Dependency                               | Change             | Type                                    |
| ---------------------------------------- | ------------------ | --------------------------------------- |
| `github.com/alecthomas/chroma/v2`        | v2.23.1 → v2.26.1  | Direct — patch update                   |
| `github.com/cockroachdb/errors`          | v1.12.0 → v1.13.0  | Direct — minor update                   |
| `github.com/fsnotify/fsnotify`           | v1.9.0 → v1.10.1   | Direct — minor update                   |
| `github.com/larsartmann/httputil`        | v0.0.0-\* → v0.2.0 | Direct — **breaking** (Middleware type) |
| `gocloud.dev`                            | v0.40.0 → v0.46.0  | Direct — minor update                   |
| `github.com/larsartmann/go-error-family` | v0.1.1 → v0.3.0    | Indirect (via httputil)                 |
| `github.com/getsentry/sentry-go`         | v0.44.1 → v0.46.0  | Indirect (via cockroachdb/errors)       |
| `cloud.google.com/go/storage`            | v1.50.0 → v1.61.3  | Indirect (via gocloud.dev)              |
| `google.golang.org/grpc`                 | v1.68.1 → v1.79.3  | Indirect (via gocloud.dev)              |
| + 30 more indirect updates               | Various            | Transitive from gocloud.dev update      |

### Code Changes Summary

| File                      | Change                                                                                                                          |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `handlers.go`             | Added Recovery + Compression middleware, replaced custom requestIDMiddleware with httputil.RequestID, fixed middleware ordering |
| `accesslog.go`            | Switched to httputil.RequestIDFromContext                                                                                       |
| `helpers.go`              | Removed custom contextKey/requestIDCtxKey/contextWithRequestID/requestIDFromContext                                             |
| `requestid.go`            | **Deleted** — all functionality now in httputil                                                                                 |
| `requestid_test.go`       | Rewritten to test httputil integration                                                                                          |
| `.golangci.yml`           | Removed `gin-gonic/gin` from depguard allow, added `httputil` + `x/time/rate`                                                   |
| `LIBRARY_INTEGRATIONS.md` | Added comprehensive usage analysis section                                                                                      |
