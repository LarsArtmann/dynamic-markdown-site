# Dynamic Markdown Site — Full Status Report

**Generated:** 2026-04-01 15:51 CEST
**Branch:** `master` (1 commit ahead of origin)
**Total Go LOC:** 13,645 lines
**TODO Progress:** 37 done / 109 pending (25% complete)

---

## Executive Summary

The 12-item execution plan from the deep-reflection session is **fully complete**. All code changes are committed and build/test cleanly locally. However, **CI is RED** — the latest push (commit `88e3367`) has **26 lint errors** and a security scan failure. This is the single most urgent issue blocking the project.

| Metric       | Status                                                |
| ------------ | ----------------------------------------------------- |
| Build        | ✅ `go build ./...` passes                            |
| Tests        | ✅ All 9 packages pass, 0 failures                    |
| Vet          | ✅ `go vet ./...` clean                               |
| CI           | 🔴 **FAILING** — 26 lint errors + Trivy security scan |
| Dependabot   | 🔴 1 critical alert (gRPC auth bypass)                |
| Coverage     | 72-100% across tested packages                        |
| Lint (local) | ⚠️ 12 warnings (non-blocking)                         |

---

## A) FULLY DONE

### 12-Item Execution Plan (100% Complete)

| #   | Commit    | What                                                                        |
| --- | --------- | --------------------------------------------------------------------------- |
| 1   | planning  | Reflected on mistakes, created prioritized execution plan                   |
| 2   | `400f046` | Reverted unsafe `.md` fallback hack (type assertion + shadowing)            |
| 3   | `0148795` | Root cause fix: `FileSystemRepository` strips `.md` from URL paths          |
| 4   | `0148795` | Updated `draft_test.go` for clean URLs                                      |
| 5   | `0148795` | Sitemap `.md` stripping (automatic from #3)                                 |
| 6   | `c13707b` | Eliminated `RenderResult`/`RenderedContent` duplication                     |
| 7   | `69f4db8` | AST-based `HasMermaid` detection via `parser.Context`                       |
| 8   | `d4065b2` | Removed 216 lines of dead regex-based diagram code                          |
| 9   | `f19390e` | Fixed `NewRenderedFile` to accept and propagate `hasMermaid`                |
| 10  | `c489007` | Replaced hand-rolled `isDraft` with `gopkg.in/yaml.v3`                      |
| 11  | `9439b33` | Added 9 sitemap test functions + fixed pre-existing sitemap.xml routing bug |
| 12  | pushed    | All commits pushed to `origin/master`                                       |

### Other Completed Work (Pre-Execution-Plan)

- Goldmark admonition extension for GitHub-style alert blocks (`88e3367`)
- Sitemap.xml generation endpoint (`4c21153`)
- Robots.txt serving
- Live reload via SSE in dev mode
- Docker multi-arch builds (amd64 + arm64)
- GitHub Actions CI/CD pipeline
- Type-safe Templ templates
- Security headers middleware
- Request ID middleware + structured logging
- ~100+ test functions parallelized
- 37 TODO_LIST items completed

---

## B) PARTIALLY DONE

### CI Pipeline

- Build step: ✅ passes on latest push
- Test step: ✅ passes (with `-race`)
- Lint step: 🔴 **26 errors** (see Section D)
- Security scan: 🔴 Trivy fails

### Test Coverage

- `internal/cache`: **100%**
- `internal/config`: **90.5%**
- `internal/renderer`: **84.3%**
- `internal/server`: **80.3%**
- `internal/domain`: **75.8%**
- `internal/content`: **72.6%**
- `internal/container`: **0%** (DI wiring, hard to unit test)

### Admonition Extension

- Feature works and is committed
- CSS styling done
- Tests written
- **BUT** CI lint fails on it (golines formatting, revive comments, errcheck, exhaustruct, gochecknoglobals)

---

## C) NOT STARTED

High-value items from TODO_LIST that have zero progress:

1. **Security vulnerability fix** — Dependabot critical alert (gRPC auth bypass)
2. **CI lint fix** — 26 errors blocking green CI
3. **Split large test files** — `handlers_test.go` (914 lines), `search_test.go` (685 lines), `markdown_test.go` (611 lines)
4. **Integration test suite** — no end-to-end tests exist
5. **Graceful shutdown tests** — untested
6. **Rate limiting tests** — untested
7. **RSS/Atom feed generation** — not started
8. **Dark mode / theme toggle** — not started
9. **Print stylesheet** — not started
10. **Code copy button** — not started
11. **ETag/If-None-Match caching** — not started
12. **Gzip/brotli compression** — not started
13. **Prometheus metrics** — not started
14. **pprof profiling endpoint** — not started
15. **Architecture decision records** — not started

---

## D) TOTALLY FUCKED UP

### 🔴 CI IS RED — 26 Lint Errors (Commit `88e3367`)

The latest CI run (`23851819894`) **failed** with 26 lint errors. Build and tests pass, but lint does not.

**Breakdown by linter:**

| Linter             | Count | Files                                                                                        |
| ------------------ | ----- | -------------------------------------------------------------------------------------------- |
| `noctx`            | 7     | `sitemap_test.go` — `NewRequest` instead of `NewRequestWithContext`                          |
| `golines`          | 4     | `admonition_extension.go`, `admonition_extension_test.go`, `diagram_extension.go`, `file.go` |
| `revive`           | 4     | `admonition_extension.go` — missing comments on exports, unused param                        |
| `errcheck`         | 2     | `admonition_extension.go` — unchecked `fmt.Fprintf` returns                                  |
| `exhaustruct`      | 2     | `admonition_extension.go` (ast.BaseBlock), `sitemap.go` (server.URLSet)                      |
| `gochecknoglobals` | 2     | `admonition_extension.go` (alertTitles), `diagram_extension.go` (hasMermaidKey)              |
| `cyclop`           | 1     | `helpers.go` — `getContentType` complexity 11 > 10                                           |
| `goconst`          | 1     | `sitemap_test.go` — `"example.com"` repeated 8 times                                         |
| `funlen`           | 1     | `filesystem_test.go` — `TestFileSystemRepository_GetRaw` too long (171 > 150)                |
| `testifylint`      | 1     | `sitemap_test.go` — float compare should use `InEpsilon`                                     |

**Root cause:** The execution plan commits (`69f4db8`–`9439b33`) and the admonition commit (`88e3367`) were pushed without running `golangci-lint run ./...` locally first. Local lint runs showed only 12 warnings (non-blocking), but CI uses stricter settings or a different linter version.

**Fix strategy:** Each error is trivial. The `noctx` errors are mechanical (add `context.Background()`). The `golines` errors need `golines -w`. The `revive` errors need comments. Total fix time: ~15 minutes.

### 🔴 Security Scan Failure

Trivy vulnerability scanner exits with code 1. The SARIF upload also fails because the file doesn't exist. This is likely caused by the gRPC Dependabot alert (`google.golang.org/grpc` auth bypass).

### ⚠️ 5 Previous CI Failures

Before the current failure, there were 4 consecutive CI failures (`10:59`, `11:03`, `11:04`, `11:41 UTC`) because `markdown.go` referenced `NewAdmonitionExtension` before the file was committed. This was a **self-inflicted wound** — the admonition extension file was untracked locally but `markdown.go` already imported it. Commits were pushed without the file.

**Lesson:** Always `git status` before pushing. Never leave imports to untracked files.

---

## E) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Always lint before pushing** — Run `golangci-lint run ./...` as a pre-push gate. The local lint config should match CI exactly.
2. **Never push with untracked imports** — The 5 CI failures from `NewAdmonitionExtension` being undefined were entirely preventable.
3. **Commit atomic changes** — The admonition CSS and test changes should have been in the same commit as the extension code, not separate.
4. **Stale status reports** — There are 6 status report files in `docs/status/`. They accumulate without cleanup. The previous cleanup (removing 33 files) already happened, but they're growing again.

### Code Improvements

5. **Container package has 0% test coverage** — The DI wiring is critical infrastructure with zero tests.
6. **Large test files need splitting** — `handlers_test.go` (914 lines), `search_test.go` (685 lines), `filesystem_test.go` (688 lines) are unwieldy.
7. **`getContentType` complexity** — Cyclop score 11/10. Should extract branches.
8. **Missing graceful shutdown tests** — Production-critical path with zero coverage.
9. **Missing rate limiter tests** — Another production-critical path untested.
10. **No integration/E2E tests** — Unit tests cover individual packages but not the full HTTP → markdown → HTML pipeline.

---

## F) Top 25 Things to Do Next

Ordered by impact-to-effort ratio. Fix CI first, then improve quality, then add features.

### Tier 1: Fix CI (CRITICAL — do today)

| #   | Task                                                                                                | Effort | Impact              |
| --- | --------------------------------------------------------------------------------------------------- | ------ | ------------------- |
| 1   | Fix 26 CI lint errors (noctx, golines, revive, errcheck, exhaustruct, goconst, funlen, testifylint) | 15 min | Unblocks CI         |
| 2   | Add golangci-lint exclusion rules for intentional globals (`hasMermaidKey`, `alertTitles`)          | 5 min  | Clean lint          |
| 3   | Add `golines` to CI pipeline or pre-push hook to prevent formatting drift                           | 10 min | Prevents recurrence |
| 4   | Fix Dependabot critical alert (`google.golang.org/grpc` auth bypass)                                | 10 min | Security            |
| 5   | Fix Trivy security scan failure in CI                                                               | 10 min | Clean CI            |

### Tier 2: Quality (do this week)

| #   | Task                                                          | Effort | Impact              |
| --- | ------------------------------------------------------------- | ------ | ------------------- |
| 6   | Add integration tests for HTTP → markdown → HTML pipeline     | 2h     | Confidence          |
| 7   | Add graceful shutdown tests                                   | 30 min | Coverage            |
| 8   | Add rate limiter tests                                        | 30 min | Coverage            |
| 9   | Split `handlers_test.go` (914 lines) into focused files       | 30 min | Maintainability     |
| 10  | Split `search_test.go` (685 lines) into focused files         | 30 min | Maintainability     |
| 11  | Increase container package test coverage (currently 0%)       | 1h     | Coverage            |
| 12  | Extract `getContentType` branches to reduce complexity 11→<10 | 15 min | Lint clean          |
| 13  | Add `just lint` command to justfile (matching CI config)      | 10 min | Developer UX        |
| 14  | Add git pre-push hook calling `just pre-push`                 | 10 min | Prevent CI breakage |

### Tier 3: Features (do next sprint)

| #   | Task                                                            | Effort | Impact                |
| --- | --------------------------------------------------------------- | ------ | --------------------- |
| 15  | Dark mode CSS + theme toggle                                    | 2h     | User experience       |
| 16  | Code copy button on code blocks                                 | 1h     | User experience       |
| 17  | RSS/Atom feed generation                                        | 2h     | Content discovery     |
| 18  | Gzip/brotli compression middleware                              | 1h     | Performance           |
| 19  | ETag/If-None-Match support                                      | 1h     | Performance           |
| 20  | Print stylesheet                                                | 30 min | User experience       |
| 21  | Prometheus metrics endpoint                                     | 2h     | Observability         |
| 22  | Architecture decision records                                   | 1h     | Documentation         |
| 23  | CONTRIBUTING.md                                                 | 30 min | Open source readiness |
| 24  | Sample markdown content in `content/` directory                 | 30 min | Demo readiness        |
| 25  | Separate CI workflows: `test.yml` (fast) + `docker.yml` (build) | 1h     | CI speed              |

---

## G) Top #1 Question I Cannot Answer Myself

**Should this project stay as a single Go binary, or should it be split into a library + binary?**

Arguments for splitting:

- `internal/renderer` with its Goldmark extensions (diagrams, admonitions, syntax highlighting) is reusable as a standalone markdown rendering library
- `internal/content` repository pattern could serve other frontends
- Would enable `go get` for just the rendering pipeline

Arguments against:

- Current architecture is clean and simple
- Only one consumer exists
- Splitting adds release/versioning complexity
- YAGNI

This is a product/architecture decision that depends on your roadmap. If you plan to use the renderer in other projects, split now before the API ossifies. If this stays a single-purpose tool, keep it simple.

---

## Appendix: CI Error Reference

All 26 lint errors from the latest failing run (`23851819894`):

```
# noctx (7) — use NewRequestWithContext
sitemap_test.go:25  NewRequest → NewRequestWithContext
sitemap_test.go:56  NewRequest → NewRequestWithContext
sitemap_test.go:103 NewRequest → NewRequestWithContext
sitemap_test.go:126 NewRequest → NewRequestWithContext
sitemap_test.go:160 NewRequest → NewRequestWithContext
sitemap_test.go:192 NewRequest → NewRequestWithContext
sitemap_test.go:207 NewRequest → NewRequestWithContext
sitemap_test.go:276 NewRequest → NewRequestWithContext

# golines (4) — line length formatting
file.go:130
admonition_extension.go:21
admonition_extension_test.go:18
diagram_extension.go:71

# revive (4) — comments + unused param
admonition_extension.go:233 exported type needs comment
admonition_extension.go:235 exported func needs comment
admonition_extension.go:239 exported method needs comment
admonition_extension.go:257 unused parameter 'source'

# errcheck (2) — unchecked error returns
admonition_extension.go:274 fmt.Fprintf return not checked
admonition_extension.go:275 fmt.Fprintf return not checked

# exhaustruct (2) — missing struct fields
admonition_extension.go:114 ast.BaseBlock missing BaseNode
sitemap.go:37 server.URLSet missing XMLName

# gochecknoglobals (2) — global variables
admonition_extension.go:29 alertTitles
diagram_extension.go:55 hasMermaidKey

# cyclop (1) — complexity
helpers.go:52 getContentType complexity 11 > 10

# goconst (1) — repeated string
sitemap_test.go:26 "example.com" has 8 occurrences

# funlen (1) — function length
filesystem_test.go:513 TestFileSystemRepository_GetRaw length 171 > 150

# testifylint (1) — float comparison
sitemap_test.go:244 use assert.InEpsilon instead of direct float compare
```

---

## Test Coverage by Package

| Package                     | Coverage | Lines of Code |
| --------------------------- | -------- | ------------- |
| `internal/cache`            | 100.0%   | ~200          |
| `internal/config`           | 90.5%    | ~300          |
| `internal/renderer`         | 84.3%    | ~600          |
| `internal/server`           | 80.3%    | ~1000         |
| `internal/domain`           | 75.8%    | ~400          |
| `internal/content`          | 72.6%    | ~1200         |
| `internal/container`        | 0.0%     | ~200          |
| `cmd/dynamic-markdown-site` | N/A      | ~200          |
| `internal/testutil`         | N/A      | ~100          |
| `internal/version`          | N/A      | ~20           |

---

## Git State

```
HEAD: 819b19c (local only, not pushed)
origin/master: 88e3367

Untracked:
  docs/status/2026-04-01_15-45_admonition-extension-project-health.md

Unpushed commits:
  819b19c docs(status): add raw asset serving implementation report and comprehensive status
```

Note: The local-only commit `819b19c` is a docs-only status report. Not critical to push.
