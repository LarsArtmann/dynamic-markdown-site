# TODO List

**Last Updated:** 2026-09-27 (docs-health AUDIT pass — rebuilt from code-verified harvest of `docs/status/` reports)
**Purpose:** Short- and mid-term actionable work. Completed items live in [CHANGELOG.md](./CHANGELOG.md); aspirations live in [ROADMAP.md](./ROADMAP.md).

## 🔴 Critical (Pipeline & Correctness)

- [ ] Triage the 3 moderate Dependabot vulnerabilities GitHub flagged on `master` (2026-09-27 push banner: github.com/LarsArtmann/dynamic-markdown-site/security/dependabot)
  - Found during the 2026-09-27 push; likely Go-side deps (distinct from the `website/` pnpm-audit item below). Bump or dismiss with rationale, then confirm the banner clears.

- [ ] Get a full BuildFlow run green
  - As of `docs/status/2026-09-13_14-52`: `test-coverage` step failed undiagnosed (suspect container coverage shift); go-structure-linter AGENTS.md finding needs a fresh full-run confirmation. `go test ./... -race` is green locally (re-verified 2026-09-27), so the step gate is the open question.
- [ ] Decide & execute the PHANTOM_TYPE findings-gate policy (44 remaining error+ findings)
  - Three options in `docs/status/2026-09-13_14-52` §g1: full rewrite, targeted domain types + nolints, or gate policy change (`fail_on` / severity). Recommendation there: policy change + targeted subset later.
- [ ] Fix `.goreleaser.yaml` license metadata (`license: MIT` in 4 places vs proprietary LICENSE; `flake.nix` correctly says `licenses.unfree`)
  - Pre-existing inconsistency documented in AGENTS.md gotcha #11; misleads Homebrew/Scoop/Nix metadata.
- [ ] Stabilize `TestGracefulShutdownStopsInFlightRequests` (timing flake: EOF on in-flight request, `shutdown_integration_test.go:97`)
  - Failed once on 2026-07-27 and once on 2026-09-13 under hook load; passes in isolation. Root cause the shutdown-vs-response timing, not a data race.

## 🟠 High Priority

- [ ] Rate limiter: evict stale `visitors` entries (TTL + periodic sweep with shutdown wiring)
  - Known production memory leak — every distinct client IP adds a never-evicted map entry (AGENTS.md gotcha #10).
- [ ] Add a rate-limiter test that intentionally exercises token refill (`rate.Every` time behavior is currently avoided, not covered)
- [ ] Add coverage tests for the `do.Invoke` error paths in `internal/container` (package reports 0.0% surface coverage)
- [ ] Add a watcher integration test for `watchForChanges` (temp dir + markdown write + assert refresh; go-filewatcher makes this easy — flagged 2026-07-26)
- [ ] Replace the skipped `pnpm-audit` with a real audit for `website/` (CI step or flake check running `pnpm audit`; documented in AGENTS.md gotcha #15) and triage its first run

## 🟡 Medium Priority

- [ ] Rate limit the `/search` endpoint (only `/refresh` is limited today — `internal/server/handlers.go:167`)
- [ ] Add search result pagination (results currently returned in full)
- [ ] Remove `firebase-tools` from `website/package.json` devDependencies and regenerate the lockfile (debugging artifact from 2026-07-13)
- [ ] Resolve the dual `bun.lock` + `pnpm-lock.yaml` in `website/` — pick the canonical lockfile, delete the other
- [ ] Confirm the Docker upload artifact (`dynamic-markdown-site-${{ sha }}`) appears in the next GitHub Actions run (`docker.yml`)
- [ ] Decide `burst = maxRequests` rate-limiter semantics (full-window burst vs smaller burst + steady refill) and document it on `newRateLimiter`

## 🌐 Website & Docs

- [ ] Add a website CI workflow (`astro check` + build on `website/` changes)
- [ ] Add GitHub social preview image (repo settings; the website OG image exists at `website/public/og/home.png`)
- [ ] Add README screenshots/GIF of the actual UI
- [ ] Replace the dead `https://nixos.wiki/wiki/Flakes` link (403) in CONTRIBUTING.md
- [ ] Migrate deprecated `exhaustruct` → `exhaustruct_v5` in `.golangci.yml`

## 🧹 Smaller Cleanups

- [ ] Extract the duplicated 16 lines in `internal/content/filesystem_test.go:211-238` (jscpd finding, 2026-09-13)
- [ ] Add `-count` repetition guard in CI for the concurrent rate-limiter test (catch future drift)
- [ ] Add missing `platforms` attribute to `flake.nix` meta (flake-meta-checker finding)
- [ ] File upstream issues for the BuildFlow/branching-flow/erraudit defects found 2026-09-13 (`tool_paths` non-functional for pnpm-audit, typed `//nolint:branching-flow:panic` vs nolintlint interop, stale findings-gate aggregation, `tsconfig-check`/`type-check` misfires) — list in `docs/status/2026-09-13_14-52` §c3

## Resources

- See [ROADMAP.md](./ROADMAP.md) for aspirational items and open questions
- See [CHANGELOG.md](./CHANGELOG.md) for completed items
- Historical session reports: [docs/status/](./docs/status/) (resolved reports archived in `docs/status/archived/`)
