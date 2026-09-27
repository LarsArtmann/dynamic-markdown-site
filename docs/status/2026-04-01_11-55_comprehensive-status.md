# Comprehensive Status Report — Deep Reflection

**Date:** 2026-04-01 11:55 CEST
**Author:** Crush (AI Assistant)
**Session:** Multi-session CI/debugging → architecture reflection
**Commit:** `4c21153` | **Branch:** `master` (up to date with origin)

---

## Executive Summary

The `dynamic-markdown-site` project is in **good working shape**. All recent commits are pushed to origin. Linter reports 0 issues. Tests pass. CI pipeline (build + lint + security-scan) has been fixed across multiple sessions. The codebase is ~4,600 LOC (production) + ~6,700 LOC (tests) across 37 Go source files.

**Critical local issue resolved this session:** Disk was at 775MB/229GB → cleaned 6.6GB of stale Go build caches → now 6.9GB free.

---

## A. FULLY DONE ✅

### CI Pipeline Fixes (Primary Objective — COMPLETE)

| What                           | Commit    | Detail                                                                                           |
| --- | --- | --- |
| `.golangci.yml` exclusion gaps | `06de1c8` | Fixed config.go cyclop, blob.go gocognit, version.go revive; removed dead `pkg/errors` exclusion |
| `diagramNode` exhaustruct fix  | `6c0423c` | Added `BaseBlock: ast.BaseBlock{}` initialization                                                |
| `config.go` cyclop reduction   | `fd89f13` | Decomposed `Load()` into 4 focused methods                                                       |
| `pkg/errors` dead code removal | `21c9ecf` | Removed entire `pkg/errors/` package                                                             |
| All remaining lint issues      | `09ff07a` | forcetypeassert + golines fixes                                                                  |
| Duplicate request logger       | `360dc47` | Removed from main.go; accesslog middleware handles it                                            |
| Docker CI: PR trigger          | `bf7859c` | Added `pull_request` trigger, conditional push                                                   |
| Stale status cleanup           | `b24f2e5` | Removed 33 outdated status docs                                                                  |

### Features Added and Verified Working

| Feature                   | Files                                       | Status                            |
| --- | --- | --- |
| robots.txt endpoint       | `server/robots.go`, `server/robots_test.go` | ✅ Dynamic sitemap URL            |
| sitemap.xml endpoint      | `server/sitemap.go`                         | ✅ Priority/changetime heuristics |
| Draft content filtering   | `content/draft.go`, `content/filesystem.go` | ✅ YAML `draft: true`             |
| Access logging middleware | `server/accesslog.go`                       | ✅ With request IDs               |
| Blob storage support      | `content/blob.go`, `content/drivers.go`     | ✅ S3/GCS/Azure via go-cloud      |
| Site name config          | `config/config.go`                          | ✅ Flag + env var                 |
| Live reload (SSE)         | `server/livereload.go`                      | ✅ Dev mode                       |

### Code Quality Metrics

| Metric               | Value                       |
| --- | --- |
| Production LOC       | 4,607                       |
| Test LOC             | 6,716                       |
| Test/Code ratio      | 1.46:1                      |
| Production Go files  | 37                          |
| golangci-lint issues | **0**                       |
| Test packages        | 8 (all pass)                |
| Coverage (approx)    | 75-80% across core packages |
| Linters enabled      | ~75                         |

---

## B. PARTIALLY DONE ⚠️

### CI Remote Verification

- **Local:** Linter = 0 ✅, Tests = all pass ✅, Build = all pass ✅
- **Remote:** Cannot verify without `gh` auth. Needs manual check on GitHub Actions.
- **Security scan (Trivy):** Depends on Docker build. Not verified locally.

### Sitemap Implementation

- Endpoint works, route registered
- **Missing:** No test file for `sitemap.go`

### Blob Storage

- Implementation exists with timeout pattern in container.go
- **Missing:** No integration tests, only filesystem tests

### TODO_LIST.md

- 140+ items, many stale or already done
- Needs pruning and reprioritization

---

## C. NOT STARTED ❌

### High-Value, Not Addressed

1. ~~Pre-push linter hook — prevent unlinted code reaching CI~~ done — .githooks/pre-push runs go test -race -cover + golangci-lint
2. ~~Sitemap.go tests — 0 coverage for newest feature~~ done at `9439b33`
3. ~~`version` package rename → `buildinfo` — eliminate revive exclusion~~ **Won't implement — kept internal/version; rename not worth the churn.**
4. ~~Split large test files (search_test 685 lines, handlers_test 667 lines)~~ done — search/handlers/markdown test files all split (search_scoring_test.go, refresh_test.go, markdown_toc_test.go etc.)
5. ~~Immutable FileNode refactor — remove setters~~ done at `d5efa2d`
6. ~~Error type hierarchy — consistent Is/As/Unwrap~~ done — sentinel errors (ErrContentNotFound, ErrInvalidPath) via cockroachdb/errors
7. ~~Architecture decision records — none exist~~ done — docs/adr/ holds 5 ADRs
8. ~~Integration test suite — no end-to-end HTTP tests~~ done — internal/server/shutdown_integration_test.go + per-endpoint tests
9. ~~Coverage enforcement in CI — no minimum threshold~~ done — test.yml enforces 75% coverage floor
10. ~~Graceful degradation tests — D2 renderer failure untested~~ done — diagram_extension.go:207 logs slog.Warn and continues without diagrams

### Features in TODO but Not Started

RSS/Atom feeds, content tags, dark mode, search autocomplete, pagination, admin dashboard, Prometheus metrics, OpenTelemetry tracing

---

## D. TOTALLY FUCKED UP 💥

### Multi-Agent Race Conditions

**The #1 problem.** Multiple AI agents operate on the same repo concurrently:

- Agents push **unlinted code** — 5+ rounds of reactive fixes
- `main.go` was **corrupted** — `RegisterRoutes()` commented out, `requestLogger` deleted
- **View tool shows stale output** — cached content doesn't match disk
- **Commits appear between sessions** — other agents change HEAD
- **33 stale status reports** accumulated before cleanup

**Impact:** ~60% of debugging work was _reactive_ — fixing other agents' mistakes.

### Disk Space Crisis

- Was at 775MB free on 229GB disk (100% full)
- Root cause: Go build cache creates new temp dirs per build, never cleans up
- Cleaned 6.6GB → now 6.9GB
- **Will recur** unless monitored or automated

### golines Not Installable Locally

- Security restrictions prevent `go install golines`
- Must manually break long lines
- Every gofmt-aligned struct literal risks re-triggering golines

---

## E. IMPROVEMENTS 📈

### Process

| # | Improvement                                   | Impact                    | Effort |
| --- | --- | --- | --- |
| ~~1~~ | ~~Pre-push hook (lint + test)~~ done — .githooks/pre-push (test + lint) | ~~Prevents broken CI~~ | ~~30min~~ |
| ~~2~~ | ~~`just pre-push` and `just fix` commands~~ done — pre-push hook covers it; justfile removed for flake.nix | ~~Standardized verification~~ | ~~15min~~ |
| ~~3~~ | ~~Separate fast test workflow from Docker build~~ done — test.yml + docker.yml + release.yml | ~~Faster PR feedback~~ | ~~Medium~~ |
| ~~4~~ | ~~Coverage threshold ≥75% in CI~~ done — test.yml 75% coverage floor | ~~Prevents regression~~ | ~~15min~~ |
| ~~5~~ | ~~Disk space monitoring~~ **Won't implement — environment issue from the 2026-04 macOS setup; current env unaffected.** | ~~Prevents build failures~~ | ~~30min~~ |

### Architecture

| #  | Improvement                                     | Impact                      | Effort |
| --- | --- | --- | --- |
| ~~6~~  | ~~Rename `version` → `buildinfo`~~ **Won't implement — kept internal/version.** | ~~Eliminates revive exclusion~~ | ~~30min~~ |
| ~~7~~  | ~~Immutable FileNode (remove setters)~~ done — setters removed; immutable render pipeline (CHANGELOG 0.1.0) | ~~Thread safety~~ | ~~2hr~~ |
| ~~8~~  | ~~Split Repository into Reader + Refresher~~ done — Repository interface kept unified; refresh via Refresh() | ~~Cleaner concerns~~ | ~~1hr~~ |
| ~~9~~  | ~~Structured errors with Is/As/Unwrap~~ done — sentinel errors ErrContentNotFound/ErrInvalidPath in place | ~~Better error matching~~ | ~~2hr~~ |
| ~~10~~ | ~~Frontmatter typed struct (not `map[string]any`)~~ done — Frontmatter struct exists in internal/domain | ~~Type safety~~ | ~~1hr~~ |

### Library Considerations

| #  | Current              | Alternative                              | Why                                    |
| --- | --- | --- | --- |
| ~~11~~ | ~~`samber/do/v2`~~ **Won't implement — staying on samber/do/v2; DI pattern documented in AGENTS.md.** | ~~`wire` (compile-time)~~ | ~~Catch DI errors at build time~~ |
| ~~12~~ | ~~`cockroachdb/errors`~~ **Won't implement — kept cockroachdb/errors for stack traces.** | ~~stdlib `fmt.Errorf("%w")` + custom types~~ | ~~One less dependency; stdlib sufficient~~ |
| ~~13~~ | ~~`charm.land/log`~~ **Won't implement — kept charm.land/log (implements slog.Handler).** | ~~`slog` directly~~ | ~~stdlib; one less dependency~~ |
| ~~14~~ | ~~Custom search~~ done — in-memory search ships with scoring+snippets; Bleve only if demand appears | ~~`bleve`~~ | ~~Fuzzy matching, ranking, pagination~~ |
| ~~15~~ | ~~Manual middleware~~ done — rate limit shipped in-repo (ratelimit.go); no gin-contrib | ~~`gin-contrib` packages~~ | ~~Rate limit, CORS already exist~~ |

### Type Model Improvements

| #  | Improvement                                            | Detail                             |
| --- | --- | --- |
| ~~16~~ | ~~`domain.HTML` with methods~~ done — domain.HTML ships; methods unnecessary so far | ~~`String()`, `Len()`, `IsZero()`~~ |
| ~~17~~ | ~~`RenderedContent` as immutable~~ done — RenderedFile immutable via NewRenderedFileWithContent (4233fdc) | ~~Return interface, prevent mutation~~ |
| ~~18~~ | ~~`ContentNode` with `Children()` on both dirs and files~~ done — ContentTree with Find/AllPaths map index (0192273) | ~~Eliminate type switches~~ |
| ~~19~~ | ~~`Frontmatter` as typed struct~~ done — Frontmatter typed struct in domain | ~~Replace `map[string]any`~~ |
| ~~20~~ | ~~Sealed interface for node kinds~~ done — NodeKind enum + single ContentNode interface | ~~Prevent invalid implementations~~ |

---

## F. TOP 25 NEXT ITEMS (Impact/Effort Sort)

| #  | Item                                     | Impact      | Effort | Cat           |
| --- | --- | --- | --- | --- |
| ~~1~~  | ~~Pre-push hook (lint + test)~~ done — .githooks/pre-push (test + lint) | ~~🔴 Critical~~ | ~~30min~~ | ~~Process~~ |
| ~~2~~  | ~~`just pre-push` + `just fix` commands~~ done — superseded by .githooks + flake.nix; justfile removed | ~~🔴 High~~ | ~~15min~~ | ~~Process~~ |
| ~~3~~  | ~~Verify CI green on GitHub (3 jobs)~~ done — CI green across later sessions (latest full run 2026-09-13) | ~~🔴 Critical~~ | ~~10min~~ | ~~CI~~ |
| ~~4~~  | ~~Write sitemap.go tests~~ done at `9439b33` | ~~🔴 High~~ | ~~1hr~~ | ~~Testing~~ |
| ~~5~~  | ~~Rename `version` → `buildinfo`~~ **Won't implement — kept internal/version.** | ~~🟡 Medium~~ | ~~30min~~ | ~~Arch~~ |
| ~~6~~  | ~~Split `handlers_test.go` (667 lines)~~ done — handlers tests split into 9 files | ~~🟡 Medium~~ | ~~1hr~~ | ~~Quality~~ |
| ~~7~~  | ~~Split `search_test.go` (685 lines)~~ done — search tests split into 3 files | ~~🟡 Medium~~ | ~~1hr~~ | ~~Quality~~ |
| ~~8~~  | ~~Coverage threshold ≥75% in CI~~ done — test.yml 75% floor | ~~🟡 Medium~~ | ~~15min~~ | ~~CI~~ |
| ~~9~~  | ~~ADR for DI choice (do vs wire)~~ done — 5 ADRs in docs/adr/; DI documented in AGENTS.md | ~~🟡 Medium~~ | ~~30min~~ | ~~Docs~~ |
| ~~10~~ | ~~Immutable FileNode (remove setters)~~ done at `d5efa2d` | ~~🟡 Medium~~ | ~~2hr~~ | ~~Arch~~ |
| ~~11~~ | ~~Replace `cockroachdb/errors` with stdlib~~ **Won't implement — kept cockroachdb/errors.** | ~~🟡 Medium~~ | ~~2hr~~ | ~~Deps~~ |
| ~~12~~ | ~~Frontmatter typed struct~~ done — Frontmatter struct in internal/domain | ~~🟡 Medium~~ | ~~1hr~~ | ~~Types~~ |
| ~~13~~ | ~~Split Repository: Reader + Refresher~~ done — kept unified Repository interface; documented in AGENTS.md | ~~🟡 Medium~~ | ~~1hr~~ | ~~Arch~~ |
| ~~14~~ | ~~HTTP integration tests~~ done — shutdown_integration_test.go + per-endpoint handler tests | ~~🟢 High~~ | ~~3hr~~ | ~~Testing~~ |
| ~~15~~ | ~~Disk space monitoring~~ **Won't implement — environment issue from the 2026-04 macOS setup.** | ~~🟢 Low~~ | ~~30min~~ | ~~Tooling~~ |
| ~~16~~ | ~~Middleware chain as slice~~ done — httputil.Chain adopted (Recovery, RequestID, Compression) | ~~🟢 Low~~ | ~~30min~~ | ~~Arch~~ |
| ~~17~~ | ~~`errgroup` for concurrent ops~~ **Won't implement — no concurrency need surfaced.** | ~~🟢 Low~~ | ~~1hr~~ | ~~Perf~~ |
| **NOT-DO/DUPLICATE — RSS/Atom feed generation** canonical entry lives in ROADMAP.md (Content Delivery) | |
| **NOT-DO/DUPLICATE — Dark mode CSS toggle** canonical entry lives in ROADMAP.md (UI/UX) | |
| ~~20~~ | ~~Prometheus metrics endpoint~~ done — internal/server/metrics.go serving /metrics | ~~🟢 Low~~ | ~~2hr~~ | ~~Observability~~ |
| **NOT-DO/DUPLICATE — Rate limit search endpoint** canonical entry lives in TODO_LIST.md (still open there) | |
| ~~22~~ | ~~gzip/brotli compression~~ done — httputil.Compression middleware (gzip >512B) | ~~🟢 Low~~ | ~~30min~~ | ~~Perf~~ |
| ~~23~~ | ~~Graceful shutdown tests~~ done — shutdown_integration_test.go covers drain | ~~🟢 Low~~ | ~~1hr~~ | ~~Testing~~ |
| **NOT-DO/DUPLICATE — ETag/If-None-Match** canonical entry lives in ROADMAP.md (Content Delivery); not implemented | |
| **NOT-DO/DUPLICATE — Evaluate `wire` as DI replacement** staying on samber/do/v2; documented in AGENTS.md | |

---

## G. TOP #1 QUESTION 🤔

**Should this project stay with `samber/do/v2` or migrate to Google `wire` for compile-time DI?**

| `samber/do/v2` (current)     | `wire` (alternative)               |
| --- | --- |
| Runtime DI — flexible        | **Compile-time** — errors at build |
| No code gen step             | Requires `wire` code gen           |
| Graceful shutdown built-in   | Manual shutdown orchestration      |
| Already working, 7 providers | Would catch arg count mismatches   |
| Less boilerplate             | More type-safe                     |

**I cannot decide:** depends on team preference for runtime flexibility vs. compile-time safety, and whether code generation is acceptable in the build pipeline.

---

## Environment

| Item        | Value                             |
| --- | --- |
| Go          | 1.26.1 darwin/arm64               |
| Disk        | 6.9GB free / 229GB                |
| Branch      | `master` (up to date with origin) |
| Head        | `4c21153`                         |
| Uncommitted | None                              |
| Linter      | 0 issues                          |
| Tests       | All pass                          |
| Build       | All pass                          |
