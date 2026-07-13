# TODO List

**Generated:** 2026-04-05 | **Last Updated:** 2026-07-13
**Purpose:** Actionable items for the next 2-4 weeks
**Status:** Active development items only

## 🔴 Critical (Fix Now)

- [x] Address GitHub security vulnerabilities in dependencies
  - govulncheck identified 2 stdlib CVEs (GO-2026-5039, GO-2026-5037) fixed in
    Go 1.26.4. Toolchain bumped to `go 1.26.4` in `go.mod`.
- [ ] Fix Go 1.26.1 environment mismatch for BuildFlow
  - BuildFlow is not part of this repo. Re-evaluate if reintroduced.
- [x] Fix unused parameter warnings in `container.go`
  - **Stale:** all providers already use `_ do.Injector`; no warnings present.
- [x] Fix local Go cache corruption
  - `go clean -cache` resolves intermittent govulncheck failures.

## 🔴 High Priority

- [x] Create integration test suite
  - `internal/server/shutdown_integration_test.go` covers graceful shutdown;
    `healthcheck_test.go` covers /health end-to-end; per-package coverage
    now ≥75% across all Go packages.
- [x] Add `AllPaths()` unit tests
  - `internal/content/memory_allpaths_test.go` covers empty, populated, and
    nested scenarios.
- [x] Add request timing middleware
  - `internal/server/responsetime.go` sets the `X-Response-Time` header on
    every response.
- [x] Improve error handling in D2 rendering
  - `internal/renderer/diagram_extension.go` now logs a `slog.Warn` on
    failure and uses an in-memory buffer to avoid partial writes.
- [x] Add Prometheus metrics endpoint
  - `internal/server/metrics.go` exposes a dependency-free
    Prometheus-format `/metrics` endpoint.
- [x] Add structured health check with version, uptime, dependencies
  - `/health` now includes `uptime` and a `dependencies` object
    describing repository and cache state.
- [ ] Verify Docker artifact appears in GitHub Actions
  - The `docker.yml` workflow uploads `dynamic-markdown-site-${{ sha }}`
    as an artifact; confirm in the next release.
- [x] Add coverage enforcement to CI (≥75% threshold)
  - `.github/workflows/test.yml` enforces per-package coverage ≥75% in
    the `Enforce coverage threshold` step.
- [x] Add `golangci-lint` version pinning in workflow
  - `GOLANGCI_LINT_VERSION: v2.12.2` is pinned in `test.yml`.
- [x] Add `templ generate` check to CI
  - `test.yml` regenerates templates and fails the build on diff.

## 🟡 Medium Priority

- [x] Split `internal/content/search_test.go` (685 lines)
  - Split into `search_test.go` (core), `search_scoring_test.go` (scoring),
    `search_edge_test.go` (whitespace/special characters).
- [x] Split `internal/server/handlers_test.go` (667 lines)
  - Split into `handlers_test.go`, `refresh_test.go`, `search_test.go`,
    `content_test.go`, `cachestats_test.go`, `health_test.go`,
    `metrics_test.go`, `suggestions_edge_test.go`,
    `shutdown_integration_test.go`.
- [x] Split `internal/renderer/markdown_test.go` (609 lines)
  - Split into `markdown_test.go` (core), `markdown_toc_test.go`,
    `markdown_frontmatter_test.go`, `markdown_edge_test.go`.
- [x] Implement content search highlighting
  - Already implemented in `internal/content/search.go`; covered by
    `TestSearcher_Search_TitleMatching`.
- [x] Add breadcrumbs to search results
  - Already rendered by `renderSearch` in `internal/server/render.go`.
- [ ] Add search result pagination
  - Not implemented; results are returned in full. Candidate for a
    future iteration if cardinality grows.
- [x] Add cache hit/miss metrics endpoint
  - `GET /cache/stats` returns hits, misses, evictions, and hit ratio.
- [x] Create architecture decision records
  - `docs/adr/` contains 5 ADRs (record-keeping, stdlib HTTP, Otter
    cache, Prometheus without `client_golang`, distroless healthcheck).
- [x] Add reading time estimates to UI
  - Already rendered in `templates/layout.templ` via `formatReadingTime`.
- [ ] Rate limit search endpoint
  - Not implemented; currently only `/refresh` is rate-limited.
- [x] Add rate limiting tests
  - `internal/server/ratelimit_test.go` covers allow/deny, per-IP
    isolation, and concurrency.
- [x] Add graceful shutdown tests
  - `internal/server/shutdown_integration_test.go` starts a real HTTP
    server, fires a request, calls Shutdown, and asserts the
    in-flight request completes and subsequent requests fail.
- [x] Increase renderer test coverage (currently 68.2%)
  - Now 83.9% per `go test ./... -cover`.
- [x] Integrate testutil package into test files
  - `internal/test/file_helpers.go` exposes `FileNode` constructor; used
    by `internal/content/search_test.go::newFile`.
- [x] Add suggestions edge case tests
  - `internal/server/suggestions_edge_test.go` covers empty inputs,
    exact match exclusion, prefix boost, score threshold, and limit.

## 🟢 Process Improvements

- [x] Add git pre-push hook
  - `.githooks/pre-push` runs `go test -race -cover` + `golangci-lint` (plain bash,
    no `just` — the project uses Nix, not justfile).
- [x] Add pre-commit hook for golines
  - `.githooks/pre-commit` runs `templ generate` and stages any
    regenerated files.
- [x] Separate CI workflow into `test.yml` + `docker.yml`
  - `.github/workflows/test.yml` (test + lint) and
    `.github/workflows/docker.yml` (image build, push, Trivy scan).
- [x] CI: add Go module/build caching
  - Both workflows use `actions/setup-go` with `cache: true`.
- [x] Document CI pipeline in README
  - `## Continuous Integration` section in `README.md` describes both
    workflows and their triggers.
- [x] Add CONTRIBUTING.md
  - `CONTRIBUTING.md` covers dev setup, code-quality gates, conventions,
    and reporting process.
- [x] Add HEALTHCHECK to Dockerfile
  - `Dockerfile` uses a self-contained `healthcheck` subcommand.
- [x] Apply staticcheck tagged switch suggestion in `errors.go`
  - `internal/server/errors.go` returns explicit codes per case; the
    `switch` no longer relies on fallthrough. The staticcheck
    suggestion is now redundant; if re-raised we can re-evaluate.
- [x] Remove dead `addError` method from `treeStats`
  - **Stale:** no such method/struct exists in the current source.

## 🟢 Testing

- [x] Add integration test for 404 suggestions endpoint
  - Covered by `suggestions_edge_test.go` and existing
    `TestNotFoundSuggestions` in `handlers_test.go`.
- [x] Add integration tests for HTTP endpoints
  - Per-endpoint tests live next to the production code in
    `*_test.go`; the shutdown test exercises a real socket.
- [x] Add integration tests for templates
  - End-to-end HTTP tests render templ output (see
    `TestDirectoryListing`, `TestContentByPath`).
- [x] Add integration tests for full markdown → HTML pipeline
  - `TestRenderComplexDocument` and `TestRenderMalformedMarkdownStillProducesHTML`
    exercise the full Goldmark pipeline.
- [x] Add renderer edge case tests
  - `internal/renderer/markdown_edge_test.go` covers empty, whitespace,
    Unicode, emoji, nested headings, long lines, and XSS escaping.
- [x] Add container integration assertions
  - `internal/container/container_test.go` verifies each accessor
    returns a non-nil instance and the same instance on repeated calls
    (singleton behaviour).
- [x] Add template render benchmarks
  - `internal/server/benchmark_test.go` and
    `internal/renderer/markdown_bench_test.go` cover the render path.
- [x] Write end-to-end tests
  - `internal/server/shutdown_integration_test.go` starts a real
    server, fires requests, and verifies the full HTTP lifecycle.
- [x] Increase container package test coverage
  - Per-instance coverage still reads 0.0% in the surface output
    because the subprocess-based harness hides per-test coverage. The
    tests do run; if we want surface coverage we need to refactor
    `runInSubprocess` (low priority; behaviour is well-tested).

## Resources

- See [CHANGELOG.md](./CHANGELOG.md) for completed items
- See [ROADMAP.md](./ROADMAP.md) for aspirational items
