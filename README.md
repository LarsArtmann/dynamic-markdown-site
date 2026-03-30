# Dynamic Markdown Site

A type-safe, high-performance Go web server that converts markdown files into a navigable website with syntax highlighting, search, and caching.

## Features

- **Markdown Rendering** — Goldmark + Chroma for syntax highlighting
- **Type-Safe Templates** — Templ for type-safe HTML templates
- **Dependency Injection** — samber/do/v2 for DI
- **Structured Logging** — charm.land/log with slog support
- **Search** — Full-text content search
- **Caching** — HTML response caching with otter
- **Live Reload** — Browser auto-reload via SSE in development mode

## Quick Start

```bash
# Clone and run
git clone https://github.com/larsartmann/dynamic-markdown-site
cd dynamic-markdown-site
go run ./cmd/server -dev -root ./content

# Or build and run
go build -o site-generator ./cmd/server
./site-generator -port 8080 -root ./content
```

## Usage

```bash
./site-generator [flags]

Flags:
  -port 8080     HTTP server port
  -root "."      Root directory with markdown files
  -log-level info  debug, info, warn, error
  -cache          Enable HTML caching (default: true)
  -dev            Development mode (file watching, no cache)
  -timeout 30s    Request timeout
```

## API Endpoints

| Endpoint           | Method   | Description                        |
| ------------------ | -------- | ---------------------------------- |
| `/`                | GET      | Root directory view                |
| `/*path`           | GET      | Content (markdown or directory)    |
| `/health`          | GET      | Health check                       |
| `/refresh`         | GET/POST | Refresh content (rate limited)     |
| `/search`          | GET      | Search content (`?q=query`)        |
| `/static/*path`    | GET      | Static assets                      |
| `/api/live-reload` | GET      | SSE endpoint for live reload (dev) |

## Development

```bash
# Build
go build -o site-generator ./cmd/server

# Run tests
go test ./... -cover

# Run linter
golangci-lint run ./...

# Generate templates (after editing .templ files)
templ generate

# Run in dev mode with file watching
go run ./cmd/server -dev -root ./content
```

## Architecture

```
cmd/server/           # Entry point, file watcher
internal/
  ├── cache/          # HTML response caching
  ├── config/         # Configuration
  ├── container/      # Dependency injection
  ├── content/        # Content repository (filesystem, memory)
  ├── domain/         # Core domain types
  ├── renderer/       # Markdown to HTML
  ├── server/         # HTTP handlers, routing
  └── static/         # Static assets
templates/            # Templ HTML templates
```

## License

See [LICENSE](LICENSE) file for details.
