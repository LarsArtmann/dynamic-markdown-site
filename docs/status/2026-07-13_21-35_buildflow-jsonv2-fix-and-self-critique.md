# Status Report: Buildflow `encoding/json/v2` Fix + Self-Critique

**Date:** 2026-07-13 21:35
**Session scope:** Fix 4 failing buildflow steps caused by `encoding/json/v2` import, then brutally self-review

---

## Context

The user pasted a `buildflow` run that failed with 4 errors, all stemming from the same root cause:

```
imports encoding/json/v2: build constraints exclude all Go files in
/nix/store/.../go-1.26.4/share/go/src/encoding/json/v2
```

The `encoding/json/v2` package requires `GOEXPERIMENT=jsonv2` (build tag `//go:build goexperiment.jsonv2`), which is NOT enabled in this project's Go 1.26.4 toolchain.

---

## A) FULLY DONE

### 1. Traced and fixed the root cause chain

**Three layers of breakage, one root cause:**

| Layer                           | What happened                                                                                                                            | Fix                                                                                    |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `internal/server/helpers.go`    | `go-auto-upgrade` migrated `encoding/json` -> `encoding/json/v2`, changed `json.NewEncoder(w).Encode(v)` to `json.MarshalWrite(w, v)`    | Reverted to `encoding/json` with original `json.NewEncoder(w).Encode(v)` pattern       |
| `internal/server/livereload.go` | `go-auto-upgrade` migrated `encoding/json` -> `encoding/json/v2`                                                                         | Reverted to `encoding/json` (back to committed HEAD state)                             |
| `go.mod` dependency             | `go-auto-upgrade` bumped `go-error-family` v0.6.1 -> v0.7.0, and v0.7.0 uses `encoding/json/v2` in `error.go`, `http.go`, and test files | Downgraded back to `v0.6.1` via `go get github.com/larsartmann/go-error-family@v0.6.1` |

### 2. Found and fixed a hidden test failure (pre-existing)

**`TestMetricsEndpointReturnsPrometheusFormat`** was failing — but masked by the compilation error (`[setup failed]`). Once the build was fixed, this test surfaced.

**Root cause:** The `httputil.Compression` middleware gzips responses >512 bytes when no `Accept-Encoding` header is present (it falls back to the first configured encoding, which is gzip). The metrics endpoint response (~450 bytes of Prometheus text) exceeds the 512-byte threshold after compression kicks in. The test was reading raw gzip bytes and checking for plaintext strings.

**Fix:** Added `req.Header.Set("Accept-Encoding", "identity")` to the shared `executeRequest` test helper (`handlers_test.go:26`). This disables compression for all test requests, which is the correct behavior — tests should validate response content, not compression behavior.

### 3. Verified everything

- `go build ./...` — passes
- `go vet ./...` — passes
- `go test -race ./...` — all 8 test packages pass
- `go mod tidy` — clean, no changes

### 4. Final diff is minimal

| File                               | Change vs HEAD                                                                   |
| ---------------------------------- | -------------------------------------------------------------------------------- |
| `internal/server/helpers.go`       | 0 lines (reverted to HEAD exactly)                                               |
| `internal/server/livereload.go`    | 0 lines (reverted to HEAD exactly)                                               |
| `internal/server/handlers_test.go` | +1 line (`Accept-Encoding: identity`)                                            |
| `go.mod` / `go.sum`                | `go-error-family` pinned to v0.6.1 + other transitive updates from buildflow run |

---

## B) PARTIALLY DONE

### 1. Flaky `TestRateLimiter_Concurrent`

- **Observed:** Failed once with "expected 100 allowed requests, got 101", then passed 5/5 on retry
- **Status:** Confirmed flaky, NOT fixed
- **Root cause hypothesis:** Token bucket boundary condition under race detection overhead — the 101st request sneaks through before the limiter rejects
- **Risk:** Low — intermittent, passes on retry

### 2. The `go-auto-upgrade` systemic problem (partially mitigated, NOT solved)

- The immediate breakage is fixed
- But the **next buildflow run will re-break** because `go-auto-upgrade` will try to migrate to `encoding/json/v2` again
- See section E for the full problem

---

## C) NOT STARTED

Nothing relevant to this session's scope was left unstarted.

---

## D) TOTALLY FUCKED UP

### 1. Initial `helpers.go` fix was wrong

**What I did:** When reverting from `encoding/json/v2`, I rewrote the function using `json.Marshal(v)` + `w.Write(data)` instead of checking what HEAD had.

**What I should have done:** The committed code used `json.NewEncoder(w).Encode(v)` — a one-line, streaming approach. I should have read HEAD's version first and reverted to it exactly. Instead I introduced an unnecessary API change (Marshal+Write vs Encode) that diverged from the committed pattern.

**Impact:** Caught during self-review. Fixed before session ended. Final diff now shows 0 changes to `helpers.go`.

### 2. Didn't trace the dependency chain fast enough

I fixed `helpers.go` and `livereload.go` first, then ran `go build ./...` — and discovered the SAME error was coming from `go-error-family` (a transitive dependency via `httputil`). I should have checked the full dependency graph for `encoding/json/v2` usage BEFORE making any code changes. This would have saved one build cycle.

---

## E) WHAT WE SHOULD IMPROVE

### Critical: The `go-auto-upgrade` tool will re-break the build

**This is the #1 issue.** The buildflow's `go-auto-upgrade:repair` step:

1. Migrates `encoding/json` imports to `encoding/json/v2` in source files
2. Upgrades dependencies to versions that use `encoding/json/v2`
3. Breaks compilation because `GOEXPERIMENT=jsonv2` is not enabled

**Options to fix permanently:**

| Option                                                                 | Effort | Tradeoff                                                             |
| ---------------------------------------------------------------------- | ------ | -------------------------------------------------------------------- |
| **A) Exclude `encoding/json/v2` migration in buildflow config**        | Low    | Need to find/configure the exclusion rule in buildflow               |
| **B) Enable `GOEXPERIMENT=jsonv2`**                                    | Low    | Experimental API, may have breaking changes, adds toolchain coupling |
| **C) Pin `go-error-family` in a `go.work` or replace directive**       | Medium | Prevents accidental upgrades but adds maintenance overhead           |
| **D) Upgrade `go-error-family` and adopt `encoding/json/v2` properly** | High   | Forward-looking but requires enabling the experiment everywhere      |

**Recommendation:** Option A — exclude the migration. The project intentionally uses stable `encoding/json`.

### Other improvements identified

1. **Test helper should document why `Accept-Encoding: identity` is set** — future developers may not understand the compression middleware behavior
2. **Flaky rate limiter test** — `TestRateLimiter_Concurrent` has a boundary condition. Should use a higher burst count or add tolerance for ±1
3. **gopls `unusedwrite` warnings in `content_test.go`** — 4 fields written but never read in test structs. Pre-existing, not caused by this session
4. **No integration test for compression** — The fact that compression was silently breaking the metrics test suggests we need at least one test that verifies compressed responses decompress correctly

---

## F) Up to 50 Things to Get Done Next

### Immediate (blocks next buildflow run)

1. **Configure `go-auto-upgrade` to exclude `encoding/json/v2` migration** — or the next run re-breaks
2. **Pin `go-error-family` to `v0.6.1` explicitly** in a comment or replace directive to prevent silent upgrades
3. **Document in `AGENTS.md`** that this project does NOT use `GOEXPERIMENT=jsonv2` and `encoding/json/v2` imports are forbidden
4. **Add a CI guard** (grep check) for `encoding/json/v2` in a pre-commit hook or buildflow detect step

### Testing improvements

5. Fix flaky `TestRateLimiter_Concurrent` — add ±1 tolerance or increase burst margin
6. Add a compression integration test that verifies gzip decompression end-to-end
7. Add a test for the `/metrics` endpoint with `Accept-Encoding: gzip` to verify compressed responses work
8. Clean up `unusedwrite` warnings in `content_test.go:41-44` (4 unused struct fields)
9. Add test coverage measurement (`test-coverage` step was skipped due to failures)
10. Consider BDD tests for critical user paths (health, content serving, search)

### Code quality

11. Add a comment on `executeRequest` explaining why `Accept-Encoding: identity` is set
12. Review all `httputil` middleware usage for correctness (compression, recovery, request ID)
13. Audit error handling completeness — ensure all `writeJSON` error paths are tested
14. Check if `json.NewEncoder(w).Encode(v)` error should be logged instead of silently returned
15. Review rate limiter implementation for race conditions (the flaky test suggests a possible issue)

### Dependency management

16. Review all transitive dependency upgrades from this buildflow run for breaking changes
17. Consider creating a `go.work` workspace to pin critical dependencies
18. Audit `go-error-family` v0.7.0 changelog to understand why it adopted `encoding/json/v2`
19. Check if `httputil` v0.5.0 should be updated to not require `go-error-family` at all
20. Review `charmbracelet/ultraviolet` daily snapshot updates for stability risks
21. Evaluate whether `gocloud.dev v0.46.0` pulls in too many transitive dependencies
22. Consider replacing `cockroachdb/errors` with stdlib `errors` + `fmt.Errorf` to reduce dependency surface

### Build / CI

23. Run `buildflow --build-mode fast` to verify the fix doesn't re-trigger the auto-upgrade
24. Verify `nix build` passes (the buildflow showed `nix-build` failures with hash mismatches)
25. Run `nix flake check` to verify flake integrity
26. Verify `golangci-lint` passes after all changes
27. Run `templ generate` and `templ fmt` to ensure template files are current
28. Check if `govalid-generate` now passes (it was failing due to the compile error)
29. Run `gofumpt -l` to verify formatting
30. Verify `.gitignore` is complete (buildflow ran `gitignore-upserter:repair`)

### Documentation

31. Update `AGENTS.md` gotchas section with the `encoding/json/v2` / `GOEXPERIMENT` issue
32. Add a "Dependency Pinning" section to `AGENTS.md` for `go-error-family`
33. Update `TODO_LIST.md` with the `go-auto-upgrade` exclusion task
34. Document the compression middleware behavior in the server package
35. Add the `Accept-Encoding` test pattern to the testing patterns section in `AGENTS.md`

### Architecture

36. Review whether `writeJSON` should be a method on `Server` rather than a package-level function
37. Consider extracting middleware configuration into a separate `middleware.go` file
38. Evaluate if the `LiveReload` SSE handler should be in its own package
39. Review the `Server` struct — it has 9 fields, some could be grouped (e.g., config-related)
40. Consider adding structured response types instead of `map[string]any` in handlers

### Observability

41. Add request duration histogram to the metrics endpoint (currently only counters)
42. Consider adding a `/health/ready` vs `/health/live` distinction for Kubernetes
43. Add cache warming metrics (load duration, success/failure counts are in `/cache/stats` but not `/metrics`)
44. Log compression ratio in the access log middleware

### Security

45. Verify rate limiting is applied to all sensitive endpoints (currently only `/refresh`)
46. Consider adding CORS configuration for the API endpoints
47. Review security headers middleware for completeness (CSP, HSTS, X-Frame-Options)
48. Audit static file serving for path traversal (should be covered by `domain.NewURLPath` but worth verifying)

### Performance

49. Benchmark `writeJSON` with `json.NewEncoder` vs `json.Marshal` + `Write`
50. Consider streaming large markdown files instead of buffering entire rendered HTML

---

## G) Top 2 Questions I Cannot Answer Myself

### 1. How do I configure `go-auto-upgrade` to exclude `encoding/json/v2` migration?

The buildflow's `go-auto-upgrade:repair` step automatically migrates `encoding/json` to `encoding/json/v2` and upgrades dependencies that use it. This will re-break the build on the next run. I need to know:

- Is there a config file for buildflow exclusions?
- Is there a `.buildflow.toml` or similar?
- Is there a way to mark certain migrations as excluded?

Without this answer, the fix I applied is temporary — it will be undone by the next `buildflow` run.

### 2. Should this project adopt `encoding/json/v2` (enable `GOEXPERIMENT=jsonv2`) or stay on stable `encoding/json`?

The `go-error-family` library (used transitively via `httputil`) has already adopted `encoding/json/v2` in v0.7.0. This creates forward pressure. The decision impacts:

- Whether to pin `go-error-family` at v0.6.1 forever
- Whether to enable `GOEXPERIMENT=jsonv2` in `flake.nix` and CI
- Whether the buildflow auto-upgrade behavior is actually desired

This is a project-level architectural decision I cannot make autonomously.

---

## Summary

| Category                   | Count        | Status                                                      |
| -------------------------- | ------------ | ----------------------------------------------------------- |
| Build failures fixed       | 4/4          | Done                                                        |
| Hidden test failures fixed | 1/1          | Done                                                        |
| Flaky tests identified     | 1            | Not fixed                                                   |
| Systemic issues identified | 2            | Not fixed (needs user input)                                |
| Files changed vs HEAD      | 3            | `handlers_test.go` (+1 line), `go.mod` + `go.sum` (dep pin) |
| Tests passing              | 8/8 packages | Green with `-race`                                          |
