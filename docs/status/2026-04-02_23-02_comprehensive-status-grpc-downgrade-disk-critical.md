# Dynamic Markdown Site — Comprehensive Status Report

**Generated:** 2026-04-02 23:02 CEST
**Reporter:** Crush (GLM-4.5-Air via Crush)
**Session:** Continuation of multi-session refactoring effort

---

## Executive Summary

**Project health: MODERATE.** The codebase has a solid architecture with ~4,300 LOC (production) and ~7,400 LOC (tests), ~12K total Go lines. The refactoring work from sessions 1-8 is **committed and pushed**. However, the system environment is under severe disk pressure (2.1GB free / 229GB, 99.9% full), making builds and tests extremely slow or impossible. The grpc dependency was **downgraded** rather than updated (v1.77.0 → v1.68.1), which is concerning for security. No build/test/lint verification has been completed this session due to environment constraints.

---

## a) FULLY DONE

### Committed & Pushed to origin/master

| # | Commit | Task | Status |
|---|--------|------|--------|
| 1 | `8906c10` | Delete testutil package (ghost, 0 imports) | **DONE** |
| 2 | Prior session | Unify skipDirs list (content vs watcher) | **DONE** |
| 3 | Prior session | Unify isMarkdownFile (content vs watcher) | **DONE** |
| 4 | Prior session | Unify getContentType (content vs server) | **DONE** |
| 5 | `983431f` | Use cache.GetOrCompute in render.go | **DONE** |
| 6 | `983431f` | Implement HasReadme properly (was hardcoded false) | **DONE** |
| 7 | `983431f` | Render SearchResult.Snippet in template | **DONE** |
| 8 | `983431f` | Unify SuggestedPath type into domain package | **DONE** |

### Earlier Completed (from TODO_LIST.md)

- Security headers middleware
- Lint compliance (75 linters, 0 issues at last check)
- Immutable FileNode refactor
- Hot-reload / SSE live reload for dev mode
- Docker image (distroless, multi-arch: amd64+arm64)
- GitHub Actions CI/CD
- Draft filtering from frontmatter
- Admonition/alert block extension
- Sitemap.xml + robots.txt
- Request ID middleware
- Access logging middleware
- Binary version via ldflags
- Config validation
- justfile with pre-push
- EditorConfig
- Request tracing
- Changelog

---

## b) PARTIALLY DONE

| Task | What's Done | What's Missing |
|------|-------------|----------------|
| **grpc/Dependabot security fix** | go.mod changed from v1.77.0 to v1.68.1 | **DOWNGRADED instead of upgraded.** The Dependabot alert was for v1.77.0 or earlier. The fix should upgrade to latest (v1.80+). Current v1.68.1 is WORSE. Need `go get google.golang.org/grpc@latest && go mod tidy`. |
| **Build/Test/Lint verification** | Code changes compile-tested in prior session before commit | **Not verified this session.** Go build cache was corrupted then cleared. Disk at 99.9% prevents rebuilding (6.3GB module cache + 1.8GB build cache). No `go build`, `go test`, or `golangci-lint` has completed this session. |
| **Status docs cleanup** | 17 status docs exist in `docs/status/` (3,759 lines) | Most are stale session artifacts. Only 2-3 have lasting value. Should be archived/deleted. |

---

## c) NOT STARTED (from TODO_LIST.md, curated)

### HIGH Priority
- Address GitHub security vulnerabilities in dependencies (grpc is WORSE now)

### MEDIUM Priority
- Fix Go 1.26.1 environment mismatch for BuildFlow
- Split large test files (search_test.go 685 lines, handlers_test.go 667 lines, markdown_test.go 609 lines)
- Fix unused parameter warnings in container.go
- Create integration test suite
- Add request timing middleware
- Content search highlighting (partial — snippets rendered, but no `<mark>` in results page)
- Search result pagination
- Cache hit/miss metrics endpoint
- Docker HEALTHCHECK
- Architecture decision records
- Reading time estimates in UI

### CI/CD
- Separate CI workflows: test.yml (fast) + docker.yml (build)
- golangci-lint version pinning in workflow
- `templ generate` check in CI
- Coverage enforcement (≥75%)
- Go module/build caching in CI
- Benchmark regression tracking
- Git pre-push hook
- Pre-commit hook for golines

### Features (Not Started)
- Dark mode / theme toggle
- Syntax highlighting themes
- Keyboard navigation
- Print stylesheet
- Code copy button
- Breadcrumbs for deep navigation
- Gzip/brotli compression
- ETag/If-None-Match support
- RSS/Atom feed
- Content tags and filtering
- Search autocomplete
- Prometheus metrics endpoint
- pprof profiling endpoint
- Full-text search with Bleve/Meilisearch
- Plugin system
- Admin dashboard/endpoints

---

## d) TOTALLY FUCKED UP

### 1. grpc Dependency — DOWNGRADED Instead of Upgraded

**Severity: HIGH**

The Dependabot alert was about a vulnerability in `google.golang.org/grpc`. The correct fix is to **upgrade** to latest. Instead, commit `983431f` **downgraded** from v1.77.0 to v1.68.1, making the security posture WORSE.

Additionally, `go mod tidy` pulled in gocloud.dev v0.40.0 (down from v0.45.0) and various OpenTelemetry library version changes. The dependency tree is now a mix of old and new versions.

**Impact:** The security vulnerability is NOT fixed. It's worse. This commit is already pushed to origin/master.

**Fix:** `GOWORK=off go get google.golang.org/grpc@latest && GOWORK=off go mod tidy`

### 2. Disk Space — CRITICAL (99.9% full, 2.1GB free)

**Severity: ENVIRONMENT BLOCKER**

| Path | Size |
|------|------|
| Go module cache (`~/go/pkg/mod/`) | 6.3GB |
| Go build cache (`~/Library/Caches/go-build/`) | 1.8GB |
| **Total recoverable** | **~8.1GB** |

Cannot build, test, or lint. Go compiler needs temporary space during compilation. The build cache was cleared earlier (`go clean -cache`) to fix corruption, which means every subsequent build recompiles from scratch, consuming even more disk during the process.

**Fix:** `go clean -cache` (saves 1.8GB), or prune module cache with `go clean -modcache` (saves 6.3GB but requires re-downloading everything).

### 3. No Verification Before Push

**Severity: PROCESS**

The code was committed and pushed to origin/master without verifying the build, tests, or lint pass in the current environment. This means:

- The grpc downgrade is live on master
- No guarantee the current code compiles (though it did in prior session before commit)
- Lint issues (golines long lines) may still exist

### 4. Session Context Handoff Issues

**Severity: MODERATE**

The previous session's comprehensive context document is accurate but highlights a pattern:
- Too many changes batched into one commit (8 files, 8 logical changes)
- Should have been 8 small commits, verified independently
- The grpc change should never have been bundled with the refactoring

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **One commit per logical change** — The 8-file mega-commit `983431f` bundles domain type consolidation, cache refactoring, template changes, and dependency downgrades. Each should be separate.
2. **Build → Test → Lint → Commit** — Never commit without verifying all three pass.
3. **Never bundle dependency changes with code changes** — Dep updates are separate commits.
4. **Read the diff before committing** — The grpc downgrade would have been caught.
5. **Clean up status docs** — 17 stale reports (3,759 lines) in `docs/status/` are noise. Delete or archive.

### Technical Improvements

1. **Free disk space** — The Go module cache (6.3GB) is the biggest offender. Run `go clean -modcache` if needed.
2. **Install golines** — Long-line formatting is a recurring issue. `go install github.com/segmentio/golines@latest`
3. **Add `go.sum` verification to CI** — Prevent dependency drift.
4. **Use `GOWORK=off` consistently** — The parent `go.work` causes subtle issues.
5. **Remove stale docs** — `docs/status/` has 17 files, most are session-specific noise.

### Architecture Improvements

1. **Split Repository interface** — Reader vs Refresher (current interface is too wide)
2. **Structured error types** — `Is()`/`As()`/`Unwrap()` for domain errors
3. **Consider removing gocloud.dev** — It pulls in 100+ transitive deps including grpc, protobuf, OpenTelemetry. If not actively used, it's massive overhead for a markdown server.
4. **Template type safety** — The `domain.SuggestedPath` in templates is good; continue this pattern for all template types.

---

## f) Top 25 Things We Should Get Done Next

Sorted by **impact × urgency / effort**:

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 1 | **Free disk space** (`go clean -modcache`) | CRITICAL | 1 min | Unblocks everything |
| 2 | **Fix grpc dependency** — upgrade to latest, not downgrade | HIGH | 5 min | Security vulnerability is LIVE on master |
| 3 | **Verify build passes** (`GOWORK=off go build ./...`) | HIGH | 5 min | Confirm code health |
| 4 | **Verify tests pass** (`GOWORK=off go test ./... -count=1`) | HIGH | 2 min | Confirm correctness |
| 5 | **Run linter** (`GOWORK=off golangci-lint run ./...`) | HIGH | 2 min | Confirm quality |
| 6 | **Delete stale status docs** (15 of 17 files in `docs/status/`) | MEDIUM | 2 min | Reduce noise, save space |
| 7 | **Install golines** (`go install github.com/segmentio/golines@latest`) | MEDIUM | 1 min | Fix formatting issues permanently |
| 8 | **Remove unused `coverage.out`** (56KB) | LOW | 1 min | Stale artifact |
| 9 | **Add git pre-push hook** (`just pre-push`) | MEDIUM | 5 min | Prevent future bad pushes |
| 10 | **Split large test files** (3 files >600 lines each) | MEDIUM | 30 min | Maintainability |
| 11 | **Separate CI workflows** (test.yml + docker.yml) | MEDIUM | 20 min | Faster feedback on PRs |
| 12 | **Pin golangci-lint version in CI** | MEDIUM | 5 min | Reproducible linting |
| 13 | **Add `templ generate` check to CI** | MEDIUM | 10 min | Catch template regression |
| 14 | **Audit go.mod for unused deps** (gocloud.dev?) | HIGH | 15 min | 6.3GB module cache is 90% from transitive deps |
| 15 | **Add gzip/brotli compression** | HIGH | 15 min | Major performance win for text-heavy site |
| 16 | **Add ETag/If-None-Match** | MEDIUM | 15 min | Bandwidth savings |
| 17 | **Dark mode CSS + theme toggle** | HIGH | 30 min | User-facing impact, high visibility |
| 18 | **Add code copy button** to code blocks | HIGH | 15 min | UX staple for documentation sites |
| 19 | **Split Repository interface** (Reader/Refresher) | MEDIUM | 20 min | Better abstraction |
| 20 | **Add Docker HEALTHCHECK** | MEDIUM | 5 min | Production readiness |
| 21 | **Add Prometheus/pprof endpoints** | MEDIUM | 20 min | Observability |
| 22 | **RSS/Atom feed generation** | MEDIUM | 20 min | Content discoverability |
| 23 | **Breadcrumbs for deep directory navigation** | MEDIUM | 15 min | Navigation UX |
| 24 | **Rate limit search endpoint** | MEDIUM | 5 min | Security (refresh is limited, search isn't) |
| 25 | **Content search autocomplete** | MEDIUM | 30 min | UX improvement |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Is `gocloud.dev` actually used by this project?**

The `go.mod` shows `gocloud.dev` as a dependency, which transitively pulls in:
- `google.golang.org/grpc` (the security concern)
- `google.golang.org/protobuf`
- `go.opentelemetry.io/*` (multiple packages)
- `cloud.google.com/go/*` (multiple packages)
- ~100+ other transitive dependencies

For a **markdown-to-HTML web server**, this is enormous dependency overhead. The project has blob storage (`internal/content/blob.go`) which might use `gocloud.dev`, but if it's not in active use, removing it would:
1. Eliminate the grpc security concern entirely
2. Cut the module cache from 6.3GB to ~1-2GB
3. Reduce Docker image size
4. Speed up builds by 3-5x

**Action needed:** Check if `blob.go` is actually wired in and if `gocloud.dev` is imported anywhere in the non-test code path. If not, `go mod tidy` should remove it.

---

## Environment State

| Metric | Value |
|--------|-------|
| Disk free | **2.1GB / 229GB (99.9% full)** |
| Go version | 1.26.1 darwin/arm64 |
| Go module cache | 6.3GB |
| Go build cache | 1.8GB |
| Project size | 2.4MB |
| Git status | **Clean** (nothing to commit) |
| Branch | master (up to date with origin/master) |
| Remote | `git@github.com:LarsArtmann/dynamic-markdown-site.git` |
| Unpushed commits | **0** (everything is pushed) |

## Key Commits (Recent)

| Hash | Message |
|------|---------|
| `983431f` | refactor: consolidate domain types and improve content rendering architecture |
| `8906c10` | refactor: remove testutil package and update linter config |
| `ba6eae6` | docs(status): full comprehensive status |
| `0192273` | refactor: deduplicate helpers, optimize ContentTree, add interface checks |

## Go Module State

| Dependency | Version | Note |
|------------|---------|------|
| `google.golang.org/grpc` | v1.68.1 | **DOWNGRADED** from v1.77.0 — needs fix |
| `gocloud.dev` | v0.40.0 | **DOWNGRADED** from v0.45.0 — investigate removal |
| `go.opentelemetry.io/contrib/.../otelgrpc` | v0.58.0 | Updated transitively |
| Go toolchain | 1.26.1 | Current |

## What Is NOT Fucked Up

Despite the issues above, the fundamentals are solid:

- **Architecture:** Clean DI, domain types, repository pattern, type-safe templates
- **Code quality:** 4,300 LOC production, 7,400 LOC tests, ~75 linters configured
- **Feature completeness:** Search, sitemap, robots, live reload, diagrams, admonitions, draft filtering, 404 suggestions
- **Infrastructure:** Docker distroless, CI/CD, multi-arch builds
- **Security:** Path traversal prevention, non-root container, rate limiting, request IDs

The project is in good shape structurally. The immediate problems are environmental (disk space) and procedural (bad dependency management in the last commit).
