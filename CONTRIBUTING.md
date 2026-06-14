# Contributing

Thanks for your interest in contributing to **Dynamic Markdown Site**!

## How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/your-feature`)
3. Make your changes
4. Verify tests and lint pass (see below)
5. Open a pull request against `master`

Please keep pull requests focused on a single change. Larger refactors should
be discussed in an issue first.

## Development Setup

The project uses [Nix flakes](https://nixos.wiki/wiki/Flakes) for reproducible
development environments. From the project root:

```bash
# Enter a dev shell with Go, golangci-lint, gopls, and templ
NIXPKGS_ALLOW_UNFREE=1 nix develop --impure

# Build
nix build --impure  # or: go build ./cmd/dynamic-markdown-site

# Run the server in dev mode against ./content
go run ./cmd/dynamic-markdown-site -dev -root ./content

# Probe the health endpoint (used by Docker HEALTHCHECK)
./result/bin/dynamic-markdown-site healthcheck --addr localhost:8080
```

## Code Quality

Before opening a pull request, all of the following must pass:

```bash
# Run the full test suite (unit + integration) with race detector + coverage
go test ./... -race -cover -coverprofile=coverage.out

# Run the linter (must produce zero issues)
golangci-lint run ./...

# Format .nix files (if you touched Nix configuration)
nix fmt
```

The CI pipeline (`.github/workflows/test.yml`) enforces these checks
alongside a 75% coverage threshold. The `test.yml` workflow also runs
`templ generate` and fails the build if the generated files are not
checked in — run `templ generate` and commit the result whenever you
change a `*.templ` file.

## Project Conventions

- **Package names**: lowercase, single word (`cache`, `domain`, `server`).
- **Types**: PascalCase structs; avoid `Manager`/`Handler`/`Helper` suffixes.
- **Functions**: PascalCase exported, camelCase unexported.
- **Errors**: wrap with `cockroachdb/errors` and add context. Sentinel
  errors live at package scope (`ErrContentNotFound`).
- **Logging**: `charm.land/log` via `*slog.Logger`. Never log secrets or
  full request bodies.
- **Domain types**: see `internal/domain/`. New domain concepts belong
  there, not in `internal/server/` or `internal/content/`.
- **Tests**: use `t.Parallel()` where possible, share helpers via
  `internal/test/`. HTTP tests use `httptest`.

## Reporting Issues

Please use [GitHub Issues](https://github.com/larsartmann/dynamic-markdown-site/issues)
to report bugs or request features. Include:

- A minimal reproduction (markdown content + steps to reproduce)
- Expected vs actual behaviour
- Version (`dynamic-markdown-site -version` or the commit SHA)

## Reporting Security Issues

Please do **not** file public issues for suspected security vulnerabilities.
Instead, follow the process in `SECURITY.md` (if present) or contact the
maintainer privately.

## License

By contributing, you agree that your contributions will be licensed under
the project's existing license (see [LICENSE](./LICENSE)).
