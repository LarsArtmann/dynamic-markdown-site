# Status Report — 2026-06-18 09:54

**Generated:** 2026-06-18 09:54 (Europe/Berlin)
**Branch:** `master` @ `f4c0f4d`
**Working tree:** 3 modified files, 1 untracked (`CODE_OF_CONDUCT.md`)
**Previous status report:** [`2026-06-18_09-04_goreleaser-deprecations-status.md`](./2026-06-18_09-04_goreleaser-deprecations-status.md)

---

## TL;DR

The buildflow pipeline that was failing on `nix-build` and `nix-flake-check` at session start is now **fully green**. Two real, distinct bugs were found and fixed, both rooted in stale assumptions about build-time values. One important side effect: a separate pre-existing test was not actually a test bug — see (e.3).

---

## a) ✅ FULLY DONE (This Session)

### a.1) Fixed `nix-build` `vendorHash` mismatch

`flake.nix` pinned `vendorHash = "sha256-/bIf2sea5gjbB8GFtl27yePL/BVP4paPr5eeKA4BLVo="`, but `go.sum` was drifted (buildflow's `go-mod-tidy` removed ~59 lines of now-unused transitive deps from `gocloud.dev`'s tree, plus a few others). Nix's fixed-output derivation refused to build:

```
error: hash mismatch in fixed-output derivation '…-go-modules.drv':
         specified: sha256-/bIf2sea5gjbB8GFtl27yePL/BVP4paPr5eeKA4BLVo=
            got:    sha256-7PsgPnmR8KAGhC+Vv7pl1E8lmUKxil83s9HfgOXlvGo=
```

**Fix:** updated `vendorHash` in `flake.nix:39` to `sha256-7PsgPnmR8KAGhC+Vv7pl1E8lmUKxil83s9HfgOXlvGo=`.

| File           | Change               | Status |
| -------------- | -------------------- | ------ |
| `flake.nix:39` | `vendorHash` updated | ✅     |

**Method:** standard Nix dev loop — set hash to a placeholder (`sha256-AAAA…`), run `nix build`, read the actual hash from the error message, paste it back. Verified by clean `nix build` succeeding.

### a.2) Fixed `TestHealthEndpoint` brittleness under ldflags injection

After fixing `vendorHash`, `nix flake check` surfaced a second failure: `checks.test` (which sets `doCheck = true`) runs the test suite with `flake.nix`'s ldflags in effect — and `flake.nix:53-54` injects `-X …version.Version=${version}` with `version = self.rev or "dev"`. The test hardcoded `"version":"dev"` and `"commit":"unknown"`, which are the package-level defaults — so the test failed the moment `nix flake check` injected a real git rev.

```
--- FAIL: TestHealthEndpoint/version (0.00s)
    body should contain "\"version\":\"dev\"", got: …"version":"f4c0f4d4…-dirty"…
--- FAIL: TestHealthEndpoint/commit (0.00s)
    body should contain "\"commit\":\"unknown\"", got: …"commit":"f4c0f4d4…-dirty"…
```

**Fix:** test now reads `version.Version` and `version.Commit` at runtime, so it asserts the _actual_ package state, not a hardcoded string. This is the correct contract: the test verifies "the handler returns the values the version package reports" — independent of how those values got there.

| File                                       | Change                         | Status |
| ------------------------------------------ | ------------------------------ | ------ |
| `internal/server/handlers_test.go:17-19`   | Add `version` import           | ✅     |
| `internal/server/handlers_test.go:114,121` | Use `version.Version`/`Commit` | ✅     |

**Verification:** `go test -run TestHealthEndpoint ./internal/server/...` passes; `nix flake check` reports `all checks passed!`.

### a.3) `nix build`, `nix flake check`, and full `buildflow` are green

End-to-end `buildflow --fix --semantic -p --build-mode=full --budget 1m --log-level warn --no-tui` completes successfully in 23.8s with **all 50+ steps ✔**.

| Step              | Before                    | After |
| ----------------- | ------------------------- | ----- |
| `statix`          | ❌ Failed                 | ✔     |
| `nix-build`       | ❌ Failed (hash mismatch) | ✔     |
| `nix-flake-check` | ❌ Failed (TestHealth)    | ✔     |
| `test-race`       | ✔                         | ✔     |
| `test-coverage`   | ✔                         | ✔     |
| `test-fuzz`       | ✔                         | ✔     |
| All other steps   | ✔                         | ✔     |

---

## b) 🟡 PARTIALLY DONE

### b.1) The flake's `vendorHash` ↔ `go.sum` drift is an ongoing maintenance tax

Fixed today, but `proxyVendor = true` (set in `flake.nix:40`) means any future `go mod tidy` will cause the same failure. The fix is mechanical but disruptive: one must either keep `go.sum` and `flake.nix` in lockstep manually, or move to a less reproducible model. (See e.1 for the proposed migration.)

### b.2) `CODE_OF_CONDUCT.md` is re-introduced as an untracked file

A previous commit (`d48c3bd`) deleted it. Something or someone has added it back to the working tree. Contents look like a stock Contributor Covenant. **Decision needed:** keep, delete, or commit?

---

## c) ❌ NOT STARTED

In the context of this session — the user asked for a status update, no new feature work was requested. The 2 open items from the previous status report remain:

- `TODO_LIST.md`: Implement rate limiting on `/search`
- `TODO_LIST.md`: Verify Docker artifact appears in GitHub Actions

The full `ROADMAP.md` (27 items) is also untouched.

---

## d) 💀 TOTALLY FUCKED UP

### d.1) (RESOLVED) `nix-build` was completely blocked

Reproduced on a clean stash at session start. **Now fixed — see a.1.**

### d.2) (RESOLVED) `nix flake check` was completely blocked

Reproduced after a.1's fix. **Now fixed — see a.2.**

### d.3) The status-report skill's prescribed output format is HTML, but the user asked for `.md`

The `status-report` skill (loaded at the start of this task) explicitly says: _"Write a **self-contained styled HTML dashboard** — not a flat Markdown file."_ The user requested `docs/status/<YYYY-MM-DD_HH-MM_WELL-NAMED>.md`. I followed the user's explicit instruction and produced Markdown. The skill's HTML template was not used. This is intentional and per the operating principle of following user instructions over skill defaults, but it does break the skill's expected output.

### d.4) `internal/server/content_test.go:41-44` has unused field writes (gopls `unusedwrite`)

```go
Content:     … // line 41 — unused
ContentType: … // line 42 — unused
ModTime:     … // line 43 — unused
Size:        … // line 44 — unused
```

LSP diagnostic surface. Not failing any test, not blocking CI, but indicates dead test setup. **Not fixed in this session** (out of scope and unrelated to the buildflow failure).

### d.5) `TestRateLimiter_Concurrent` is racy in theory

The test expects exactly 100 allowed requests out of 200 concurrent attempts. With token-bucket semantics, "exactly 100" is hard to guarantee under heavy contention. Verified to pass 20/20 in a tight loop locally — the token bucket is generous enough (10 rps per IP, but the limiter is per-IP and 200 simultaneous goroutines can race to claim tokens). However, the design of the assertion (`allowedCount != 100`) is fragile and could break under different timing. **Not fixed in this session** (out of scope and passes today).

---

## e) 🔧 WHAT WE SHOULD IMPROVE

### e.1) Drop `proxyVendor = true`; let `go build` resolve modules on demand

The `vendorHash` ↔ `go.sum` lockstep problem will recur every time we `go mod tidy`. Two paths:

1. **Drop `proxyVendor`** — `buildGoModule` will use `go.sum` directly. Less reproducible across toolchain versions, but eliminates the manual hash step.
2. **Add a `pre-commit` hook that runs `nix build`** — catches drift before commit, but adds 30+ seconds to every commit and requires Nix in the dev shell.

I recommend **#1** with a follow-up CI step that builds from a clean cache to catch any genuine hash drift.

### e.2) Wire `nix flake check` into `.github/workflows/test.yml`

It currently runs `go test` + `golangci-lint` + `templ generate` — but not `nix flake check`. Adding it would have caught **both** regressions fixed in this session at PR time. ~5 lines of YAML.

### e.3) The `TestHealthEndpoint` "bug" was actually a build/test design issue, not a test bug

The test was a victim of the flake's ldflags injection. The deeper design question: **should the test be testing the health endpoint's contract, or the version package's contract?** Right now it tests the wire-level JSON shape, which is right. But by hardcoding `"dev"`/`"unknown"` it was implicitly asserting "and the version package will not be overridden" — a build-time guarantee, not a runtime one. The fix (a.2) decouples the two contracts. This is a good pattern to apply to similar tests elsewhere.

### e.4) Status reports have inconsistent naming and overlap

`docs/status/` has 18+ files with a mix of `comprehensive-status`, `comprehensive-refactoring`, and date-prefixed names. Some of them are snapshots of a moment in time; some are execution plans. They should be split into:

- `docs/status/<DATE>_<SLUG>.md` — point-in-time status snapshots (keep)
- `docs/planning/<DATE>_<SLUG>.md` — execution plans and todo breakdowns (already exists at `docs/planning/`)

### e.5) `TODO_LIST.md` has 6 "Stale:" items that should be removed

From the previous report and current review: items marked `[x]` but still listed under the work-in-progress sections. E.g. "Remove dead `addError` method from `treeStats`" — that struct doesn't even exist anymore. The TODO list should be a _todo_ list, not a _done_ list.

### e.6) `go.sum` modification is uncommitted in the working tree

The buildflow run modified `go.sum` (removing unused transitives) but the change is uncommitted. The `vendorHash` fix is meaningless without committing the new `go.sum` — otherwise the next clean checkout will re-introduce the hash mismatch. **This report's commit will include the go.sum change.**

### e.7) The `result` symlink in the working tree is a build artifact

`result → /nix/store/…-dirty` was modified by the `nix build` invocation. It should be in `.gitignore` (or `pre-commit` should delete it). Currently it shows as a modified file in `git status`, which is noise.

---

## f) 🎯 Top #25 things to get done next

Ranked by impact / effort ratio. **Bold** = addresses a finding from this status report.

| #   | Task                                                                                            | Impact  | Effort | Notes                                                          |
| --- | ----------------------------------------------------------------------------------------------- | ------- | ------ | -------------------------------------------------------------- |
| 1   | **Drop `proxyVendor = true` in `flake.nix`** to eliminate `vendorHash` ↔ `go.sum` drift         | 🔴 High | S      | Stops a recurring CI failure mode. See e.1.                    |
| 2   | **Add `nix flake check` to `.github/workflows/test.yml`** to catch flake regressions at PR time | 🔴 High | XS     | Would have caught both fixes in this session. See e.2.         |
| 3   | **Add `result` (and other Nix build outputs) to `.gitignore`**                                  | 🟢 Low  | XS     | Pure noise reduction. See e.7.                                 |
| 4   | **Remove 6 "Stale:" items from `TODO_LIST.md`**                                                 | 🟢 Low  | XS     | Cleanup. See e.5.                                              |
| 5   | **Commit the working-tree `go.sum` cleanup alongside the `vendorHash` fix**                     | 🔴 High | XS     | Currently these are two separate uncommitted changes. See e.6. |
| 6   | Implement rate limiting on `/search` (TODO_LIST)                                                | 🟡 Med  | S      | One IP can hammer the in-memory search index.                  |
| 7   | Decide on `CODE_OF_CONDUCT.md` — keep, delete, or commit (see b.2)                              | 🟡 Med  | XS     | Tracked; awaiting user decision.                               |
| 8   | Verify Docker artifact appears in GitHub Actions (TODO_LIST)                                    | 🟡 Med  | XS     | One-line confirmation on the next release.                     |
| 9   | Add `--cask` to `release:` footer in `.goreleaser.yaml` and `README.md` install instructions    | 🟡 Med  | XS     | From the previous status report.                               |
| 10  | Audit other tests for ldflags-injection brittleness (apply a.2's pattern)                       | 🟡 Med  | M      | Search for hardcoded strings that should read package vars.    |
| 11  | Add `goreleaser check` to `.github/workflows/release.yml`                                       | 🟡 Med  | XS     | Prevents deprecation drift.                                    |
| 12  | Implement search result pagination (TODO_LIST)                                                  | 🟢 Low  | S      | Only needed at high cardinality.                               |
| 13  | Dark mode CSS and theme toggle (ROADMAP)                                                        | 🟢 Low  | M      | Popular ask; templ + CSS variables.                            |
| 14  | Code copy button on code blocks (ROADMAP)                                                       | 🟢 Low  | XS     | A few lines of JS in `layout.templ`.                           |
| 15  | ETag / `If-None-Match` support (ROADMAP)                                                        | 🟢 Low  | S      | Drop-in for `static/*` responses.                              |
| 16  | gzip/brotli compression middleware (ROADMAP)                                                    | 🟢 Low  | S      | `compress` middleware; ~10 lines.                              |
| 17  | Wire `cache.Warm()` for the most-recent N pages on startup (ROADMAP)                            | 🟢 Low  | M      | Use `AllPaths()` from content tree.                            |
| 18  | pprof endpoint behind a flag (ROADMAP)                                                          | 🟢 Low  | XS     | `net/http/pprof` + flag gate.                                  |
| 19  | OpenTelemetry tracing (ROADMAP)                                                                 | 🟢 Low  | L      | Big surface; defer.                                            |
| 20  | Kubernetes manifests (ROADMAP)                                                                  | 🟢 Low  | S      | Single Deployment + Service + Ingress.                         |
| 21  | RSS/Atom feed (ROADMAP)                                                                         | 🟢 Low  | S      | Walk `AllPaths()`; filter drafts.                              |
| 22  | Verify `sitemap.xml` is wired end-to-end with a smoke test                                      | 🟢 Low  | XS     | FEATURES.md claims it; never exercised.                        |
| 23  | Sample markdown content in `content/` (ROADMAP)                                                 | 🟢 Low  | S      | One demo page per feature.                                     |
| 24  | Mutation testing (ROADMAP)                                                                      | 🟢 Low  | M      | `go-mutesting` integration.                                    |
| 25  | Plugin system design doc (ROADMAP)                                                              | 🟢 Low  | L      | ADR-grade write-up before any code.                            |

**Pareto (top 5 = 80% of value):** items 1–5, all of which are direct consequences of this session's findings.

---

## g) ❓ The one question I cannot answer myself

**Should we drop `proxyVendor = true` (e.1) and accept slightly less reproducible Nix builds, or keep it and add a `pre-commit` hook that runs `nix build` to catch drift early?**

`proxyVendor` was added in commit `d6c0298` ("Add proxyVendor = true for deterministic Go module downloads"). The intent was reproducibility. But the actual cost is one human debugging session per `go mod tidy`, plus the recurring CI failure mode we just hit. The alternatives:

- **Drop `proxyVendor`** — `go.sum` is the source of truth, Nix uses it directly. Reproducibility degrades from "byte-identical" to "byte-identical for a given `go.sum` and toolchain", which is still a strong guarantee.
- **Keep `proxyVendor`, add `pre-commit` hook** — adds ~30s to every commit, requires Nix in dev shell, still doesn't help external contributors who don't use Nix.
- **Keep `proxyVendor`, add CI step** — catches it in PR but the local dev experience stays the same.

I can't decide this without knowing whether the project's reproducibility target is "byte-identical across machines" or "byte-identical within a Go toolchain version". **What's the intended contract?**

---

## Snapshot

- **Source size:** 83 Go files, ~14 500 lines (unchanged from previous report)
- **Internal packages:** `cache`, `config`, `container`, `content`, `domain`, `renderer`, `server`, `test`, `version`
- **Test coverage (per package):** cache 95.5%, config 90.7%, content 76.4%, domain 81.0%, renderer 83.9%, server 84.5%; container 0.0% (subprocess harness hides surface coverage — pre-existing)
- **CI workflows:** `docker.yml`, `release.yml`, `test.yml`
- **Stale status reports in `docs/status/`:** 18 (including this one)
- **Open TODO items:** 2 (rate-limit search, verify Docker artifact)
- **Open ROADMAP items:** 27
- **Buildflow status:** ✅ all 50+ steps green

---

_Generated by Crush._
