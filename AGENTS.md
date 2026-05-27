# Dynamic Markdown Site - Agent Guidelines

**Version:** 1.0 | **Updated:** 2026-03-27

---

## Project Overview

A type-safe, high-performance Go web server that converts markdown files into a navigable website with syntax highlighting, search, and caching.

**Key Technologies:**

- Go 1.26.1 with modules
- Gin web framework for HTTP routing
- Goldmark + Chroma for markdown rendering with syntax highlighting
- Templ for type-safe HTML templates
- samber/do/v2 for dependency injection
- charm.land/log for structured logging

---

## Essential Commands

### Build & Run

```bash
# Build the server (Nix — preferred)
NIXPKGS_ALLOW_UNFREE=1 nix build --impure

# Build the server (Go directly)
go build -o dynamic-markdown-site ./cmd/dynamic-markdown-site

# Enter development shell (Go, gopls, golangci-lint, templ)
NIXPKGS_ALLOW_UNFREE=1 nix develop --impure

# Run in development mode (file watching, no caching)
go run ./cmd/dynamic-markdown-site -dev -root ./content

# Run with custom port
go run ./cmd/dynamic-markdown-site -port 3000 -root ./docs

# Production flags
-port 8080          # HTTP port
-root "."           # Root directory containing markdown files
-log-level debug    # debug, info, warn, error
-cache              # Enable response caching (default: true)
-dev                # Development mode (disables caching, enables file watching)
-timeout 30s        # Request timeout
```

### Nix

```bash
# Build binary
NIXPKGS_ALLOW_UNFREE=1 nix build --impure

# Enter dev shell
NIXPKGS_ALLOW_UNFREE=1 nix develop --impure

# Run all checks (format + build)
NIXPKGS_ALLOW_UNFREE=1 nix flake check --impure

# Format .nix files
nix fmt

# Update flake inputs
nix flake update
```

> **Note:** `--impure` and `NIXPKGS_ALLOW_UNFREE=1` are needed because the project uses a proprietary license.

### Testing

```bash
# Run all tests with coverage
go test ./... -cover

# Run tests with verbose output
go test ./... -v

# Run specific package tests
go test ./internal/server -v

# Run benchmarks
go test ./internal/content -bench=. -benchmem

# Run tests matching pattern
go test ./... -run "TestSearch"
```

### Linting

```bash
# Run golangci-lint (configured in .golangci.yml)
golangci-lint run ./...

# Run specific linter
golangci-lint run --enable=gocritic ./...

# The project uses ~75 linters including:
# - errcheck, errorlint, revive (error handling)
# - gocyclo, gocognit, funlen (complexity)
# - gosec (security)
# - testifylint, usetesting (testing)
# - paralleltest (parallel tests)
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
dynamic-markdown-site/
├── cmd/server/           # Application entry point
│   ├── main.go          # Main function, graceful shutdown
│   └── watcher.go       # File system watcher (dev mode)
├── internal/            # Private application code
│   ├── cache/          # HTML response caching (otter cache)
│   ├── config/          # Configuration from flags/env vars
│   ├── container/      # Dependency injection container (samber/do)
│   ├── content/        # Content repository (filesystem, memory)
│   ├── domain/         # Core domain types (DirectoryNode, FileNode, URLPath)
│   ├── renderer/       # Markdown to HTML (Goldmark + Chroma)
│   ├── server/         # HTTP handlers, routing, rate limiting
│   └── static/         # Static assets (CSS, favicon)
├── templates/          # Templ HTML templates
│   └── layout.templ    # Main layout, directory/file views, search
├── go.mod              # Go module definition
├── .golangci.yml       # Linter configuration (75 linters enabled)
└── .gitignore
```

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

File watcher only runs in dev mode (`-dev` flag). It refreshes the content repository when markdown files change.

### 3. Cache Behavior

- Caching enabled by default
- Dev mode (`-dev`) disables caching automatically
- Cache invalidates on `/refresh` endpoint
- Rate limit: 10 refresh requests per minute per IP

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

The server handles SIGINT/SIGTERM and waits up to 30 seconds for in-flight requests.

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

## Quality Gates

Before declaring complete:

- [ ] `go test ./...` passes
- [ ] `golangci-lint run ./...` passes (or documented exceptions in `.golangci.yml`)
- [ ] `templ generate` succeeds (if templates changed)
- [ ] `NIXPKGS_ALLOW_UNFREE=1 nix flake check --impure` passes (if nix files changed)
- [ ] `nix fmt -- --check` passes (if nix files changed)
- [ ] New code follows existing patterns
- [ ] Tests use `t.Parallel()` where applicable
