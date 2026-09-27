# TODO List

**Last Updated:** 2026-09-27 (post-Pareto follow-up pass — dependency bumps, Trivy policy, branch protection, upstream filings)
**Purpose:** Short- and mid-term actionable work. Completed items live in [CHANGELOG.md](./CHANGELOG.md); aspirations and decided non-goals live in [ROADMAP.md](./ROADMAP.md).

## 🔴 Critical (Pipeline & Correctness)

- [x] Confirm the first fully-green CI run lands (Test, Build Docker Image including the Trivy security-scan, Website, Nix flake check)
  - **Confirmed 2026-09-27:** run `36332656665` — build ✓ + security-scan ✓ (Trivy gate green under the new CRITICAL+HIGH + ignore-unfixed policy); sibling run `36332656732` — Nix flake check ✓, Lint ✓, Unit + integration tests ✓; Website ✓ (28 s). Root causes fixed earlier the same day: stale `vendorHash.nix` after the dependency bumps (repaired via `nix-hash-fix`), missing `GOEXPERIMENT=jsonv2` in CI, and the Dockerfile that copied a prebuilt binary no step produced.
- [x] Confirm the 3 moderate Dependabot alerts auto-resolve after the next push (`uuid`, `stream-json`, `csv-parse`)
  - Confirmed 2026-09-27: alerts API reports **0 open** after the `firebase-tools` removal + lockfile regen.

## 🟠 High Priority

- [x] Enable branch protection on `master`
  - Done 2026-09-27 via API: required check `Unit + integration tests`, force-push/deletion denied, linear history required, `enforce_admins: false` so the auto-commit daemon's direct pushes keep working; non-admin pushes must pass CI.
- [ ] Add a GitHub social preview image (repo Settings → Social preview; upload `website/public/og/home.png` — the README already embeds it)
  - No public API for this upload; manual 2-minute step.
- [ ] Re-pin otel to stable once `v1.47.0` (or the first stable carrying the CVE fixes) ships — the RC is a documented, accepted tradeoff (see ROADMAP); a Dependabot grouped PR or `buildflow update` should pick it up, verify the Trivy gate stays green.

## 🟡 Medium Priority

- [x] Configure markdownlint MD013 (line length)
  - Done 2026-09-27: `.markdownlint.yml` disables MD013 with rationale (long-line style is deliberate); verified a plain `buildflow -s markdown-lint` run reports 0 MD013 (the initial 2968 were stale result-cache entries — surgically purged, see BuildFlow#20). MD060/MD029/MD056 remain on-demand triage noise, not gate failures.
- [x] Adopt a vulnix policy
  - Decided 2026-09-27: policy-accept build-closure-only CVEs (distroless runtime ships none of it); re-triage on nixpkgs channel bumps. Recorded in ROADMAP.

## 🧹 Smaller Cleanups

- [x] Decide on `github.com/go-sourcemap/sourcemap v2.1.4+incompatible` (indirect via d2)
  - Decided 2026-09-27: accept — upstream dormant (no go.mod, no release since v2.1.4); the `/v2` migration belongs to d2. Recorded in ROADMAP.
- [x] File upstream BuildFlow issues found 2026-09-13
  - Filed 2026-09-27: BuildFlow#19 (`tool_paths` no-op for pnpm-audit), #20 (findings gate aggregates stale cache), #21 (typed `//nolint:branching-flow:panic` breaks nolintlint), #22 (`tsconfig-check`/`type-check` emit tsc help text as findings).

## Resources

- See [ROADMAP.md](./ROADMAP.md) for aspirational items and decided open questions
- See [CHANGELOG.md](./CHANGELOG.md) for completed items
- Historical session reports: [docs/status/](./docs/status/) (resolved reports archived in `docs/status/archived/`; index in [docs/status/README.md](./docs/status/README.md))
