# Comprehensive Status Report — 2026-04-01 08:28

**Session Start:** 2026-04-01 ~07:23 CEST
**Session End:** 2026-04-01 08:28 CEST
**Branch:** `master`
**Commit:** `170ac66` (1 ahead of origin, not yet pushed)

---

## a) FULLY DONE

### CI Pipeline Debugging & Root Cause Analysis
- **Identified 3 independent CI failure categories** across 6+ consecutive failed runs:
  1. Test failure: `TestBlobRepositoryWithContent` — `fileNode.Title` used as field instead of method call `fileNode.Title()`
  2. Linter: `gochecknoglobals` flagging `version.go` globals, `golines` flagging long lines in `security.go`, `handlers.go`, `version.go`
  3. Trivy security scan: runs in parallel with build, tries to scan Docker image that doesn't exist yet

### Test Fix Applied
- **File:** `internal/content/blob_test.go`
- **Root cause:** `FileNode.Title` is a **method** (not an exported field). Test used `fileNode.Title` (function reference) instead of `fileNode.Title()` (method call)
- **Error message:** `Invalid operation: "index" == (func() string)(0x2312760) (cannot take func type as argument)`
- **Fix:** Changed `fileNode.Title` → `fileNode.Title()` on lines 127 and 139
- **Verified:** Tests pass locally: `ok github.com/larsartmann/dynamic-markdown-site/internal/content 2.449s coverage: 76.3%`

### Linter Fix: gochecknoglobals
- **File:** `internal/version/version.go`
- **Root cause:** `Version`, `Commit`, `BuildDate` are package-level globals, required for ldflags injection at build time
- **Fix:** Added `//nolint:gochecknoglobals` directive with explanatory comment
- **Verified:** `golangci-lint run ./internal/version/` — clean, no output

### Linter Fix: golines (CSP header)
- **File:** `internal/server/security.go`
- **Root cause:** Content-Security-Policy header value was a single 250+ character string literal
- **Fix:** Split into multi-line string concatenation for readability
- **Verified:** Linter clean

### CI Workflow Fix: Trivy Security Scan
- **File:** `.github/workflows/docker.yml`
- **Root cause:** `security-scan` job had no `needs:` dependency on `build` job, so it ran in parallel and tried to scan a Docker image that hadn't been built yet
- **Fix:** Added `needs: [build]` to `security-scan` job
- **Additional CI warnings noted but NOT fixed** (deprecation warnings, not failures):
  - Node.js 20 actions deprecation (June 2026)
  - CodeQL Action v3 deprecation (December 2026)

### Local Environment Recovery
- **Disk space:** Freed ~3.4GB by cleaning Go build caches (`goimports`, `gopls`, `golangci-lint`, temp build dirs)
- **Disk status:** 99% → 98% (2.4GB → 5.8GB free)
- **Go toolchain:** Working for test execution (all internal packages pass)
- **Note:** `cmd/` package fails locally due to Go 1.26.0 vs 1.26.1 mismatch (Nix-installed Go is 1.26.0, go.mod requires 1.26.1). CI uses correct 1.26.1.

---

## b) PARTIALLY DONE

### Blob Storage Feature (from commit `b5533f5`)
- Core implementation: **Done** — `BlobRepository` supports file, S3, GCS, Azure Blob, mem backends via `gocloud.dev/blob`
- URL path stripping: **Done** — blob paths strip `.md` extension (e.g., `/index` not `/index.md`)
- Test coverage: **Fixed** — was broken due to method-vs-field bug, now passes
- **Remaining:** Not yet tested against actual cloud storage (only file:// and mem:// tested)

### CI Pipeline Green Status
- All code fixes committed locally in `170ac66`
- **Not yet pushed** — awaiting push + CI verification
- Linter was skipped in all recent CI runs (tests failed first, blocking linter step)
- Need to confirm linter passes in CI after push

---

## c) NOT STARTED

### Filesystem Repository Path Inconsistency
- `FileSystemRepository` creates URL paths WITH `.md` extension (e.g., `/test.md`)
- `BlobRepository` creates URL paths WITHOUT `.md` extension (e.g., `/index`)
- This is an **inconsistency** between the two `Repository` implementations
- Tests in `filesystem_test.go` expect `/test.md` format
- Decision needed: should both strip extensions, or both keep them?

### Node.js 20 Deprecation in GitHub Actions
- `actions/checkout@v4`, `actions/setup-go@v5` use Node.js 20
- Forced upgrade to Node.js 24 on June 2, 2026
- Could update to `actions/checkout@v5` (if available) or set `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true`

### CodeQL Action v3 Deprecation
- `github/codeql-action/upload-sarif@v3` deprecated December 2026
- Should update to `@v4`

### CSP Header Testing
- No automated tests verify the Content-Security-Policy header values are correct
- No tests verify security headers are present in responses

### Search Index for Blob Storage
- `Searcher` may need adaptation to work with blob-backed content
- Not investigated yet

---

## d) TOTALLY FUCKED UP

### Local Disk Space Crisis
- 229GB disk at 98% capacity (5.8GB free) — **critically low**
- Go caches alone were 6+ GB before cleanup
- Will fill up again quickly with normal development
- Root cause likely: accumulated Nix store, Docker images, Go module cache
- **Recommendation:** Audit large directories with `du -sh ~/Library/Caches/* ~/go/pkg/*` and clean Docker images with `docker system prune`

### Local Go Version Mismatch
- Nix-installed Go: 1.26.0
- go.mod requires: 1.26.1
- `cmd/` package fails to build locally
- Fix: update Nix Go package to 1.26.1 or use `GOTOOLCHAIN=auto`

### CI Was Broken for 6 Consecutive Runs
- Runs #17-#22 all failed (from `b5533f5` through `8b05995`)
- Only run #16 (pre-blob-storage) succeeded
- The `fileNode.Title` bug existed in the ORIGINAL blob test code from commit `b5533f5`
- This was a code review miss — the test was never run before committing

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Run tests locally before pushing** — The blob test bug would have been caught immediately
2. **Code review for test code** — Test code quality matters; method-vs-field is a basic mistake
3. **CI fast-feedback** — Consider splitting lint and test into separate jobs so lint failures are visible even when tests fail
4. **Disk space monitoring** — Add a CI step or local hook that warns when disk > 95%
5. **Go version alignment** — Pin local Go version to match go.mod exactly

### Technical Improvements

6. **Consistent URL path behavior** — Both repository implementations should handle extensions identically
7. **Interface compliance testing** — Add compile-time checks that test types satisfy `ContentNode` correctly
8. **Security header tests** — Automated verification of CSP, HSTS, and other security headers
9. **Blob integration tests** — Test against real cloud storage (even if just in CI with LocalStack/minio)
10. **Error messages in tests** — The `assert.Equal` error for method-vs-field was confusing; consider `assert.Equal(t, "index", fileNode.Title())` with explicit error context

---

## f) Top #25 Things We Should Get Done Next

### Critical (CI must be green)

| # | Task | Priority | Effort |
|---|------|----------|--------|
| 1 | **Push commit `170ac66` to origin** | P0 | 1 min |
| 2 | **Verify CI passes** on the pushed commit | P0 | 5 min |
| 3 | **Fix any remaining linter issues** if CI lint step fails | P0 | 15 min |

### High Priority (Stability & Consistency)

| # | Task | Priority | Effort |
|---|------|----------|--------|
| 4 | **Resolve filesystem vs blob URL path inconsistency** | P1 | 1-2 hr |
| 5 | **Fix local Go version** (update Nix to 1.26.1) | P1 | 15 min |
| 6 | **Free more disk space** (audit Nix store, Docker images) | P1 | 30 min |
| 7 | **Update GitHub Actions** to Node.js 24 compatible versions | P1 | 30 min |
| 8 | **Update CodeQL action** to v4 | P1 | 10 min |

### Medium Priority (Quality & Testing)

| # | Task | Priority | Effort |
|---|------|----------|--------|
| 9 | **Add security header tests** for all HTTP responses | P2 | 1-2 hr |
| 10 | **Add blob integration tests** with minio/localstack | P2 | 2-3 hr |
| 11 | **Separate lint and test CI jobs** for parallel failure visibility | P2 | 30 min |
| 12 | **Add `content.Repository` interface compliance test** | P2 | 1 hr |
| 13 | **Increase blob test coverage** above 76% | P2 | 2-3 hr |
| 14 | **Add CSP header validation** (parse and verify directives) | P2 | 1 hr |

### Lower Priority (Nice-to-Have)

| # | Task | Priority | Effort |
|---|------|----------|--------|
| 15 | **Add dependabot config** for Go modules and GitHub Actions | P3 | 30 min |
| 16 | **Add release-please** or similar for automated releases | P3 | 1-2 hr |
| 17 | **Add branch protection rules** (require CI pass before merge) | P3 | 15 min |
| 18 | **Add README badges** (CI status, coverage, Go version) | P3 | 15 min |
| 19 | **Set up monitoring/alerting** for disk space on dev machine | P3 | 30 min |
| 20 | **Add pre-commit hooks** (go test, golangci-lint) | P3 | 30 min |
| 21 | **Document blob storage configuration** (S3, GCS, Azure) in README | P3 | 1 hr |
| 22 | **Add E2E tests** with actual HTTP server + markdown content | P3 | 2-3 hr |
| 23 | **Performance benchmarks** for blob vs filesystem repository | P4 | 1-2 hr |
| 24 | **Add OpenTelemetry tracing** for content loading | P4 | 2-3 hr |
| 25 | **Add structured error responses** (JSON error pages) | P4 | 1-2 hr |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `FileSystemRepository` strip `.md` extensions from URL paths to match the `BlobRepository` behavior?**

Currently:
- `BlobRepository.Get("/index")` returns the file `index.md` — URL path has no extension
- `FileSystemRepository.Get("/test.md")` returns the file `test.md` — URL path HAS extension

This means:
- Clients using the filesystem backend must request `/page.md`
- Clients using the blob backend must request `/page`
- The same markdown files produce different URL structures depending on storage backend

This is a **design decision** that affects the API contract. The "correct" answer depends on:
- Should users see `.md` in URLs? (Generally no for clean URLs)
- Should both backends behave identically? (Generally yes for portability)
- Does the existing filesystem-based deployment have users depending on `.md` URLs?

My recommendation: **Yes, both should strip extensions** — but this is a breaking change for existing filesystem deployments.

---

## File Change Summary (Commit `170ac66`)

| File | Change | Lines |
|------|--------|-------|
| `.github/workflows/docker.yml` | Added `needs: [build]` to security-scan | +1 |
| `internal/content/blob_test.go` | Fixed `fileNode.Title` → `fileNode.Title()` (×2) | 2 changed |
| `internal/server/security.go` | Split CSP header into multi-line string | +10/-1 |
| `internal/version/version.go` | Added `//nolint:gochecknoglobals` directive | +2 |

**Total: 4 files changed, 14 insertions, 3 deletions**

---

## CI Run History

| Run | Commit | Status | Notes |
|-----|--------|--------|-------|
| #23 | `8b05995` | ❌ Failure | Blob test: method-vs-field bug |
| #22 | `0115428` | ❌ Failure | Same test bug (earlier fix attempt) |
| #21 | `c888c39` | ❌ Failure | Same test bug (formatting commit) |
| #20 | `b5533f5` | ❌ Failure | Original blob storage commit, test bug present |
| #19 | `09a0a16` | ❌ Failure | Pre-blob, different issue |
| #18 | `c813efe` | ❌ Failure | Pre-blob, different issue |
| #16 | `20b275f` | ✅ Success | Last green CI run (pre-blob-storage) |
| `170ac66` | **NOT YET PUSHED** | ⏳ Pending | All 4 fixes applied |

---

_Auto-generated by Crush (AI Assistant) at 2026-04-01T08:28:10+02:00_
