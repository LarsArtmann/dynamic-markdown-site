# Dynamic Markdown Site - Agent Guidelines

---

## Project Overview

A type-safe, high-performance Go web server that converts markdown files into a navigable website with syntax highlighting, search, and caching.

**Key Technologies:**

- Go 1.27+ with modules (`GOEXPERIMENT=jsonv2` required — see gotcha 12)
- Standard `net/http` with Go 1.22+ method routing (no Gin)
- Goldmark + Chroma for markdown rendering with syntax highlighting
- Templ for type-safe HTML templates
- samber/do/v2 for dependency injection
- charm.land/log/v2 for structured logging (implements `slog.Handler`)
- golang.org/x/time/rate for rate limiting
- Otter/v2 for HTML caching
- gocloud.dev for blob storage (S3, GCS, filesystem)
- D2 + Mermaid for diagram rendering
- go-filewatcher/v2 for dev-mode file watching (replaces raw fsnotify)

---

## Essential Commands

### Build & Run

```bash
# Build the binary
go build -o dynamic-markdown-site ./cmd/dynamic-markdown-site

# Run in dev mode (live reload, no cache)
go run ./cmd/dynamic-markdown-site -dev -root ./content

# Run from S3/GCS
go run ./cmd/dynamic-markdown-site -storage-url s3://my-bucket/docs
```

### Nix

> **Note:** `--impure` and `NIXPKGS_ALLOW_UNFREE=1` are needed because the project uses a proprietary license.

```bash
# Build
NIXPKGS_ALLOW_UNFREE=1 nix build . --impure

# Run
NIXPKGS_ALLOW_UNFREE=1 nix run . --impure

# Dev shell (Go, golangci-lint, gopls, templ)
NIXPKGS_ALLOW_UNFREE=1 nix develop --impure
```

### Testing

```bash
go test ./... -race -cover
```

### Linting

```bash
golangci-lint run ./...
```

### Code Generation

```bash
# Generate Templ templates (required after editing .templ files)
templ generate

# Go tools (if any additional are added)
go mod tidy
```

---

## Project Structure

```
cmd/dynamic-markdown-site/   # main.go, watcher.go (go-filewatcher), healthcheck.go
internal/cache/              # Otter/v2 HTML cache (GetOrCompute)
internal/config/             # flags, env overrides, blob storage config
internal/container/          # samber/do/v2 wiring; accessors return (T, error)
internal/content/            # Repository impls: filesystem, blob, memory + search
internal/domain/             # URLPath, nodes, Frontmatter, Renderer, SuggestedPath
internal/renderer/           # Goldmark + Chroma + admonition/diagram extensions
internal/server/             # handlers, middleware, SSE live reload, metrics
internal/test/               # shared test fixtures (file helpers)
internal/version/            # version/commit injected via ldflags
templates/                   # layout.templ — run `templ generate` after edits
website/                     # Astro + Starlight docs site (dynamicmarkdown.lars.software)
```

Detailed architecture tree: [README.md](README.md#architecture).

---

## Code Patterns

### Dependency Injection

Uses `samber/do/v2`. Register providers in `container.New()`; resolve with `do.Invoke` (returns error) — never `do.MustInvoke`, which panics at runtime:

```go
cfg, err := do.Invoke[*config.Config](i)
if err != nil {
    return nil, errors.Wrap(err, "failed to resolve config")
}
```

The container's typed accessors (`Config()`, `Logger()`, `Repository()`, `Server()`) also return `(T, error)`; `main.go` handles those errors.

### Repository Pattern

Content stored in `internal/content/` behind the `content.Repository` interface (`Get`, `GetRaw`, `Root`, `Refresh`, `LastModified`, `AllPaths` — see `internal/content/repository.go`). Implementations:

- `FileSystemRepository` - reads from disk
- `BlobRepository` - reads from S3/GCS/Azure via gocloud.dev
- `InMemoryRepository` - for testing

### Domain Types

Domain types in `internal/domain/`:

- `URLPath` - validated URL paths (prevents traversal)
- `DirectoryNode` / `FileNode` - content nodes with hierarchy
- `NodeKind` - enum (directory/file)

### Error Handling

Uses `cockroachdb/errors` for wrapped errors with stack traces:

```go
return nil, errors.Wrap(err, "failed to create filesystem repository")
```

Sentinel errors defined at package level:

```go
var ErrContentNotFound = errors.New("content not found")
```

### Template Rendering

Uses `a-h/templ` for type-safe templates. After editing `.templ` files:

```bash
templ generate
```

Templates receive typed props structs:

```go
type FileViewProps struct {
    Layout LayoutProps
    File   *domain.FileNode
    TOC    []domain.TOCItem
}
```

### Logging

Uses `charm.land/log` which implements `slog.Handler`:

```go
logger := slog.New(logger)  // charmbracelet/log implements slog.Handler
```

Log levels via `-log-level` flag or `DYNAMIC_MARKDOWN_LOG_LEVEL` env var.

---

## Naming Conventions

| Element             | Convention                                    | Example                             |
| ------------------- | --------------------------------------------- | ----------------------------------- |
| Package names       | lowercase, single word                        | `cache`, `domain`, `server`         |
| Interface names     | PascalCase                                    | `Repository`, `ContentNode`         |
| Struct names        | PascalCase                                    | `FileNode`, `HTMLCache`             |
| Function names      | PascalCase (exported), camelCase (unexported) | `NewServer`, `handleContentByPath`  |
| Variable names      | camelCase                                     | `urlPath`, `searchResults`          |
| Constants           | PascalCase or SCREAMING_SNAKE                 | `idleTimeout`, `ErrContentNotFound` |
| Test files          | `*_test.go`                                   | `handlers_test.go`                  |
| Test functions      | `TestXxx`                                     | `TestHealthEndpoint`                |
| Benchmark functions | `BenchmarkXxx`                                | `BenchmarkRepositoryRefresh`        |

---

## Testing Patterns

### Test Helpers

```go
func newTestServer(t *testing.T, repo content.Repository) *Server {
    t.Helper()
    return NewServer(repo, content.NewSearcher(repo), slog.New(slog.DiscardHandler),
        cache.NewHTMLCache(100), renderer.NewGoldmarkRenderer(), false, "Test Site")
}
```

### HTTP Testing

Uses `net/http/httptest`:

```go
req := httptest.NewRequest(http.MethodGet, "/health", nil)
rec := httptest.NewRecorder()
router.ServeHTTP(rec, req)
```

### Test Tables

```go
tests := []struct { name string; path string; wantStatus int }{}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { /* test */ })
}
```

### Mock Repositories

Embed the interface to override only what you need (see `internal/test/` for ready-made fixtures):

```go
type FailingRepository struct{ content.Repository }
func (f *FailingRepository) Get(_ domain.URLPath) (domain.ContentNode, error) {
    return nil, content.ErrContentNotFound
}
```

### Parallel Tests

Use `t.Parallel()` in tests (required by paralleltest linter).

---

## Important Gotchas

### 1. URLPath Validation

All paths go through `domain.NewURLPath()` which prevents directory traversal:

```go
urlPath, err := domain.NewURLPath(filepath)
// Returns ErrInvalidPath if contains ".." or invalid chars
```

### 2. File Watching

File watcher only runs in dev mode (`-dev` flag). It uses `github.com/larsartmann/go-filewatcher/v2` for recursive watching with extension filtering (`.md`, `.markdown`), directory ignoring (`content.SkipDirs`), and 500ms global debounce. The watcher receives the SIGINT/SIGTERM context and exits cleanly on shutdown.

### 3. Cache Behavior

- Caching enabled by default
- Dev mode (`-dev`) disables caching automatically
- Cache invalidates on `/refresh` endpoint
- Rate limit: 10 refresh requests per minute per IP
- **Otter cache has background goroutines** — always call `cache.Close()` on shutdown or via `t.Cleanup()` in tests

### 4. Frontmatter Support

Markdown files support YAML frontmatter (`title`, `description`, `author`, `tags`, `draft` — see FEATURES.md for the full field table).

### 5. Templ Generation

After editing `templates/*.templ` files, run `templ generate` (CI fails on drift).

### 6. Hidden Files/Directories

Files/directories starting with `.` are ignored by the filesystem repository.

### 7. Graceful Shutdown

The server handles SIGINT/SIGTERM and waits up to 30 seconds for in-flight requests. `Server.Shutdown()` stops the rate limiter and closes the cache.

### 8. Templ Version Mismatch

The `templ` CLI version must match `go.mod`. If the CLI is newer, it generates code the library version doesn't understand (e.g., `templ.ResolveAttributeValue` undefined). Always run `go get github.com/a-h/templ@latest` after updating the CLI.

### 9. SSE Handler Blocks Without Cancellable Context

`handleSSE` enters an infinite loop blocking on `ctx.Done()`. `httptest.ResponseRecorder` implements `http.Flusher`, so the early-return path is NOT taken. Tests must use a cancellable context.

### 10. Rate Limiting Uses Token Bucket

Rate limiting uses `golang.org/x/time/rate` (token bucket). Burst semantics: the bucket starts full (`burst = maxRequests`) and refills at `window/maxRequests` tokens per second — a client may spend the whole quota up front, then trickle (decided & documented 2026-09-27). A sweep goroutine (1-minute tick) evicts visitors idle beyond a TTL of 3× the window, so the per-IP `visitors` map no longer grows unbounded (leak fixed 2026-09-27); `Server.Shutdown()` → `rateLimiter.Stop()` terminates the sweep, and `Stop()` is idempotent. Tests that assert **exact** allowed-counts MUST use a window long enough that no token refills during the test (the package uses the `newBurstOnlyLimiter(burst)` helper, which sets `burst = maxRequests` with a `time.Hour` window). With a short window like `time.Second`, the bucket refills on wall-clock time and goroutine-scheduling latency can admit one extra token, producing flaky off-by-one failures (e.g. 101 instead of 100). Eviction tests use `newRateLimiterWithSweepInterval` (fast sweep + short TTL) and either controlled `evictIdle(now)` calls or polling with a generous deadline — never tight wall-clock assertions.

### 11. GoReleaser License Metadata

**Fixed 2026-09-27:** `.goreleaser.yaml` previously claimed `license: MIT` in 4 places (homebrew_casks, nfpms, nix, scoops) while the `LICENSE` file is proprietary. Now: homebrew_casks/nfpms/scoops use SPDX `LicenseRef-Proprietary`, the nix section uses `unfree` (matching `flake.nix`), and `goreleaser check` validates green. Note the publisher-specific validation: the nix section only accepts nixpkgs license names (`unfree`), not SPDX `LicenseRef-` IDs.

### 12. encoding/json/v2 Is the Server Standard

Since commit `1fc8408` (2026-09-04) `internal/server` deliberately uses `encoding/json/v2`, which requires `GOEXPERIMENT=jsonv2` and a `go 1.27+` directive (kept in sync between `go.mod` and `.golangci.yml run.go`). All `go` commands that compile this repo must set `GOEXPERIMENT=jsonv2`. This supersedes the older guidance (httputil pinned at `v0.5.0`, stable `encoding/json` only); `httputil` is at `v1.2.0`.

### 13. Compression Middleware Affects Tests

`httputil.Compression` gzips responses >512 bytes when no `Accept-Encoding` header is set. The shared test helper `executeRequest` in `handlers_test.go` sets `Accept-Encoding: identity` to disable compression so tests can assert on plaintext response bodies.

### 14. golangci-lint Runs makezero in always Mode

`make([]T, n)` with n > 0 is forbidden (`makezero.always: true`). Build with `make([]T, 0, cap)` + `append`, `slices.Clone`, or the `curr = curr[:0]` + append pattern.

### 15. BuildFlow Findings Gate and pnpm-audit Skip

Full-mode buildflow fails on remaining severity error+ findings (`branching-flow`, `erraudit`, `go-structure-linter`). Suppress provably-safe findings with `//nolint:erraudit` (keep lines < 120 chars) or `//nolint:branching-flow` (untyped only; the typed `:panic` form breaks nolintlint, and the directive must be on the first line of multi-line calls).

`pnpm-audit` is excluded via `skip_steps` in `.buildflow.yml`: buildflow runs `pnpm audit` at the repo root where no lockfile exists (the JS project lives in `website/`, and `tool_paths` has no effect). Audit manually: `cd website && pnpm audit`.

---

## Configuration

### Flags

| Flag           | Default | Description                                                |
| -------------- | ------- | ---------------------------------------------------------- |
| `-port`        | 8080    | HTTP server port                                           |
| `-root`        | `.`     | Root directory with markdown files                         |
| `-storage-url` |         | Blob storage URL: `file://`, `s3://`, `gs://`, `azblob://` |
| `-log-level`   | info    | debug, info, warn, error                                   |
| `-cache`       | true    | Enable HTML caching                                        |
| `-dev`         | false   | Dev mode (no cache, file watching)                         |
| `-timeout`     | 30s     | Request timeout                                            |

### Environment Variables

Every flag has an env override: `DYNAMIC_MARKDOWN_` + uppercase flag name (`DYNAMIC_MARKDOWN_PORT`, `DYNAMIC_MARKDOWN_ROOT`, `DYNAMIC_MARKDOWN_STORAGE_URL`, `DYNAMIC_MARKDOWN_LOG_LEVEL`, `DYNAMIC_MARKDOWN_CACHE`, `DYNAMIC_MARKDOWN_DEV`, `DYNAMIC_MARKDOWN_TIMEOUT`), plus `DYNAMIC_MARKDOWN_SITE_NAME` (env only, no flag).

---

## HTTP API

| Endpoint           | Method   | Description                     |
| ------------------ | -------- | ------------------------------- |
| `/`                | GET      | Root directory view             |
| `/*path`           | GET      | Content (markdown or directory) |
| `/health`          | GET      | Health check                    |
| `/refresh`         | GET/POST | Refresh content (rate limited)  |
| `/search`          | GET      | Search content (`?q=query`, paginated, rate limited 30/min) |
| `/sitemap.xml`     | GET      | XML sitemap                     |
| `/robots.txt`      | GET      | Robots file                     |
| `/metrics`         | GET      | Prometheus-format metrics       |
| `/cache/stats`     | GET      | Cache statistics (JSON)         |
| `/static/*path`    | GET      | Static assets                   |
| `/api/live-reload` | GET      | SSE live reload (dev mode)      |

---

## Website build gotcha (pnpm 11)

Build-script approvals live in `website/pnpm-workspace.yaml` under `allowBuilds:` (`esbuild: true`) — pnpm v11 ignores `pnpm.*` in `package.json` and silently skips unapproved postinstall scripts, so `astro build` then fails on a missing esbuild binary. A placeholder value (e.g. `esbuild: set this to true or false`) silently disables the whole key (cmdguard incident, fixed 2026-09-19).

### 16. Watcher Ignore Filter Matches Any Path Component

`go-filewatcher`'s `FilterIgnoreDirs` matches a directory name anywhere in the event path (e.g. `tmp` matches `/tmp/x/…`), so running dev mode with `-root` under an ancestor named `tmp`, `temp`, `build`, `dist`, `vendor`, or `node_modules` silently watches nothing — every event is filtered out. Tests must create watch roots under the package directory, not `t.TempDir()` (surfaced by the watcher integration test, 2026-09-27).

### 17. Repository Tree Pointers Are Read Under Lock

`FileSystemRepository`/`BlobRepository` swap `r.tree` during `Refresh()` while HTTP readers traverse it. The shared helpers (`getFromTree`, `rootFromTree`, `allPaths`) take `**domain.ContentTree` and dereference under the read lock — always pass `&r.tree`, never `r.tree` (passing the value reads the pointer outside the lock; that race shipped undetected until the watcher integration test, fixed 2026-09-27).

---
