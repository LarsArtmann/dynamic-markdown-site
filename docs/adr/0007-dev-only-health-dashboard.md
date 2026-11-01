# 7. Dev-only health dashboard via go-health-dashboard

Date: 2026-11-01

## Status

Accepted (dev-only scope). Federation documented as the production answer.

## Context

ADR-0006 gave the server a real probe (go-health) whose verdict orchestrators
consume as JSON. Humans watching a dev server had no equivalent: verdicts are
visible only by curling `/readyz`.

The workspace maintains
[go-health-dashboard](https://github.com/larsartmann/go-health-dashboard) —
a templ + Datastar + SSE dashboard that renders go-health probes in real
time. It ships in two shapes:

1. **Embedded SDK**: `dashboard.New(probe, …)` mounts an HTML page + SSE
   stream in the same binary; the pusher reads `probe.CachedResponse()`
   in memory (no network hop).
2. **Federation hub** (`health-hub` binary): a standalone process that
   polls each service's go-health JSON document over HTTP
   (`HEALTH_HUB_REMOTES=name=url`), one card per remote, and injects a
   synthetic `reachable` check when a remote is down.

The cost asymmetry is deliberate upstream design: the dashboard module pulls
templ, templ-components, go-datastar, and go-sse; go-health itself stays
single-dependency. Embedding the dashboard in release binaries would tax
every deployment for a dev convenience.

## Decision

1. **Embedded SDK, dev builds only.** `go build -tags dev` mounts the
   dashboard at `/debug/health-dashboard` (HTML), `/debug/health-dashboard/sse`
   (push stream), and sibling favicon/datastar assets, via the same
   build-tag seam the live-reload route uses (`dashboard_dev.go` /
   `dashboard_prod.go`). Release binaries — Docker, goreleaser, nix — build
   without the dev tag, link zero dashboard code, and return 404 for the
   debug routes (verified by `go version -m` dependency comparison).
2. **Path collision guard.** The dashboard's default route set claims
   `/health` (HTML) and `/healthz`/`/readyz`/`/startupz` (its own JSON
   mirrors). Those paths belong to the probe contract (ADR-0005/0006), so
   dev wiring pins every dashboard route under `/debug/health-dashboard/…`.
   A dev-tagged test asserts `/health` keeps serving the readiness JSON
   beside a mounted dashboard.
3. **Version pin: go-health-dashboard v0.10.2.** It compiles against
   go-health v0.8.1 (its only probe usage is `CachedResponse`/`SanitizeResponse`)
   and its templ dependency (v0.3.1020) matches this repo's generated
   templates exactly (gotcha 8). v0.12.x adds empty-route disabling but
   forces templ v0.3.1070 and a template regeneration; not worth it for a
   dev-only page. Re-evaluate on the next templ bump.
4. **Federation is the production answer.** Operators who want a dashboard
   for deployed instances run `health-hub` pointed at each instance's
   `/health` document — no SDK, no dev tag, no new dependency in this repo.

## Consequences

- Dev workflows gain a live health view at `/debug/health-dashboard`
  (refresh cadence follows the probe sweep, `-probe-interval`, default 15s).
- go.mod gains the dashboard dependency; dev binaries carry templ/datastar
  in their link graph, release binaries do not. Nix vendorHash absorbs the
  new module (repaired via `buildflow -s nix-hash-fix` per gotcha 18).
- The dev pusher runs on a background context for the dev process lifetime;
  dev binaries are killed by the dev loop or Ctrl-C, so no explicit teardown
  is wired. If the dashboard ever graduates beyond dev, lifecycle moves into
  `Server.Shutdown()` first.
