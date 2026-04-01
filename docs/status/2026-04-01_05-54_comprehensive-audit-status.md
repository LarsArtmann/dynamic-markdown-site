# Comprehensive Status Report — 2026-04-01 05:54

_After deep audit of every file in the codebase._

---

## Metrics Snapshot

| Metric | Value |
|---|---|
| Total commits | 96 |
| Go source files | 46 |
| Test files | 15 |
| Production Go LOC | ~3,960 |
| Test Go LOC | ~5,930 |
| Total Go LOC (incl. generated) | ~11,190 |
| Templ templates | 1 (417 lines) |
| Enabled linters | 62 |
| Direct dependencies | 14 |
| Status reports | 23 (this is #24) |
| Packages with tests | 7 / 9 testable |
| Test coverage | 66–100% (varies by package) |
| Docker image | ~16MB compressed |
| Unstaged changes | 3 files (handlers.go, testutil/http.go, new security.go) |

### Coverage by Package

| Package | Coverage |
|---|---|
| `internal/cache` | **100.0%** |
| `internal/config` | **94.5%** |
| `internal/content` | 76.4% |
| `internal/domain` | 75.8% |
| `internal/renderer` | 66.2% |
| `internal/server` | (build cache issue — last known ~78%) |
| `internal/container` | (build cache issue — last known ~85%) |
| `cmd/dynamic-markdown-site` | 0% (no tests) |
| `pkg/errors` | 0% (no tests) |

---

## A) FULLY DONE ✅

Things that work correctly and are complete.

### Infrastructure
- ✅ **Go module** — clean `go.mod` with 14 well-chosen direct dependencies
- ✅ **Dockerfile** — multi-stage build, static binary, distroless runtime (has bugs though — see D)
- ✅ **CI pipeline** — GitHub Actions: test + lint + Docker build + smoke test
- ✅ **Justfile** — build, test, lint, generate, bench, clean, install, run-dev
- ✅ **Graceful shutdown** — SIGINT/SIGTERM, 30s drain timeout, DI container teardown
- ✅ **Version injection** — `internal/version` package with ldflags for Version/Commit/BuildDate
- ✅ **Git history** — 96 clean commits, no merge conflicts

### Core Rendering
- ✅ **Goldmark markdown** — full extension suite (tables, strikethrough, task lists, definition lists, footnotes, linkify, typographer, auto heading IDs)
- ✅ **Chroma syntax highlighting** — Monokai theme, 200+ languages, zero config
- ✅ **YAML frontmatter** — title, description, author, date, tags, draft field extraction
- ✅ **Table of contents** — hierarchical (h2+), anchor links, sidebar navigation
- ✅ **Reading time** — 200 WPM estimate, displayed in article header
- ✅ **Templ templates** — compile-time type-safe HTML, 5 views (layout, directory, file, search, error)

### Diagrams
- ✅ **D2 server-side rendering** — compiles to SVG via `d2lib.Compile`, dagre layout, wrapped in `<div class="d2-diagram">`
- ✅ **Mermaid client-side rendering** — `<pre class="mermaid">` + Mermaid.js v11 CDN, loaded on-demand
- ✅ **Goldmark diagram extension** — intercepts fenced code blocks for `d2`/`mermaid` during AST walk
- ✅ **Graceful D2 degradation** — if `NewDiagramRenderer()` fails, continues without diagram support (container.go:131)
- ✅ **D2 error fallback** — renders as plain code block if compilation fails
- ✅ **Mermaid HTML escaping** — `<`, `>`, `&`, `"` escaped before embedding

### HTTP Server
- ✅ **Gin framework** — clean route registration, middleware chain
- ✅ **Health endpoint** — `/health` returns JSON `{"status":"healthy"}`
- ✅ **Content routing** — `/*path` resolves to directory or file view
- ✅ **Static file serving** — `//go:embed`, path traversal protection, content-type detection
- ✅ **Rate limiting** — per-IP sliding window (10/min on `/refresh`)
- ✅ **Request IDs** — crypto-random 32-char hex, `X-Request-ID` header propagation
- ✅ **Structured request logging** — IP, method, path, status, duration, errors
- ✅ **404 suggestions** — Levenshtein distance with prefix/substring boosting, up to 5 suggestions
- ✅ **Security headers middleware** — (unstaged) X-Content-Type-Options, X-Frame-Options, HSTS, CSP, Referrer-Policy, Permissions-Policy

### Content
- ✅ **Filesystem repository** — thread-safe (RWMutex), recursive directory walking, hidden file/blacklisted dir filtering
- ✅ **Content filtering** — skips `.`-prefixed, `node_modules`, `.git`, `vendor`, `dist`, `build`, `tmp`, `temp`
- ✅ **In-memory repository** — map-backed for testing
- ✅ **Repository interface** — `Get`, `Root`, `Refresh`, `LastModified`, `AllPaths`
- ✅ **Full-text search** — case-insensitive, 3-tier scoring (1.0/0.5/0.3), `<mark>` highlighting, snippet extraction
- ✅ **Content tree** — `DirectoryNode`/`FileNode` hierarchy with sorted children

### Caching
- ✅ **Otter auto-tuning cache** — 10,000 entries, 1-hour access TTL
- ✅ **Stats tracking** — hits, misses, evictions, hit ratio
- ✅ **Invalidation on refresh** — `/refresh` clears cache
- ✅ **Dev mode disables cache** — automatic via config

### Architecture
- ✅ **DI container** — samber/do/v2 with 7 providers, singleton lifecycle
- ✅ **Repository pattern** — filesystem/memory implementations behind interface
- ✅ **Domain types** — `URLPath` (traversal-safe), `HTML` (escape-aware), `ContentNode`, `RenderedFile` (immutable)
- ✅ **Error handling** — cockroachdb/errors with stack traces, sentinel errors
- ✅ **Error page fallback chain** — template → 500 page → plain text
- ✅ **Composition over inheritance** — small focused types, no deep hierarchies

### Testing
- ✅ **15 test files** across 7 packages
- ✅ **Parallel tests** where applicable
- ✅ **Benchmarks** — repository, renderer, HTTP handlers
- ✅ **Test utilities** — `internal/testutil` with shared fixtures
- ✅ **Mock repositories** — `FailingRepository`, `InMemoryRepository`
- ✅ **Table-driven tests** — standard pattern throughout

### Documentation
- ✅ **FEATURES.md** — 297 lines, 11 sections, comprehensive feature catalog
- ✅ **AGENTS.md** — 369 lines, developer guide with patterns and gotchas
- ✅ **README.md** — 254 lines, quick start, usage, architecture
- ✅ **Content examples** — `content/diagrams/README.md` with D2/Mermaid samples

---

## B) PARTIALLY DONE 🔧

Things that exist but have significant gaps or bugs.

### 🔧 Live Reload (Dev Mode)
- **Works**: SSE endpoint, browser auto-reload, toast notifications, `HasMermaid` flag loads Mermaid.js on-demand
- **Broken**: Endpoint `/api/live-reload` registered **unconditionally** — available in production (handlers.go:64)
- **Broken**: File watcher has **NO debouncing** — every file event triggers immediate full refresh (watcher.go:99-118). FEATURES.md falsely claims "500ms debounce"
- **Broken**: File watcher does **NOT invalidate cache** — stale content served for up to 1 hour after changes
- **Broken**: CORS wildcard `Access-Control-Allow-Origin: *` on SSE endpoint in production (livereload.go:94)

### 🔧 Security Headers (new, unstaged)
- **Exists**: `internal/server/security.go` with X-Content-Type-Options, X-Frame-Options, HSTS, CSP, Referrer-Policy, Permissions-Policy
- **Not committed**: File is untracked. `handlers.go` has the middleware wired but also unstaged
- **Issues**: CSP allows `'unsafe-inline'` scripts (needed for live reload in dev, but also applied in prod). HSTS set even when serving HTTP (breaks non-HTTPS deployments)

### 🔧 Diagram Support (D2 + Mermaid)
- **D2 works**: Server-side SVG rendering via d2lib, dagre layout, graceful fallback
- **Mermaid works**: Client-side via CDN, auto-detected and loaded
- **Issue**: `ProcessMarkdown()` in `diagrams.go:196` is dead code — goldmark `DiagramExtension` handles rendering during the AST pipeline
- **Issue**: No D2 render timeout — complex diagrams could hang indefinitely
- **Issue**: `codeBlockRegex` can't handle backticks inside diagram content

### 🔧 Request ID Middleware
- **Works**: Crypto-random 32-char hex generation
- **Broken**: Unvalidated `X-Request-ID` header blindly accepted — attacker can inject arbitrary data (requestid.go:52)
- **Broken**: Fallback ID is always the same predictable `"000102030405060708090a0b0c0d0e0f"` (requestid.go:37)
- **Broken**: Request IDs generated but **never included in log output** — `getRequestID()` exists but unused

### 🔧 Version Injection
- **Works**: `internal/version` package with `Version`, `Commit`, `BuildDate` vars
- **Broken**: Dockerfile `ARG COMMIT=$(git rev-parse...)` — Docker ARG defaults are literal strings, not shell expansions. Version info will be the literal `"$(git rev-parse --short HEAD 2>/dev/null || echo \"unknown\")"`
- **Broken**: `/health` endpoint doesn't include version info despite ldflags setup

### 🔧 Error Handling
- **Works**: cockroachdb/errors, sentinel errors, fallback chain, 404 suggestions
- **Broken**: `handleRefresh` returns raw `result.Error` in JSON — leaks filesystem paths (handlers.go:124)
- **Broken**: `getOrRenderContent` silently returns empty `RenderedContent{}` on render failure — user sees blank page (render.go:89)
- **Broken**: `renderComponent` may send 200 headers before detecting render failure — garbled output possible (errors.go:76)

### 🔧 Configuration
- **Works**: Flags + env vars, validation, dev mode
- **Broken**: Case-sensitive log level comparison — `-log-level INFO` passes validation but doesn't set Gin to release mode (config.go:147 vs main.go:140)
- **Broken**: Invalid port values silently ignored instead of returning errors (config.go:63)
- **Broken**: No timeout validation — `0s` or `-5s` accepted

### 🔧 URLPath Domain Type
- **Works**: `..` traversal prevention, path cleaning, safe URL construction
- **Broken**: Uses `filepath.Clean`/`filepath.Dir`/`filepath.Base` — OS-specific separators. Breaks on Windows (urlpath.go:18,53,65,85). Should use `path.Clean`/`path.Dir`/`path.Base`
- **Missing**: No maximum path length, no null byte validation

### 🔧 Content Repository
- **Works**: Thread-safe filesystem walking, filtering, tree building
- **Broken**: `processFile` silently swallows `os.ReadFile` and `NewFileNode` errors — file load failures invisible to operators (filesystem.go:261-276)
- **Missing**: No file size limit — multi-GB markdown files cause OOM
- **Missing**: No frontmatter draft filtering — `draft: true` files included despite README claiming exclusion

### 🔧 Test Utilities
- **Works**: HTTPTestRunner, ServerFixture, CacheFixture, ContentFixture
- **Unstaged change**: `init()` moved to `NewHTTPTestRunner()` — better pattern but not committed
- **Missing**: No tests for `pkg/errors`, `cmd/dynamic-markdown-site`, `templates`

---

## C) NOT STARTED ❌

Features that are documented/desired but not implemented.

1. ❌ **Draft file filtering** — `draft: true` in frontmatter should exclude files. README claims this works. It doesn't. No code checks the `Draft` field.
2. ❌ **File watcher debouncing** — FEATURES.md claims 500ms debounce. Comment in watcher.go references "full debouncing logic" but it's not implemented. Every file event triggers immediate full refresh.
3. ❌ **File watcher cache invalidation** — Watcher refreshes repository but doesn't clear HTML cache. Stale content served.
4. ❌ **Health endpoint enrichment** — Version/commit/buildDate are injected via ldflags but `/health` returns only `{"status":"healthy"}`. No version info exposed.
5. ❌ **Static asset caching** — No `Cache-Control`, `ETag`, or `Last-Modified` headers for CSS/favicon. Re-downloaded every request.
6. ❌ **Search rate limiting** — Only `/refresh` is rate limited. Search endpoint has no protection.
7. ❌ **Search pagination** — All results returned. No limit or pagination.
8. ❌ **Search query length limit** — Extremely long queries cause expensive operations.
9. ❌ **File watcher cancellation** — Goroutine runs forever, no shutdown path.
10. ❌ **Watcher for non-markdown files** — Only `.md`/`.markdown` changes trigger refresh. CSS/template changes ignored.
11. ❌ **Cache size configuration** — Hardcoded at 10,000 entries. No config option.
12. ❌ **Cache `Close()` method** — No way to release resources during shutdown.
13. ❌ **Authentication on `/refresh`** — Any client can trigger content refresh.
14. ❌ **PR-triggered CI** — Only runs on push to master. Pull requests get no feedback.
15. ❌ **Image push to registry** — CI only saves as artifact with 14-day retention. No deployment path.
16. ❌ **CHANGELOG.md** — Still at initial placeholder (`## [0.1.0] - 2026-01-01`). No entries despite 96 commits.
17. ❌ **License resolution** — README says "Proprietary", Dockerfile says "MIT". TODO_LIST notes this. Unresolved.
18. ❌ **`pkg/errors` removal** — Vestigial package, barely used, conflicts with stdlib `errors`, incompatible with `errors.Is`/`errors.As`. TODO_LIST notes removal but unchecked.
19. ❌ **Dead `FileNode` fields cleanup** — `html`, `toc`, `metadata`, `hasMermaid` fields never set (legacy from before `RenderedFile` refactor). TODO_LIST notes "Complete immutable FileNode refactor".
20. ❌ **Unicode anchor IDs** — `generateAnchorID` strips all non-ASCII characters. Non-English headings get empty anchors.
21. ❌ **`extractHeadingText` inline handling** — Only handles `*ast.Text` children. Inline code/links in headings lose content.
22. ❌ **Justfile `fix` command** — No `golines` auto-format command.
23. ❌ **Justfile `generate` dependency** — `build` and `run-dev` don't depend on `templ generate`. Forgetting causes compilation errors.
24. ❌ **Dockerfile HEALTHCHECK fix** — Uses `wget` which doesn't exist in `distroless/static-debian13`.
25. ❌ **Dockerfile SHA256 pin fix** — Fake repeating hash that doesn't match any real image.

---

## D) TOTALLY FUCKED UP 💥

Critical bugs that break things or are just plain wrong.

### 💥 1. Dockerfile Won't Build (3 separate issues)
- **Fake SHA256** (line 69): `sha256:fe9d4b46e7b7c5d1e0c5c7e0e0c5c7e0e0c5c7e0e0c5c7e0e0c5c7e0e0c5c7e` — this is a clearly fake repeating hash. Docker will refuse to pull it.
- **Chinese characters in OCI label** (line 87): `org.opencontainers软件.name` — `软件` means "software". Invalid label key.
- **ARG shell expansion** (lines 18-19): `ARG COMMIT=$(git rev-parse...)` — Docker ARG defaults are literal strings. The version info baked into the binary will be the literal string `"$(git rev-parse --short HEAD 2>/dev/null || echo \"unknown\")"`.

### 💥 2. File Watcher Doesn't Actually Work for Dev Mode
- **No debouncing**: Every file event triggers immediate `repo.Refresh()`. Rapid saves = multiple full filesystem walks.
- **No cache invalidation**: After refreshing the repository, the HTML cache still serves stale content for up to 1 hour. The `/refresh` HTTP endpoint does invalidate, but the watcher doesn't call it. **Dev mode live reload shows stale content.**

### 💥 3. Live Reload SSE Endpoint in Production
- Registered unconditionally in `RegisterRoutes` (handlers.go:64). Available in production.
- CORS wildcard `*` means any website can connect to the SSE stream.
- No client limit — unlimited SSE connections can exhaust server resources.

### 💥 4. Documentation Lies
| Claim | File | Reality |
|---|---|---|
| "Files with `draft: true` are excluded" | README.md:111 | Not implemented. Draft files included. |
| "500ms debounce" | FEATURES.md:99 | Not implemented. Immediate refresh. |
| "`GetOrCompute` prevents duplicate renders" | FEATURES.md:129 | `GetOrCompute` is dead code. Manual Get/Set used. |
| "`t.Parallel()` enforced by linter" | FEATURES.md:292 | `paralleltest` linter excluded for all test files. |
| "~75 linters" | README.md:187, FEATURES.md:297 | Actually 62. |
| "/api/live-reload — dev mode only" | README.md:147 | Endpoint always registered. |
| "Live reload connected → reloading" | FEATURES.md (implied) | File view doesn't set `DevMode` in LayoutProps (render.go:62-69). Live reload script not included on file pages. |

### 💥 5. URLPath Uses OS-Specific Path Operations
- `filepath.Clean`, `filepath.Dir`, `filepath.Base` in urlpath.go — these use OS-specific separators. On Windows, URL paths like `/foo/bar` would be mangled to `\foo\bar`. URL paths must use `path.Clean`/`path.Dir`/`path.Base` (always forward slashes).

### 💥 6. Silent Failures Make Debugging Impossible
- **File load errors swallowed** (filesystem.go:261-276) — operators can't see which files failed to load
- **Diagram degradation silent** (container.go:131) — no logging when diagram support is disabled
- **Render failures → blank pages** (render.go:89) — user sees nothing, no error shown
- **Cache `GetOrCompute` dead code** — thundering herd possible on uncached pages

---

## E) WHAT WE SHOULD IMPROVE

### Code Quality
1. **Fix all Dockerfile bugs** — fake SHA, Chinese chars, ARG expansion, HEALTHCHECK. This is blocking Docker builds.
2. **Fix file watcher** — add debouncing AND cache invalidation. Dev mode is currently broken for live reload.
3. **Fix documentation lies** — every claim in README/FEATURES must match reality. Remove false claims or implement the feature.
4. **Use `path` package not `filepath`** for URL path operations — cross-platform correctness.
5. **Stop swallowing errors** — log file load failures, diagram degradation, render failures.
6. **Remove dead code** — `ProcessMarkdown`, `GetOrCompute` (or use it), `FileNode` dead fields, `pkg/errors` package.
7. **Gate live reload behind dev mode** — conditional registration, no CORS wildcard in production.
8. **Validate request IDs** — reject oversized/malformed `X-Request-ID` headers.
9. **Include request IDs in logs** — `getRequestID()` exists but is never called from the request logger.

### Architecture
10. **Make cache size configurable** — via flag/env var, not hardcoded.
11. **Add `Close()` to cache** — resource cleanup during shutdown.
12. **Add context cancellation to file watcher** — stoppable goroutine.
13. **D2 render timeout** — prevent hangs on complex diagrams.

### Testing
14. **Fix go build cache corruption** — `go clean -cache` failed, leaving partial cache that breaks `-count=1` runs.
15. **Add tests for `cmd/dynamic-markdown-site`** — currently 0% coverage.
16. **Add tests for `pkg/errors`** — currently 0% coverage.
17. **Remove `paralleltest` exclusion** — either enforce or stop claiming enforcement.

### DevEx
18. **Justfile `fix` command** — auto-format with golines.
19. **Justfile generate dependency** — `build`/`run-dev` should depend on `generate`.
20. **Pin CI tool versions** — golangci-lint `version: latest` is unstable. templ version mismatch between CI and Dockerfile.

---

## F) Top #25 Things We Should Do Next

Prioritized by **impact × urgency** (critical bugs first, then improvements).

### P0 — Broken Right Now (do today)

| # | Item | Effort | Impact |
|---|---|---|---|
| 1 | **Fix Dockerfile**: real SHA256, fix Chinese label, move ARG expansion to RUN, fix HEALTHCHECK | 30 min | Docker builds will work |
| 2 | **Fix file watcher**: add 500ms debounce + cache invalidation on refresh | 30 min | Dev mode live reload actually works |
| 3 | **Fix live reload prod leak**: gate SSE registration behind `DevMode` config | 15 min | Security: no SSE in production |
| 4 | **Fix FEATURES.md lies**: remove false claims (debounce, GetOrCompute, paralleltest, draft, linter count) | 20 min | Documentation accuracy |
| 5 | **Fix README lies**: draft filtering, live-reload dev-only, linter count | 15 min | Documentation accuracy |

### P1 — High Impact, Low Effort (this week)

| # | Item | Effort | Impact |
|---|---|---|---|
| 6 | **Enrich `/health` endpoint**: include version, commit, buildDate, uptime | 30 min | Observability |
| 7 | **Log request IDs**: call `getRequestID()` from request logger | 5 min | Request tracing works |
| 8 | **Log file load errors**: use `stats.recordError` in `processFile` | 10 min | Debugging visibility |
| 9 | **Log diagram degradation**: warn when D2 renderer fails in container.go | 5 min | Debugging visibility |
| 10 | **Commit unstaged security headers**: review CSP, fix HSTS for HTTP, commit | 30 min | Security headers shipped |
| 11 | **Fix URLPath: `path` package not `filepath`** | 10 min | Cross-platform correctness |
| 12 | **Fix case-sensitive log level**: use `strings.ToLower` in Gin mode check | 5 min | `-log-level INFO` works |

### P2 — High Impact, Medium Effort (next sprint)

| # | Item | Effort | Impact |
|---|---|---|---|
| 13 | **Implement draft file filtering**: check frontmatter during `buildTree` | 1 hour | Long-requested feature |
| 14 | **Use `GetOrCompute` in render.go**: replace manual Get/Set, prevent thundering herd | 30 min | Performance |
| 15 | **Add static asset caching**: Cache-Control, ETag headers for CSS/favicon | 1 hour | Performance |
| 16 | **Update CHANGELOG.md**: document all major changes since v0.1.0 | 1 hour | Release readiness |
| 17 | **Remove dead code**: `ProcessMarkdown`, `FileNode` dead fields, `pkg/errors` | 1 hour | Code cleanliness |
| 18 | **Add Justfile `fix` + `generate` dependency** | 15 min | Developer experience |

### P3 — Worth Doing (backlog)

| # | Item | Effort | Impact |
|---|---|---|---|
| 19 | **Add search rate limiting + query length limit** | 1 hour | Protection against abuse |
| 20 | **Add file size limit in filesystem repository** | 30 min | OOM prevention |
| 21 | **Add file watcher cancellation** — context-based stop | 30 min | Clean shutdown |
| 22 | **Pin CI tool versions** (golangci-lint, templ) | 15 min | CI stability |
| 23 | **Add PR-triggered CI** + image push to registry | 1 hour | Deployment pipeline |
| 24 | **Fix Unicode anchor IDs** — transliterate or keep with hyphens | 1 hour | International content |
| 25 | **Fix `extractHeadingText` for inline code/links** | 30 min | Accurate TOC for all headings |

---

## G) My Top #1 Question I Cannot Answer Myself

**What is the intended deployment model?**

The Dockerfile exists, CI builds images, but:
- No image is pushed to any registry (only saved as a 14-day artifact)
- No PR CI (only master push)
- No deployment manifests (no k8s, no docker-compose, no terraform)
- No tagging/versioning strategy beyond `dev` / git SHA
- License is contradictory (MIT vs Proprietary)

This affects almost every priority decision:
- If **self-hosted single-binary**: focus on config, CLI UX, documentation
- If **Docker-first**: fix Dockerfile bugs immediately, add registry push, add deployment docs
- If **cloud-deployed**: add k8s manifests, health probes (real ones), metrics endpoint
- If **open-source library**: resolve license, add contribution guide, add PR CI, fix documentation accuracy

**I recommend fixing the Dockerfile regardless** (it's currently broken), but the broader deployment strategy determines whether we invest in CI/CD, observability, or distribution next.

---

## Unstaged Work-In-Progress

The working tree has 3 unstaged files from a previous session:

| File | Status | What It Does |
|---|---|---|
| `internal/server/security.go` | New (untracked) | Security headers middleware |
| `internal/server/handlers.go` | Modified | Wires `securityHeadersMiddleware()` into route chain |
| `internal/testutil/http.go` | Modified | Moves `gin.SetMode(gin.TestMode)` from `init()` to `NewHTTPTestRunner()` |

These should be reviewed (CSP `unsafe-inline` concern, HSTS-over-HTTP concern) and committed.
