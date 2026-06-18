# Dynamic Markdown Site - Agent Guidelines

**Version:** 1.1 | **Updated:** 2026-06-07

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

### Nix

> **Note:** `--impure` and `NIXPKGS_ALLOW_UNFREE=1` are needed because the project uses a proprietary license.

### Testing

### Linting

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
    Root() (*domain.DirectoryNode, error)
    Refresh() domain.RefreshResult
    LastModified() time.Time
}
```

Implementations:

- `FileSystemRepository` - reads from disk
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

### 11. GoReleaser Config Deprecations

`goreleaser check` fails on deprecated keys. Two known ones in this project:

- `archives.format_overrides[].format` → renamed to `formats` (accepts a list, e.g. `formats: ["zip"]`)
- `brews` → fully deprecated since v2.16. Use `homebrew_casks` instead. Key differences:
  - Directory must be `Casks/` (not `Formula/`)
  - `install:` block is replaced by `binaries:` (auto-installed from archive)
  - `test:` block has no cask equivalent — use `caveats:` for user-facing notes
  - First release may need a `tap_migrations.json` entry in the tap repo to redirect users from the old formula

---

## Configuration

### Flags

| Flag         | Default | Description                        |
| ------------ | ------- | ---------------------------------- |
| `-port`      | 8080    | HTTP server port                   |
| `-root`      | `.`     | Root directory with markdown files |
| `-log-level` | info    | debug, info, warn, error           |
| `-cache`     | true    | Enable HTML caching                |
| `-dev`       | false   | Dev mode (no cache, file watching) |
| `-timeout`   | 30s     | Request timeout                    |

### Environment Variables

Prefix: `DYNAMIC_MARKDOWN_`

| Variable                     | Description     |
| ---------------------------- | --------------- |
| `DYNAMIC_MARKDOWN_PORT`      | Server port     |
| `DYNAMIC_MARKDOWN_ROOT`      | Root directory  |
| `DYNAMIC_MARKDOWN_LOG_LEVEL` | Log level       |
| `DYNAMIC_MARKDOWN_CACHE`     | Enable caching  |
| `DYNAMIC_MARKDOWN_DEV`       | Dev mode        |
| `DYNAMIC_MARKDOWN_TIMEOUT`   | Request timeout |

---

## HTTP API

| Endpoint        | Method   | Description                     |
| --------------- | -------- | ------------------------------- |
| `/`             | GET      | Root directory view             |
| `/*path`        | GET      | Content (markdown or directory) |
| `/health`       | GET      | Health check                    |
| `/refresh`      | GET/POST | Refresh content (rate limited)  |
| `/search`       | GET      | Search content (`?q=query`)     |
| `/static/*path` | GET      | Static assets                   |

---
