# TODO List

**Last Updated:** 2026-09-27 (Pareto plan execution pass — T1-T24 of `docs/planning/2026-09-27_04-35_pareto-execution-plan-verification-trust-and-security-honesty.md`)
**Purpose:** Short- and mid-term actionable work. Completed items live in [CHANGELOG.md](./CHANGELOG.md); aspirations and decided non-goals live in [ROADMAP.md](./ROADMAP.md).

## 🔴 Critical (Pipeline & Correctness)

- [ ] Confirm the 3 moderate Dependabot alerts auto-resolve after the next push (GHSA-w5hq-g745-h8pq `uuid`, GHSA-528h-pc64-c93x `stream-json`, GHSA-8cw4-87c7-c6xx `csv-parse`)
  - Fix already applied 2026-09-27: all three were `firebase-tools` transitives, and removing `firebase-tools` + regenerating `website/pnpm-lock.yaml` eliminated them from the tree (verified by grep). The alerts re-scan the dependency graph on push — check `<https://github.com/LarsArtmann/dynamic-markdown-site/security/dependabot>` clears (owner-only page, so link checkers see a 404).
- [ ] Confirm the first green CI run after the pipeline fixes lands (Test, Build Docker Image, Website, and the new Nix flake check job)
  - Root causes fixed 2026-09-27: workflows ran with `GOEXPERIMENT=''` (golangci-lint crashed on json/v2), and `docker.yml` copied a prebuilt binary that no step produced (Dockerfile is now multi-stage). Verify the banner clears on the next master push.

## 🟠 High Priority

- [ ] Enable branch protection on `master` (repo Settings → Branches; found unprotected 2026-09-27)
  - Suggested: require the `Test` + `Website` checks to pass, require a pull request for direct master pushes. Owner decision — affects the auto-commit daemon's ability to push straight to master, so not applied autonomously.
- [ ] Add a GitHub social preview image (repo Settings → Social preview; upload `website/public/og/home.png` — the README already embeds it)
- [ ] Audit `goreleaser` `ldflags` usage for brittleness (the `version`/`Commit`/`BuildDate` `-X` flags are duplicated across `flake.nix`, `package.nix`, and the Dockerfile builder stage; extract a shared source or accept and document the duplication)

## 🟡 Medium Priority

- [ ] Configure markdownlint MD013 (line length) or accept the ~2,700-line finding noise — markdown-lint runs on-demand only (`-s markdown-lint`), so this is a triage decision, not a gate failure
- [ ] Adopt a vulnix policy: decide whether nixpkgs base-closure CVEs (binutils, curl, bison, coreutils; 24 findings at 2026-09-27, all build-time-only) are dismissed by policy or trigger a nixpkgs channel bump

## 🧹 Smaller Cleanups

- [ ] Decide on `github.com/go-sourcemap/sourcemap v2.1.4+incompatible` (indirect via d2): file an upstream issue for the `+incompatible` module path or add a gomod-check ignore
- [ ] File upstream BuildFlow issues found 2026-09-13 (`tool_paths` non-functional for pnpm-audit, typed `//nolint:branching-flow:panic` vs nolintlint interop, stale findings-gate aggregation, `tsconfig-check`/`type-check` misfires) — list in `docs/status/2026-09-13_14-52` §c3

## Resources

- See [ROADMAP.md](./ROADMAP.md) for aspirational items and decided open questions
- See [CHANGELOG.md](./CHANGELOG.md) for completed items
- Historical session reports: [docs/status/](./docs/status/) (resolved reports archived in `docs/status/archived/`; index in [docs/status/README.md](./docs/status/README.md))
