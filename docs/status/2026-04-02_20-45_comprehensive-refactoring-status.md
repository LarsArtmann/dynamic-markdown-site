# Dynamic Markdown Site — Comprehensive Status Report

**Date:** 2026-04-02 20:45\
**Author:** Crush (GLM-5.1)\
**Session:** Continuation from interrupted session (previous context provided by user)

---

## Executive Summary

Significant refactoring work completed across 2 sessions. CI pipeline fixed, code deduplicated, ContentTree optimized from O(n) to O(1), and compile-time safety added. Build compiles cleanly. Tests were running at time of report.

**TL;DR:** 6 of 8 planned refactoring tasks DONE. 2 remain (lint verification, push). Build passes. All committed work is 1 push away from remote.

---

## A) FULLY DONE ✅

### 1. CI Pipeline Fix (commit `7701c90`)

- Fixed 3 bugs in `.github/workflows/docker.yml`:
  - **ghcr.io case sensitivity** — `github.repository` preserves original case but container registries require lowercase. Fixed with `tr '[:upper:]' '[:lower:]'`
  - **Digest-based security scan** — tag format mismatch caused scan failures. Switched to digest-based reference
  - **PR guard on scan** — security-scan was running on PRs where images aren't pushed. Added `if: github.event_name != 'pull_request'`

### 2. Export Content Helpers (commit `6df056b` + current session)

- `shouldSkipDir` → `ShouldSkipDir` (exported)
- `isMarkdownFile` → `IsMarkdownFile` (exported)
- `getContentType` → `GetContentType` (exported)
- `skipDirs` → `SkipDirs` (exported)
- `contentTypes` → `ContentTypes` (exported)
- Added font MIME types (`.woff`, `.woff2`) to the shared ContentTypes map
- All internal callers in `filesystem.go` and `blob.go` updated

### 3. Deduplicate `getContentType` in `server/static.go` (commit `6df056b`)

- Removed 20-line switch-based `getContentType` from server package
- Replaced with `staticContentType()` wrapper that delegates to `content.GetContentType()`
- Handles semantic difference: server returns `""` for unknown types, content returns `"application/octet-stream"`

### 4. Deduplicate `skipDirs` in `cmd/watcher.go` (current session, uncommitted)

- Removed local `skipDirs := []string{...}` duplicate
- Replaced with `content.ShouldSkipDir(filepath.Base(path))`
- Also replaced `shouldTriggerRefresh` with `content.IsMarkdownFile(path)`

### 5. Optimize ContentTree — O(n) → O(1) lookups (current session, uncommitted)

- Added `paths map[URLPath]ContentNode` field to `ContentTree`
- New `indexNode()` method recursively populates the map at construction time
- `Find()` now does a single map lookup instead of recursive tree traversal
- `AllPaths()` now iterates the map keys instead of recursing the tree
- Removed dead `findInNode()` and `collectPaths()` methods

### 6. Compile-Time Interface Checks (current session, uncommitted)

- Added `var _ Repository = (*FileSystemRepository)(nil)` in `filesystem.go`
- Added `var _ Repository = (*BlobRepository)(nil)` in `blob.go`
- Added `var _ Repository = (*InMemoryRepository)(nil)` in `memory.go`
- These will cause compile errors if any Repository method is missing or has wrong signature

### 7. Test Update (current session, uncommitted)

- Updated `TestGetContentType` in `handlers_test.go` to call `staticContentType` instead of removed `getContentType`

### 8. Build Verification

- `GOWORK=off go build ./...` — **PASSES** (clean exit, zero output)
- Tests were running at time of report

---

## B) PARTIALLY DONE ⚠️

### 1. Test Verification

- Build compiles clean
- `GOWORK=off go test ./... -cover` was started but not confirmed passed at time of report

### 2. Lint Verification

- `GOWORK=off golangci-lint run ./...` — **NOT RUN** this session
- Parallel golangci-lint runner conflict from IDE was blocking diagnostics
- The `staticContentType` function may trigger `unparam` or similar linters — needs verification

---

## C) NOT STARTED ❌

### From Original Plan (Not Yet Addressed)

1. **Remove dead `treeStats.addError` method** — needs verification if it exists and is actually dead
2. **Apply staticcheck tagged switch suggestion** in `internal/server/errors.go` — the `renderComponent` switch could use tagged switch syntax
3. **Container package tests** — `internal/container` has 0.0% test coverage
4. **Go version mismatch fix** — local Go 1.26.0 vs go.mod 1.26.1 causes ~50 warnings per test run
5. **go.work integration** — parent `~/projects/go.work` doesn't include this project, requiring `GOWORK=off` for all commands

---

## D) TOTALLY FUCKED UP 💥

### 1. Commit `6df056b` — Previous Session Commits Without Push

- The previous session committed helpers.go and static.go changes in `6df056b`
- This commit is on master but was **NEVER PUSHED** to remote
- Current session has 1 additional unpushed commit (`c1779fd` — status report)
- **Total: 2 unpushed commits on master** + 6 uncommitted modified files
- If this machine dies, all this work is lost

### 2. Duplicate Work — helpers.go/static.go

- The previous session had ALREADY committed the helpers.go and static.go changes
- Current session re-did the same edits (they showed no diff because they were identical)
- No harm done (idempotent), but it means ~10 minutes was wasted re-doing already-done work
- Root cause: the handoff context didn't clearly indicate that `6df056b` included these changes

### 3. Build/Test Speed

- `go build ./...` took several minutes (downloading gocloud.dev, cloud.google.com, etc.)
- This is a transitive dependency issue from `gocloud.dev/blob` — it pulls in the entire Google Cloud SDK
- Not broken, but painful for iteration speed

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Architecture

| Area                     | Issue                                                       | Impact                   | Effort                                       |
| ------------------------ | ----------------------------------------------------------- | ------------------------ | -------------------------------------------- |
| `gocloud.dev` dependency | Pulls in Google Cloud SDK, Azure SDK, AWS SDK               | Slow builds, huge binary | Medium — consider interface + plugin pattern |
| Container tests          | 0.0% coverage on DI container                               | Untested wiring          | Low — straightforward tests                  |
| Content search           | Linear scan of all files                                    | Poor perf on large repos | Medium — add inverted index                  |
| Error context            | `cockroachdb/errors` stack traces verbose in HTTP responses | Leaks internals          | Low — sanitize before sending                |
| Config validation        | No validation on flag values                                | Silent misconfiguration  | Low                                          |

### Code Quality

| Area                     | Issue                                               | Fix                        |
| ------------------------ | --------------------------------------------------- | -------------------------- |
| Domain tree immutability | `SetChildren()` breaks immutability contract        | Remove or make private     |
| `filterEmptyDirectories` | Returns `bool` AND mutates — confusing              | Split into query + command |
| `shouldComeAfter` sort   | Unexported, untested                                | Export and test            |
| Watcher debouncing       | Comment says "simplified version"                   | Implement proper debounce  |
| `scheduleRefresh`        | No actual debouncing — every event triggers refresh | Add time-based debounce    |

### DevEx

| Area                             | Issue                           | Fix                                                     |
| -------------------------------- | ------------------------------- | ------------------------------------------------------- |
| `GOWORK=off` required            | Parent go.work interferes       | Add project to go.work or use directory-specific config |
| Go 1.26.0 vs 1.26.1              | Noisy warnings on every build   | Upgrade local Go                                        |
| No Makefile/Justfile integration | Commands scattered across docs  | Already has justfile — document it                      |
| No hot-reload for templates      | Need `templ generate` + restart | Add air or similar                                      |

---

## F) Top 25 Things We Should Get Done Next

### Priority 1 — Ship What We Have (1-3)

1. **Commit all uncommitted refactoring changes** (6 files modified)
2. **Push everything to remote** — 2 unpushed commits + new refactoring commit
3. **Verify CI passes on GitHub Actions** — confirm the CI fix actually works

### Priority 2 — Safety & Correctness (4-8)

4. **Run full lint suite** — `GOWORK=off golangci-lint run ./...` and fix any issues
5. **Add container package tests** — `internal/container` at 0.0% coverage
6. **Verify all tests pass** — confirm the tree optimization didn't break anything
7. **Add ContentTree benchmarks** — verify O(1) improvement with `BenchmarkFind`
8. **Check for dead code** — `treeStats.addError`, unused error types

### Priority 3 — DevEx & DX (9-13)

9. **Fix Go version mismatch** — upgrade local Go to 1.26.1
10. **Fix go.work interference** — add project to `~/projects/go.work` or use `GOWORK=off` wrapper
11. **Add pre-commit hook** — run `go test` and `golangci-lint` before push
12. **Add `.editorconfig`** — consistent formatting across editors
13. **Document the justfile commands** — add to README or AGENTS.md

### Priority 4 — Performance (14-18)

14. **Implement watcher debouncing** — actual time-based debounce, not "simplified version"
15. **Add inverted index for search** — replace linear scan in `internal/content/search.go`
16. **Cache rendered HTML** — currently caches raw markdown lookups; render is per-request
17. **Benchmark the full request pipeline** — establish performance baselines
18. **Profile memory usage** — ContentTree path map trades memory for speed

### Priority 5 — Features & Polish (19-25)

19. **Add ETag/If-None-Match support** — HTTP caching headers for content
20. **Add RSS/Atom feed** — auto-generate from content tree
21. **Add sitemap.xml generation** — SEO
22. **Implement proper 404 page** — with path suggestions (already exists but needs styling)
23. **Add OpenGraph meta tags** — social sharing previews
24. **Add table-of-contents sidebar** — already extracted TOC data, just needs template work
25. **Add dark mode toggle** — CSS variables approach, persist in localStorage

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should the `gocloud.dev` dependency be kept or replaced?**

The `gocloud.dev/blob` dependency in `internal/content/blob.go` pulls in the entire Google Cloud SDK (~200MB of transitive dependencies). This causes:

- Slow builds (3-5 minutes for cold `go build`)
- Huge binary sizes
- Unnecessary dependencies for users who only use the filesystem backend

**The question:** Is blob storage (S3/GCS/Azure) actually a planned feature, or was it an exploration? Options:

1. **Keep it** — it works, supports multiple backends, the interface is clean
2. **Move to separate module** — `internal/content/blob` becomes its own Go module with `gocloud.dev` as an optional dependency
3. **Remove it** — if no one is using cloud storage, the cost isn't worth it

This is a product/architecture decision that only the project owner can make.

---

## Git State at Time of Report

```
Current branch: master
Ahead of origin/master by: 2 commits (6df056b, c1779fd)
Uncommitted changes: 6 files modified (watcher, blob, filesystem, memory, tree, handlers_test)
Build status: PASSES (clean)
Test status: RUNNING (not confirmed at time of report)
Lint status: NOT RUN
```

### Modified Files (uncommitted)

| File                                   | Change                                                | Lines   |
| -------------------------------------- | ----------------------------------------------------- | ------- |
| `cmd/dynamic-markdown-site/watcher.go` | Use `content.ShouldSkipDir`, `content.IsMarkdownFile` | -13/+6  |
| `internal/content/blob.go`             | Add `var _ Repository = (*BlobRepository)(nil)`       | +2      |
| `internal/content/filesystem.go`       | Add `var _ Repository = (*FileSystemRepository)(nil)` | +2      |
| `internal/content/memory.go`           | Add `var _ Repository = (*InMemoryRepository)(nil)`   | +2      |
| `internal/domain/tree.go`              | O(1) path index, remove recursive methods             | -19/+38 |
| `internal/server/handlers_test.go`     | Rename test to use `staticContentType`                | -2/+2   |

---

_Generated by Crush (GLM-5.1) — 2026-04-02T20:45:24_
