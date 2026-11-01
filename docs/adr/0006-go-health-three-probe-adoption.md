# 6. go-health three-probe adoption

Date: 2026-11-01

## Status

Accepted. Supersedes the endpoint wiring described in ADR-0005 (the distroless
`healthcheck` subcommand itself remains valid and unchanged).

## Context

Until 2026-11-01 the server exposed a single hand-rolled `/health` endpoint
that conflated two different questions: "is this process serving?" and "are
the dependencies reachable?". A repository blip returned 503 (after the
2026-11-01 truthfulness fix) even though the process was fine, with no notion
of degraded-but-serving states, and per-dependency detail was ad-hoc JSON
assembled in the handler.

The workspace maintains [go-health](https://github.com/larsartmann/go-health)
— a Kubernetes health-probe SDK built specifically for samber/do v2
containers: it sweeps the DI injector, classifies services as critical or
non-critical, caches verdicts on a refresh interval, and serves the
conventional three probe routes. Reinventing that in this repo duplicated a
first-party library.

A decision was also needed for partial capability: the D2 diagram renderer is
optional (markdown renders fine without it), and operator dashboards could not
see that degradation.

## Decision

1. Adopt `github.com/larsartmann/go-health` v0.8.1.
2. The content repository is the **only critical** service: readiness (`/readyz`)
   and the `/health` alias return 503 exactly when the repository check fails.
3. The diagram renderer is a **non-critical** check: when D2 is unavailable it
   reports a failure that classifies the overall verdict as `warn` with
   HTTP 200 — degraded, still serving, visible in the payload.
4. `GET /health` remains as a readiness alias. ADR-0005's `healthcheck`
   subcommand (exit 0 on 200, exit 1 otherwise) is unchanged and now probes a
   truthful verdict end to end.
5. Readiness verdicts are cached by the probe (15 s refresh) so orchestrator
   probes never fan out into the storage backend.
6. `/api/live-reload` is compiled out of non-dev binaries via the `dev` build
   tag (decision recorded 2026-11-01); the route, hub, and SSE code path do
   not exist in Docker/goreleaser/nix release artifacts.

## Consequences

Positive:

- Liveness and readiness are decoupled: a repository blip removes the pod from
  rotation (503) without restarting the process; the process is only restarted
  for genuine crashes.
- Degradation is observable (`warn` + per-check error detail) without lying
  about health.
- Per-service health comes from the DI graph itself (repositories and renderer
  implement `do.HealthcheckerWithContext`), so adding a service with a health
  check automatically extends the probe.
- The health payload vocabulary is the RFC health-check format (`pass`, `warn`,
  `fail`) shared with the rest of the fleet.

Negative:

- `/health` payload shape changed: `status` moved from `healthy`/`unhealthy`
  to `pass`/`warn`/`fail`; `dependencies` became `checks`; `commit` and
  `build_date` are no longer in the payload (they remain in startup logs and
  `/metrics`). Consumers parsing the old fields must migrate.
- Readiness is cached (15 s): a freshly-broken repository takes up to 15 s to
  flip the verdict — bounded staleness in exchange for probe isolation.
- Probe routes bypass the site middleware chain (no request IDs, access-log
  entries, or compression for `/health*`); k8s probes do not need them, and
  probe noise stays out of the access log.
- Dev builds diverge from release builds by the `dev` build tag; CI must run
  the tagged test suite (`go test -tags dev ./internal/server/`) or the SSE
  path rots untested.

## Alternatives considered

- **Keep the hand-rolled `/health` and add a `/readyz`** — two health systems
  drifting apart; the workspace already owns a better one.
- **Liveness that checks the repository** — restart cascades on storage blips;
  the exact failure mode go-health's three-probe split exists to prevent.
- **Classify the D2 renderer as critical** — a missing optional binary would
  take the whole instance out of rotation for a cosmetic degradation.
