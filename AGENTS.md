# Dynamic Markdown Site - Agent Guidelines

---

## Project Overview

A type-safe, high-performance Go web server that converts markdown files into a navigable website with syntax highlighting, search, and caching.

**Key Technologies:**

- Go 1.26.3 with modules
- Standard `net/http` with Go 1.22+ method routing (no Gin)
- Goldmark + Chroma for markdown rendering with syntax highlighting
- Templ for type-safe HTML templates
- samber/do/v2 for dependency injection
- charm.land/log for structured logging
- golang.org/x/time/rate for rate limiting
- Otter for HTML caching
- gocloud.dev for blob storage (S3, GCS, filesystem)
- D2 + Mermaid for diagram rendering

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

---

## Code Patterns

### Dependency Injection

Uses `samber/do/v2`. Register providers in `container.New()`:

```go
func New() (*Container, error) {
    injector := do.New()
    do.Provide(injector, provideConfig)
    do.Provide(injector, provideLogger)
    // ...
}
```

Access via typed accessors: `do.MustInvoke[*config.Config](c.injector)`

### Repository Pattern

Content stored in `internal/content/` with interface:

```go
type Repository interface {
    Get(path domain.URLPath) (domain.ContentNode, error)
    GetRaw(path domain.URLPath) (*RawFile, error)
    Root() (*domain.DirectoryNode, error)
    Refresh() domain.RefreshResult
    LastModified() time.Time
    AllPaths() []domain.URLPath
}
```

Implementations:

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
    return NewServer(repo, content.NewSearcher(repo), slog.New(slog.DiscardHandler), cache.NewHTMLCache(100))
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

```go
type FailingRepository struct{}
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

File watcher only runs in dev mode (`-dev` flag). It refreshes the content repository when markdown files change with 500ms debounce to coalesce bulk operations.

### 3. Cache Behavior

- Caching enabled by default
- Dev mode (`-dev`) disables caching automatically
- Cache invalidates on `/refresh` endpoint
- Rate limit: 10 refresh requests per minute per IP
- **Otter cache has background goroutines** — always call `cache.Close()` on shutdown or via `t.Cleanup()` in tests

### 4. Frontmatter Support

Markdown files support YAML frontmatter:

```yaml
---
title: "Page Title"
description: "Page description"
author: "Author Name"
tags: ["tag1", "tag2"]
draft: false
---
```

### 5. Templ Generation

After editing `templates/*.templ` files, run:

```bash
templ generate
```

### 6. Hidden Files/Directories

Files/directories starting with `.` are ignored by the filesystem repository.

### 7. Graceful Shutdown

The server handles SIGINT/SIGTERM and waits up to 30 seconds for in-flight requests. `Server.Shutdown()` stops the rate limiter and closes the cache.

### 8. Templ Version Mismatch

The `templ` CLI version must match `go.mod`. If the CLI is newer, it generates code the library version doesn't understand (e.g., `templ.ResolveAttributeValue` undefined). Always run `go get github.com/a-h/templ@latest` after updating the CLI.

### 9. SSE Handler Blocks Without Cancellable Context

`handleSSE` enters an infinite loop blocking on `ctx.Done()`. `httptest.ResponseRecorder` implements `http.Flusher`, so the early-return path is NOT taken. Tests must use a cancellable context.

### 10. Rate Limiting Uses Token Bucket

Rate limiting uses `golang.org/x/time/rate` (token bucket). No background goroutines. `Stop()` is a no-op kept for API compatibility.

### 11. GoReleaser License Mismatch

The `.goreleaser.yaml` declares `license: MIT` in 4 places (homebrew_casks, nfpms, nix, scoops sections), but the `LICENSE` file is proprietary and `flake.nix` correctly uses `licenses.unfree`. This is a **pre-existing inconsistency** that causes Homebrew/Scoop/Nix to publish wrong license metadata. (The previous `archives.format_overrides` and `brews` deprecations mentioned here have already been fixed — `formats: ["zip"]` and `homebrew_casks` are now used.)

### 12. encoding/json/v2 Must Not Be Used

The project intentionally uses stable `encoding/json`. The `go-error-family` dependency must stay pinned at `v0.6.1` (v0.7.0+ adopts `encoding/json/v2`). Automated upgrade tools (go-auto-upgrade) will re-break the build by migrating imports — the migration must be excluded. `GOEXPERIMENT=jsonv2` is NOT enabled.

### 13. Compression Middleware Affects Tests

`httputil.Compression` gzips responses >512 bytes when no `Accept-Encoding` header is set. The shared test helper `executeRequest` in `handlers_test.go` sets `Accept-Encoding: identity` to disable compression so tests can assert on plaintext response bodies.

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

Prefix: `DYNAMIC_MARKDOWN_`

| Variable                       | Description      |
| ------------------------------ | ---------------- |
| `DYNAMIC_MARKDOWN_PORT`        | Server port      |
| `DYNAMIC_MARKDOWN_ROOT`        | Root directory   |
| `DYNAMIC_MARKDOWN_STORAGE_URL` | Blob storage URL |
| `DYNAMIC_MARKDOWN_LOG_LEVEL`   | Log level        |
| `DYNAMIC_MARKDOWN_CACHE`       | Enable caching   |
| `DYNAMIC_MARKDOWN_DEV`         | Dev mode         |
| `DYNAMIC_MARKDOWN_TIMEOUT`     | Request timeout  |
| `DYNAMIC_MARKDOWN_SITE_NAME`   | Site name        |

---

## HTTP API

| Endpoint           | Method   | Description                     |
| ------------------ | -------- | ------------------------------- |
| `/`                | GET      | Root directory view             |
| `/*path`           | GET      | Content (markdown or directory) |
| `/health`          | GET      | Health check                    |
| `/refresh`         | GET/POST | Refresh content (rate limited)  |
| `/search`          | GET      | Search content (`?q=query`)     |
| `/sitemap.xml`     | GET      | XML sitemap                     |
| `/robots.txt`      | GET      | Robots file                     |
| `/metrics`         | GET      | Prometheus-format metrics       |
| `/cache/stats`     | GET      | Cache statistics (JSON)         |
| `/static/*path`    | GET      | Static assets                   |
| `/api/live-reload` | GET      | SSE live reload (dev mode)      |

---
