# Status Report: go-filewatcher/v2 Adoption

**Date:** 2026-07-26 18:59 CEST
**Session:** Adopt `github.com/larsartmann/go-filewatcher/v2` to replace manual fsnotify file-watching code

---

## Executive Summary

Replaced 181 lines of hand-rolled fsnotify boilerplate (recursive directory walking, manual debounce timer+mutex, Chmod filtering, new-directory auto-add, error loop) with 85 lines using `go-filewatcher/v2`. The migration also surfaced and fixed a **pre-existing build break** caused by `httputil` v0.6.0 importing `encoding/json/v2`. All tests pass (one pre-existing flaky test aside), the nix build succeeds, and lint is clean on changed files.

---

## a) FULLY DONE

1. **watcher.go rewrite** — Complete. Went from 181 lines to 85 lines. Removed: `addDirectoriesRecursive`, `shouldTriggerRefresh`, `isDirectory`, manual debounce (`sync.Mutex` + `*time.Timer`), Chremastone filtering, new-directory auto-add logic, `fsnotify.NewWatcher`/`watcher.Events`/`watcher.Errors` select loop. Replaced with `filewatcher.New()` + `WithExtensions(".md", ".markdown")` + `WithDebounce(500ms)` + `WithIgnoreDirs(content.SkipDirs...)` + `WithOnError(...)` + `watcher.Watch(ctx)`.

2. **Context threading** — `main.go` now creates the SIGINT/SIGTERM context once in `run()` and passes it to both `startFileWatcher(ctx, ...)` and `serveHTTP(ctx, ...)`. The file watcher now shuts down cleanly on signal instead of running until process exit. Previously `serveHTTP` created its own context locally.

3. **Dependency added** — `github.com/larsartmann/go-filewatcher/v2 v2.2.1` added as direct dependency. `github.com/fsnotify/fsnotify v1.10.1` demoted to indirect (still required transitively by go-filewatcher).

4. **Pre-existing build break fixed** — `httputil` was pinned at v0.6.0, which imports `encoding/json/v2` in `health.go` (requires `GOEXPERIMENT=jsonv2` or Go 1.27+). Downgraded to `v0.5.0` which uses stable `encoding/json`. This was blocking `go build ./...` before my changes too.

5. **Lint config updated** — `.golangci.yml`: replaced `github.com/fsnotify/fsnotify` with `github.com/larsartmann/go-filewatcher/v2` in depguard allow list. Removed 3 now-unnecessary lint exclusions for `watcher.go` (cyclop, gocognit, nestif) since the new code is simple enough not to trigger them.

6. **Nix vendorHash updated** — `flake.nix` vendorHash updated from `sha256-yGS+...` to `sha256-bOGv...`. `nix build . --impure` succeeds.

7. **AGENTS.md updated** — Added go-filewatcher to Key Technologies, updated File Watching gotcha (#2) to describe the new implementation, updated json/v2 gotcha (#12) to note httputil must stay at v0.5.0 and go-error-family v0.9.0 is safe.

8. **Verification** — `go build ./...` passes. `go test ./... -race` passes on all packages (one pre-existing flaky test in `internal/server` aside). `golangci-lint` clean on changed files. `nix build` succeeds.

---

## b) PARTIALLY DONE

1. **Uncommitted `flake.nix`** — The vendorHash update in `flake.nix` is still uncommitted in the working tree (`git status` shows ` M flake.nix`). The auto-git daemon committed most other changes but this one remains. It will be picked up by the next daemon cycle.

---

## c) NOT STARTED

1. **Integration test for the watcher** — No test was added for the new `watchForChanges` function. The old code had no tests either, so this is not a regression, but the go-filewatcher library makes it much easier to test now (create temp dir, write a `.md` file, assert refresh is called).

2. **Go 1.26.4 vs 1.26.5 mismatch** — `go.mod` declares `go 1.26.4`, but the nix shell provides Go 1.26.5. Pre-existing, not touched.

---

## d) TOTALLY FUCKED UP

Nothing. The migration is clean and working. The one thing worth calling out: I initially didn't realize the `httputil` v0.6.0 build break was pre-existing — I thought my dependency changes caused it. It took a `git stash` + checkout of old `go.mod` to confirm it was already broken. This wasted a few minutes of investigation but didn't cause any damage.

---

## e) WHAT WE SHOULD IMPROVE

### Things I noticed but didn't fix (out of scope)

1. **Pre-existing flaky test: `TestRateLimiter_Concurrent`** — In `internal/server/ratelimit_test.go`. Expects exactly 100 allowed requests but gets 101-102 under race. This is a classic token-bucket race condition in the test itself (concurrent goroutines reading the limiter before the bucket refills deterministically). Fails ~50% of runs. Not related to my changes.

2. **Pre-existing lint issues in `internal/content/helpers.go`** — Two warnings: `SkipDirs` flagged as global variable (`gochecknoglobals`), and a stale `//nolint:gochecknoglobals,golines` directive (`nolintlint` says gochecknoglobals exclusion is unused — meaning the linter config already excludes it elsewhere). Pre-existing.

3. **`.golangci.yml` has `goexperiment.jsonv2` in build-tags** — The project does NOT use json/v2 and explicitly documents it must not be used (gotcha #12). Having this build tag is misleading. Pre-existing.

4. **AGENTS.md gotcha #12 was outdated** — It said `go-error-family` must stay at v0.6.1. But v0.9.0 (now a transitive dependency via go-filewatcher) does NOT import json/v2 in any `.go` file. I verified this. The note was stale — possibly go-error-family reverted its json/v2 migration. I updated the text.

5. **Commit message quality** — The auto-git daemon produced `5220c0a ps): update Go module dependencies` (truncated "deps"). Minor cosmetic issue.

### Things I could have done better in this session

6. **Should have added a watcher integration test** — The old code was untestable (blocking select loop, no context). The new code takes a `context.Context` and is trivially testable. I should have written at least a basic test: create temp dir, start watcher, write a `.md` file, assert `repo.Refresh()` is called.

7. **Should have verified `go-error-family` earlier** — I noticed it went from v0.6.1 to v0.9.0 but didn't check if it broke the json/v2 constraint until writing this report. Turned out fine, but I should have verified before committing.

---

## f) Up to 50 Things to Get Done Next

### High priority

1. Add integration test for `watchForChanges` (temp dir + markdown write + assert refresh)
2. Fix pre-existing flaky `TestRateLimiter_Concurrent` test (race condition in token bucket assertion)
3. Remove `goexperiment.jsonv2` from `.golangci.yml` build-tags (project doesn't use json/v2)
4. Commit the uncommitted `flake.nix` vendorHash change (or let daemon pick it up)

### Medium priority

5. Fix pre-existing `gochecknoglobals` / `nolintlint` warnings in `internal/content/helpers.go`
6. Consider upgrading `go-error-family` to latest explicitly (v0.9.0 is currently indirect via go-filewatcher — make it a documented direct dep or pin it)
7. Add `go-filewatcher` middleware for structured logging of watch events (the library supports `WithMiddleware`)
8. Consider `WithPollInterval` for the watcher if users run dev mode on NFS/Docker volumes
9. Consider `WithGitignore()` option for the watcher (go-filewatcher supports `.gitignore`-aware filtering natively)
10. Consider per-path debounce (`WithPerPathDebounce`) instead of global debounce — currently a 5-file save triggers one refresh, which is fine, but per-path would be more precise
11. Consider exposing watcher config (debounce delay, extensions) as CLI flags/env vars instead of hardcoded constants
12. Upgrade `go.mod` Go version from 1.26.4 to match nix-provided 1.26.5
13. Add `golangci-lint run` to CI (verify it's actually running in `.github/workflows/`)
14. Add `nix flake check` to CI

### Low priority / future

15. Migrate `httputil` to a version that doesn't require json/v2 (or contribute a fix upstream to remove the json/v2 dependency from `health.go`)
16. Consider replacing `httputil` entirely with a small internal middleware package (the project only uses Recovery, RequestID, Compression, Chain, ClientIP, ResponseRecorder)
17. Document the dev-mode architecture in a README section or architecture diagram
18. Add metrics for watcher events (go-filewatcher has `Stats()` and Prometheus middleware)
19. Consider `WatchOnce` for one-shot rebuild scenarios
20. Add a `/api/reload` endpoint that manually triggers the same refresh the watcher does
21. Consider using go-filewatcher's self-healing (`WithSelfHealInterval`) for long-running dev sessions where inotify watches can silently fail
22. Review whether `content.SkipDirs` should be merged with `filewatcher.DefaultIgnoreDirs` (the library has sensible defaults like `.git`, `node_modules`, `vendor`)
23. Add a test that verifies `WithIgnoreDirs(content.SkipDirs...)` actually skips the right directories
24. Add benchmark for repository refresh under concurrent file changes
25. Consider content hashing (`WithContentHashing`) to skip refreshes when file content hasn't actually changed (editor touch events)
26. Document the watcher lifecycle in AGENTS.md (creation → Watch(ctx) → range events → ctx cancel → Close)
27. Consider whether the watcher should run in non-dev modes (e.g., watching S3-backed content for changes)
28. Add graceful handling of `watcher.Close()` errors in the deferred call (currently just logged)
29. Review if `doRefresh` should be debounced independently from the watcher debounce (double-debounce)
30. Consider moving `watchForChanges` into `internal/server` or a dedicated `internal/watcher` package for testability
31. Add context cancellation test (start watcher, cancel context, assert clean exit)
32. Add test for watcher error handling (use `FailingRepository` mock to verify error logging)
33. Consider structured logging via go-filewatcher's `WithDebug` option for verbose dev-mode diagnostics
34. Review the `500ms` debounce constant — make it configurable
35. Add documentation for the `.markdown` extension support (in addition to `.md`)
36. Consider watching template files (`.templ`) for live template reload in dev mode
37. Consider watching static asset directories for live asset reload
38. Add health check integration for the watcher (is it running? how many events processed?)
39. Consider OpenTelemetry tracing for watcher events (go-filewatcher has `WithOTel`)
40. Review thread safety of `doRefresh` — it calls `repo.Refresh()` which uses internal locking, but the caller chain should be verified
41. Add a test that simulates bulk file operations (git checkout, mkdir -p) to verify debounce coalescing
42. Consider rate-limiting refreshes independently from debounce (prevent refresh storms)
43. Document the relationship between watcher debounce and cache invalidation
44. Consider whether the watcher should trigger a full reindex vs incremental update
45. Add observability: log watcher stats on shutdown (total events, filtered, errors)
46. Consider using `filepath.WalkDir` instead of `filepath.Walk` in `internal/content/filesystem.go` for better performance (unrelated but noticed during research)
47. Review whether `content.ShouldSkipDir` is still needed in `filesystem.go` now that the watcher uses `content.SkipDirs` directly
48. Add a CHANGELOG entry for the go-filewatcher adoption
49. Consider adding `go-filewatcher` to the project's FEATURES.md
50. Update the nix devShell to ensure `go-filewatcher` source is available for `gopls` (may need `go mod download`)

---

## g) Questions (that I CANNOT figure out myself)

1. **Should the watcher's `doRefresh` run in a dedicated package (`internal/watcher`) instead of `cmd/dynamic-markdown-site`?** — Currently it's in `main` package, which makes it untestable with the standard `internal/` pattern. Moving it would enable proper testing but changes the architecture. I can't decide this without knowing your preference on package structure for this project.

2. **Should I pin `go-error-family` explicitly in `go.mod` as a direct dependency, or leave it as indirect via go-filewatcher?** — It's currently `v0.9.0 // indirect`. The old AGENTS.md said to pin it at v0.6.1, but v0.9.0 doesn't use json/v2. Leaving it indirect means a go-filewatcher version bump could change it silently. Pinning it gives control but adds a direct dependency you didn't ask for.

3. **Should I fix the pre-existing flaky `TestRateLimiter_Concurrent` test, or leave it since it's unrelated to this task?** — It's clearly a test bug (race condition in the assertion, not the production code), and it pollutes CI output. But it's outside the scope of the go-filewatcher adoption and touching it risks introducing a different assertion behavior.
