# TODO List

**Generated:** 2026-04-05
**Purpose:** Actionable items for the next 2-4 weeks
**Status:** Active development items only

## 🔴 Critical (Fix Now)

- [ ] Address GitHub security vulnerabilities in dependencies
- [ ] Fix Go 1.26.1 environment mismatch for BuildFlow
- [ ] Fix unused parameter warnings in `container.go`
- [ ] Fix local Go cache corruption

## 🔴 High Priority

- [ ] Create integration test suite
- [ ] Add `AllPaths()` unit tests
- [ ] Add request timing middleware
- [ ] Improve error handling in D2 rendering
- [ ] Add Prometheus metrics endpoint
- [ ] Add structured health check with version, uptime, dependencies
- [ ] Verify Docker artifact appears in GitHub Actions
- [ ] Add coverage enforcement to CI (≥75% threshold)
- [ ] Add `golangci-lint` version pinning in workflow
- [ ] Add `templ generate` check to CI

## 🟡 Medium Priority

- [ ] Split `internal/content/search_test.go` (685 lines)
- [ ] Split `internal/server/handlers_test.go` (667 lines)
- [ ] Split `internal/renderer/markdown_test.go` (609 lines)
- [ ] Implement content search highlighting
- [ ] Add breadcrumbs to search results
- [ ] Add search result pagination
- [ ] Add cache hit/miss metrics endpoint
- [ ] Create architecture decision records
- [ ] Add reading time estimates to UI
- [ ] Rate limit search endpoint
- [ ] Add rate limiting tests
- [ ] Add graceful shutdown tests
- [ ] Increase renderer test coverage (currently 68.2%)
- [ ] Integrate testutil package into test files
- [ ] Add suggestions edge case tests

## 🟢 Process Improvements

- [ ] Add git pre-push hook calling `just pre-push`
- [ ] Add pre-commit hook for golines
- [ ] Separate CI workflow into `test.yml` + `docker.yml`
- [ ] CI: add Go module/build caching
- [ ] Document CI pipeline in README
- [ ] Add CONTRIBUTING.md
- [ ] Add HEALTHCHECK to Dockerfile
- [ ] Apply staticcheck tagged switch suggestion in `errors.go`
- [ ] Remove dead `addError` method from `treeStats`

## 🟢 Testing

- [ ] Add integration test for 404 suggestions endpoint
- [ ] Add integration tests for HTTP endpoints
- [ ] Add integration tests for templates
- [ ] Add integration tests for full markdown → HTML pipeline
- [ ] Add renderer edge case tests
- [ ] Add container integration assertions
- [ ] Add template render benchmarks
- [ ] Write end-to-end tests
- [ ] Increase container package test coverage

## Resources

- See [CHANGELOG.md](./CHANGELOG.md) for completed items
- See [ROADMAP.md](./ROADMAP.md) for aspirational items
