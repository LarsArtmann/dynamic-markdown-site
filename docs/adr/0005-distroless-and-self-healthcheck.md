# 5. Distroless image with self-contained healthcheck

Date: 2026-06-14

## Status

Accepted.

## Context

Docker's `HEALTHCHECK` instruction requires a runnable command inside
the container. Traditional healthchecks shell out to `curl`, `wget`, or
a small script — but our base image is `gcr.io/distroless/static-debian13`,
which contains no shell, no `curl`, and no `wget`. The distroless
`static` variant is preferred for security (smaller attack surface,
fewer CVEs) but it cannot satisfy a `HEALTHCHECK` instruction written
in the conventional way.

## Decision

We extend the binary with a `healthcheck` subcommand that:

1. Accepts `--addr` (default `localhost:8080`) and `--timeout` (default 5s)
2. Sends an HTTP `GET /health` to that address
3. Exits 0 on a 200 response, 1 otherwise (with a wrapped error
   explaining the failure)

The Dockerfile's `HEALTHCHECK` uses it directly:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/dynamic-markdown-site", "healthcheck", "--addr", "localhost:8080"]
```

This keeps the runtime image distroless (no shell, no extra binaries)
while satisfying Docker's healthcheck contract.

## Consequences

Positive:

- Runtime image stays minimal: distroless/static, ~10 MB compressed
- No new vulnerabilities introduced by a shell or by `curl`/`wget`
- The healthcheck command is testable as a regular Go function
  (`runHealthcheck`), so it is covered by unit tests

Negative:

- The healthcheck subcommand is a public surface of the binary; it
  must remain stable across releases
- If we ever migrate to a non-distroless image we should remove the
  subcommand and use the standard `curl` / `wget` instead
- The flag parsing logic in `runHealthcheck` must tolerate the literal
  `"healthcheck"` subcommand token (so the subcommand can be invoked
  directly without it)

## Alternatives considered

- **Distroless base + `wget` or `curl` static binary** — adds a
  ~3 MB static binary and a new attack surface for no functional
  benefit
- **Switch to `gcr.io/distroless/base-debian13`** — includes
  `/bin/sh` but not `curl`/`wget`; still need a custom healthcheck
  tool
- **Drop the HEALTHCHECK directive** — operators lose the ability to
  detect a wedged process; orchestrators fall back to TCP probes
  which don't validate that the server is actually serving
