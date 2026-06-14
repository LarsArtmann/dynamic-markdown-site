# 2. Use Go stdlib `net/http` only

Date: 2026-06-14

## Status

Accepted.

## Context

The project exposes an HTTP API for static content, search, refresh,
and a health endpoint. Go 1.22+ ships a fully-featured router
(`http.ServeMux`) supporting method routing, path variables, and
wildcards. The standard library also provides everything needed for
middleware composition, including context plumbing, timeouts, and
graceful shutdown.

Third-party routers (Chi, Gin, Echo, Fiber) offer more features on
paper (richer middleware ecosystems, request validation, OpenAPI
generators) at the cost of:

- A sizeable dependency surface (each is 200+ transitive deps)
- Cognitive overhead for new contributors who must learn the router's
  idioms
- Lock-in to the chosen router's middleware and error model

## Decision

We use **only the Go standard library** for HTTP serving. Specifically:

- `http.ServeMux` for routing
- Handlers implement `http.Handler` directly
- Middleware is composed manually with a small `chain` helper
- Request ID, recovery, and compression are taken from the local
  `larsartmann/httputil` package — a thin wrapper around stdlib
- Timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`,
  `IdleTimeout`) are set on `http.Server` directly
- Graceful shutdown is implemented via `http.Server.Shutdown` with a
  `context.WithTimeout`

## Consequences

Positive:

- Zero non-stdlib HTTP dependencies; supply-chain footprint stays small
- Handlers are easy to test with `net/http/httptest`
- Performance is on par with the best third-party routers for our
  workload (static file serving, low-cardinality JSON)
- New contributors already know the API

Negative:

- We cannot use third-party middleware (auth, CORS, rate limiting
  libraries) without porting or vendoring them. We accept this for
  the dependencies we currently need; we can revisit if the need
  grows.
- Some convenience features (automatic `OPTIONS` handling, declarative
  route schemas) are not available. We work around them with
  middleware and explicit 405/415 responses where needed.

## Alternatives considered

- **Chi** — small and stdlib-friendly, but still a dependency.
- **Gin** — large dependency footprint, opinionated API.
- **Fiber** — fasthttp-based, not net/http compatible.
