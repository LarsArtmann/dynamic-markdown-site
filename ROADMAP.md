# Roadmap

**Generated:** 2026-04-05 | **Last Updated:** 2026-09-27 (docs-health AUDIT pass)
**Purpose:** Aspirational items without timeline, plus open design questions

> Items here are goals and ideas. No commitment on delivery dates.

## 🚀 Features

### Search & Discovery

- [ ] Implement search autocomplete
- [ ] Implement search indexing for better search relevance
- [ ] Add full-text search with Bleve/Meilisearch
- [ ] Implement related content suggestions
- [ ] Implement content tags and filtering

### Performance & Caching

- [ ] Implement cache warming strategy
- [ ] Create cache statistics dashboard
- [ ] Add distributed cache with Redis
- [ ] Benchmark regression tracking in CI
- [ ] Create benchmark suite
- [ ] Benchmark with 1,000+ files

### Rendering & Content

- [ ] Implement content draft preview
- [ ] Add image optimization
- [ ] Add diagram export (PNG/SVG download buttons)
- [ ] Add diagram zoom for large diagrams
- [ ] Implement content versioning (git-based history)
- [ ] Implement live preview (WYSIWYG markdown editing)

### UI/UX

- [ ] Dark mode CSS and theme toggle
- [ ] Implement syntax highlighting themes
- [ ] Add keyboard navigation and shortcuts
- [ ] Implement table of contents with sticky positioning — _base TOC shipped; sticky UX not done_
- [ ] Add print stylesheet
- [ ] Add code copy button (one-click code copying)
- [ ] Add pagination for directories

### Content Delivery

- [ ] Add RSS/Atom feed generation
- [ ] Implement WebSocket live reload
- [ ] Add brotli encoding — EVALUATED & DECLINED 2026-09-27: `httputil` v1.2.0 can wire `br` via `WriterFactories`, but it bundles no brotli encoder, so adopting it means a new direct dependency (e.g. andybalholm/brotli) for a 10-20% gain on already-gzipped HTML. Revisit if payload sizes or bandwidth costs grow (gzip parity is regression-tested in `handlers_test.go`).

- [ ] Add ETag/If-None-Match support

### Observability

- [ ] Add structured logging to renderer package
- [ ] Add response time histograms
- [ ] Implement distributed tracing with OpenTelemetry
- [ ] Add pprof profiling endpoint
- [ ] Add request/response logging with correlation IDs
- [ ] Implement rate limiting per-endpoint configuration

### Admin & API

- [ ] Add admin/debug endpoints (content stats dashboard)
- [ ] Create admin dashboard
- [ ] Add API endpoint for programmatic content access
- [ ] Add analytics integration

### Architecture

- [ ] Implement plugin system
- [ ] Design plugin system for custom markdown extensions
- [ ] Split Repository interface (Reader/Refresher)
- [ ] Structured error types (Is/As/Unwrap)
- [ ] Targeted strong-ID adoption for genuinely identifier-shaped values (top candidates from the 2026-09-27 PHANTOM_TYPE triage: `ratelimit` visitor `ip`, search `query`, sitemap `baseURL`; NOT bools like `HasMermaid` or internal plumbing like `errMsg`)
- [ ] Review `gocloud.dev` dependency weight (large transitive tree for blob storage; evaluate modularization or interface + plugin pattern)

### Internationalization

- [ ] Implement internationalization (multi-language support)

### Platform

- [ ] gRPC/CLI API for programmatic search and content access
- [ ] Consider `nix flake check --all-systems` in CI (aarch64-darwin/aarch64-linux/x86_64-darwin currently omitted)

## ❓ Open Questions

- **Rate-limiter burst semantics:** is `burst = maxRequests` (full-window burst up front, then trickle) the intended production behavior, or should burst be smaller with steady refill? Affects tests and docs. (Raised 2026-07-27.)
- **PHANTOM_TYPE findings-gate policy — DECIDED 2026-09-27 (autonomous, under blanket execution approval):** the gate stays at `error+` severity. Current BuildFlow classifies all `branching-flow` PHANTOM_TYPE suggestions as warning/info (66 + 17 across 25 files, triaged 2026-09-27 — mostly string params like `query`, `ip`, `title`, `baseURL`; zero at error level), so the findings gate is green without suppression. Bulk-rebranding 66 params is rejected for now: many are internal plumbing (`errMsg`, `context`, test helpers) where branded types add ceremony without safety. Targeted adoption lives under Architecture below; revisit if the linter promotes findings to error severity.
- **Changelog automation (git-cliff) — DECLINED 2026-09-27 (autonomous):** releases are rare and the manual Keep-a-Changelog entries are curated, not generated from commit subjects (the auto-commit daemon's `chore:` subjects would pollute generated notes). Revisit only if release cadence grows beyond a few per year.
- **version → buildinfo package rename — WON'T DO (permanent):** `internal/version` is conventional across GoReleaser/Homebrew/distroless ldflags wiring in this repo and matches the ecosystem convention; renaming buys nothing and breaks muscle memory. Closed 2026-09-27.
- **proxyVendor → direct dependency:** `gocloud.dev`'s proxy vendor setup vs. promoting to a direct dependency — re-evaluate when the blob-storage abstraction work (Architecture) is picked up. (Harvested 2026-09-27 from June-era reports.)
- **otel `v1.47.0-rc.1` keep-or-revert — DECIDED 2026-09-27 (keep, autonomous):** the two HIGH otel CVEs (2026-29181/39883) are only fixed on the RC train (latest stable v1.46.0 remains flagged), the packages are indirect transitives of gocloud.dev's cloud drivers rather than our API surface, and the full suite passes under `-race`. Revert trigger: the next stable release carrying the fixes, or any RC-induced test instability.
- **`go-sourcemap v2.1.4+incompatible` — DECIDED 2026-09-27 (accept):** gomod-check flags the `+incompatible` path, but the upstream repo is dormant (no `go.mod`, no release since v2.1.4) and the `/v2` migration is the consumer's call — d2's import, not ours. Unfixable from this side; becomes actionable only if d2 migrates or d2 is dropped (see the gocloud.dev weight review).
- **vulnix nixpkgs base-closure CVEs — DECIDED 2026-09-27 (policy-accept):** the ~24 findings (binutils, curl, bison, coreutils) live in the nixpkgs build closure only; the shipped artifact is a distroless static image and the nix binary carries no closure at runtime. Per-finding dismissal is noise; the policy is to re-triage on nixpkgs channel bumps and act only if a build-time vector becomes exploitable.
- **ldflags `-X` duplication across flake.nix / package.nix / Dockerfile — DECIDED 2026-09-27 (accept + document):** no single source can serve both nix and Docker; the audit's real finding was divergence (package.nix injected nothing), now fixed. The three sites stay synchronized by the `internal/version` contract.

## 🚢 Deployment

- [ ] Add Kubernetes manifests
- [ ] CDN/edge deployment manifests (Cloud Run/Fly.io)
- [ ] Create deployment documentation

## 🔬 Quality

- [ ] Add mutation testing
- [ ] Add content preview functionality
- [ ] Add sample markdown content to content/ directory

## Resources

- See [TODO_LIST.md](./TODO_LIST.md) for actionable items
- See [CHANGELOG.md](./CHANGELOG.md) for completed items
