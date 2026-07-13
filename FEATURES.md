# Features

<!-- Last updated: 2026-07-13 -->

A complete catalog of everything Dynamic Markdown Site does — from the user's browser to the server's internals.

---

## Content Rendering

### Markdown Processing

Full [Goldmark](https://github.com/yuin/goldmark) pipeline with all extensions enabled:

- **GFM Tables** — standard pipe-delimited tables
- **Strikethrough** — `~~deleted text~~`
- **Task Lists** — `- [ ]` / `- [x]` interactive checkboxes
- **Definition Lists** — term followed by `: definition`
- **Footnotes** — `[^1]` with `[^1]: text` at bottom
- **Auto Heading IDs** — slugged anchors for deep linking (`#section-title`)
- **Linkify** — bare URLs auto-linked
- **Typographer** — smart quotes (`""` → `""`), dashes (`--` → `–`), ellipses (`...` → `…`)
- **YAML Frontmatter** — title, description, author, date, tags, draft flag

### Syntax Highlighting

[Chroma](https://github.com/alecthomas/chroma) with Monokai theme — 200+ languages, zero configuration.

````markdown
```go
func main() {
    fmt.Println("Hello, World!")
}
```
````

### Diagrams

| Engine                             | Rendering                          | Syntax                           |
| ---------------------------------- | ---------------------------------- | -------------------------------- |
| [D2](https://d2lang.com/)          | Server-side SVG                    | ` ```d2 ` fenced code block      |
| [Mermaid](https://mermaid.js.org/) | Client-side via Mermaid.js v11 CDN | ` ```mermaid ` fenced code block |

D2 diagrams are compiled to SVG at render time. Mermaid diagrams load the library on-demand — only when the page contains ` ```mermaid ` blocks.

### Admonition / Alert Blocks

GitHub-style callout blocks using blockquote syntax:

```markdown
> [!NOTE]
> Useful information that users should be aware of.

> [!WARNING]
> Critical content that needs extra attention.
```

Six alert types with distinct colors:

| Type        | Color       | Use case                     |
| ----------- | ----------- | ---------------------------- |
| `NOTE`      | Blue        | Informational notes          |
| `TIP`       | Green       | Helpful suggestions          |
| `IMPORTANT` | Purple      | Key information              |
| `WARNING`   | Amber       | Cautionary advice            |
| `CAUTION`   | Red         | Potential risks              |
| `CRITICAL`  | Intense red | Urgent/essential information |

Supports multiline content, inline formatting (bold, code, links), and lists inside alert blocks. Regular blockquotes without a `[!TYPE]` marker are unaffected.

### Table of Contents

Auto-generated from headings (h2+) with:

- Hierarchical nesting (parent/child items)
- URL-friendly anchor links
- Sidebar navigation in the file view

### Reading Time

Estimated at 200 words per minute, displayed in the article header.

---

## Navigation & Discovery

### Directory Browsing

Directory pages show a card grid with:

- File/folder icons (📄/📁)
- Title and type label
- Last modified date
- Sorted: directories first, then files, alphabetically

### Breadcrumbs

Header breadcrumb trail generated from the URL path, with the current page marked as active.

### Full-Text Search

- **Endpoint**: `/search?q=query`
- **Scoring**: exact title match (100%) > partial title (50%) > content body (30%)
- **Highlighting**: matches wrapped in `<mark>` tags
- **Snippets**: context window around each match with ellipsis
- **Results**: sorted by relevance score descending

### Smart 404 Pages

When a path is not found:

- Levenshtein distance calculates similar paths
- Score boosted for prefix and substring matches
- Up to 5 "Did you mean?" suggestions displayed
- Minimum score threshold (0.3) filters noise

---

## Developer Experience

### Live Reload

Dev mode (`-dev` flag) enables:

- **File watching** via fsnotify — monitors all `.md`/`.markdown` files recursively
- **500ms debounce** — batches rapid changes
- **SSE endpoint** (`/api/live-reload`) — browser receives reload events
- **Toast notifications** — connection status in the bottom-right corner
- **Auto-reconnect** — recovers from dropped connections

### Content Refresh

- **On-demand**: `GET` or `POST /refresh` reloads content from disk
- **Rate limited**: 10 requests per minute per IP
- **Cache invalidation**: refresh clears the entire HTML cache
- **File watcher**: dev mode auto-triggers refresh on changes

### Type-Safe Templates

[Templ](https://templ.guide/) compiles templates to Go code at build time:

- Compile-time HTML safety — no runtime template errors
- Typed props structs — IDE autocomplete, compiler checks
- No string concatenation for HTML — injection-safe by default

---

## Performance

### HTML Caching

[Otter](https://github.com/maypok86/otter) auto-tuning cache:

- **10,000 entries** capacity
- **1-hour access-based TTL** — popular pages stay cached
- **Atomic cache-or-render** — `GetOrCompute` prevents duplicate renders
- **Full invalidation** on content refresh
- **Stats**: hit ratio, requests, evictions

### Embedded Static Assets

CSS and favicon embedded in the binary via `//go:embed` — no external file dependencies at runtime.

### Static Binary

Compiled with `CGO_ENABLED=0`, `-tags netgo`, and static linking — zero runtime dependencies, runs on any Linux.

---

## Security

- **Path traversal prevention** — `URLPath` type rejects `..` and invalid characters at the type level
- **Static file traversal protection** — rejects `..` in static asset paths
- **HTML escaping** — Mermaid diagram content is escaped before rendering
- **Crypto-random request IDs** — 32-character hex from `crypto/rand`
- **Rate limiting** — prevents refresh endpoint abuse
- **Non-root container** — Docker image runs as UID 65532 (`nonroot`)
- **Distroless runtime** — minimal attack surface in production

---

## Configuration

| Option          | Flag           | Env Var                        | Default |
| --------------- | -------------- | ------------------------------ | ------- |
| Port            | `-port`        | `DYNAMIC_MARKDOWN_PORT`        | `8080`  |
| Content root    | `-root`        | `DYNAMIC_MARKDOWN_ROOT`        | `.`     |
| Storage URL     | `-storage-url` | `DYNAMIC_MARKDOWN_STORAGE_URL` |         |
| Log level       | `-log-level`   | `DYNAMIC_MARKDOWN_LOG_LEVEL`   | `info`  |
| Caching         | `-cache`       | `DYNAMIC_MARKDOWN_CACHE`       | `true`  |
| Dev mode        | `-dev`         | `DYNAMIC_MARKDOWN_DEV`         | `false` |
| Request timeout | `-timeout`     | `DYNAMIC_MARKDOWN_TIMEOUT`     | `30s`   |
| Site name       |                | `DYNAMIC_MARKDOWN_SITE_NAME`   | `Site`  |

Dev mode (`-dev`) automatically disables caching and enables file watching + live reload.

---

## HTTP API

| Endpoint           | Method   | Description                                   |
| ------------------ | -------- | --------------------------------------------- |
| `/`                | GET      | Root directory listing                        |
| `/*path`           | GET      | Markdown file or subdirectory listing         |
| `/health`          | GET      | Health check — returns `{"status":"healthy"}` |
| `/refresh`         | GET/POST | Reload content from source (rate limited)     |
| `/search`          | GET      | Full-text search — `?q=query`                 |
| `/static/*`        | GET      | Embedded static assets (CSS, favicon)         |
| `/sitemap.xml`     | GET      | XML sitemap for search engine crawlers        |
| `/robots.txt`      | GET      | robots.txt with sitemap reference             |
| `/metrics`         | GET      | Prometheus-format metrics                     |
| `/cache/stats`     | GET      | Cache hit/miss/eviction statistics (JSON)     |
| `/api/live-reload` | GET      | SSE stream for live reload (dev mode)         |

---

## Content Filtering

The filesystem repository automatically excludes:

- Hidden files and directories (`.` prefix)
- Common dependency/build directories: `node_modules`, `vendor`, `dist`, `build`, `tmp`, `temp`
- Non-markdown files (only `.md` and `.markdown` processed)
- Empty directories

---

## Frontmatter

Markdown files support YAML metadata:

```yaml
---
title: "Page Title"
description: "Page description"
author: "Author Name"
date: 2026-01-01
tags: ["go", "markdown"]
draft: false
---
```

| Field         | Effect                                               |
| ------------- | ---------------------------------------------------- |
| `title`       | Overrides filename as the page title                 |
| `description` | Meta description tag and directory card subtitle     |
| `author`      | Stored in metadata (available for templates)         |
| `date`        | Stored in metadata (available for templates)         |
| `tags`        | Stored in metadata (available for templates)         |
| `draft`       | When `true`, file is excluded from the site entirely |

---

## Infrastructure

### Docker

Single-stage distroless build:

- **Runtime**: `gcr.io/distroless/static-debian13:nonroot` — no shell, no package manager, minimal attack surface
- **Built-in healthcheck**: binary probes `/health` via a `healthcheck` subcommand (distroless has no shell/curl)
- **Non-root**: runs as UID 65532 (`nonroot` user)

```bash
docker build -t dynamic-markdown-site .
docker run -p 8080:8080 -v ./content:/content:ro dynamic-markdown-site
```

### CI Pipeline

Three GitHub Actions workflows:

| Workflow      | Purpose                                                                  | Triggers                                   |
| ------------- | ------------------------------------------------------------------------ | ------------------------------------------ |
| `test.yml`    | `go test -race -cover`, 75% coverage floor, `golangci-lint`, templ check | Go/Templ/go.mod changes                    |
| `docker.yml`  | Multi-arch Docker build & push to GHCR, Trivy scan, artifact attest      | Go/Templ/Dockerfile changes, `v*.*.*` tags |
| `release.yml` | GoReleaser: cross-compile, cosign signing, SBOM, Homebrew, Nix, Scoop    | `v*.*.*` tags                              |

### Graceful Shutdown

- Catches `SIGINT`/`SIGTERM`
- 30-second drain timeout for in-flight requests
- Clean shutdown of all services via DI container

---

## Architecture

### Dependency Injection

[samber/do/v2](https://github.com/samber/do) container wires all services:

- Config, Logger, Cache, Renderer, Repository, Searcher, Server
- Singleton lifecycle — each service created once
- Graceful shutdown propagates through the container

### Repository Pattern

`content.Repository` interface abstracts content storage:

- `FileSystemRepository` — production, reads from disk
- `BlobRepository` — production, reads from S3/GCS/Azure via gocloud.dev
- `InMemoryRepository` — testing, backed by a map

### Domain Types

- `URLPath` — validated URL path (directory traversal impossible by construction)
- `HTML` — type alias for pre-escaped HTML (distinguishes safe from unsafe)
- `DirectoryNode` / `FileNode` — content tree nodes with sorted children
- `RenderedFile` — immutable wrapper combining a file with its rendered output
- `ContentTree` — hierarchical tree with `Find()` and `AllPaths()`

### Error Handling

- [cockroachdb/errors](https://github.com/cockroachdb/errors) for stack traces and context
- Sentinel errors: `ErrContentNotFound`, `ErrInvalidPath`, `ErrInvalidRoot`, `ErrEmptyPath`
- Graceful degradation: D2 renderer failure → continues without diagram support
- Render fallback chain: template error → 500 page → plain text

---

## Testing

- **Unit tests** across all packages (`domain`, `config`, `content`, `renderer`, `server`, `cache`, `container`)
- **Parallel tests** where applicable (`t.Parallel()` enforced by linter)
- **Benchmarks** for content loading, rendering, search, and HTTP handlers
- **Test utilities** package (`internal/test`) with shared fixtures
- **Mock repositories** for isolated handler testing
- **~75 linters** configured in `.golangci.yml`
- CI runs with `-race` detector
