# samber/do v2 + Health Checks: Fixes Executed & Re-Review

|          |                                                                                                                                                                                                                                                                                                                                                                                    |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Date     | 2026-11-01 01:30                                                                                                                                                                                                                                                                                                                                                                   |
| Scope    | DI container (`internal/container`), lifecycle wiring (`cmd/dynamic-markdown-site`), health checks + SSE shutdown (`internal/server`), cache/blob lifecycle (`internal/cache`, `internal/content`)                                                                                                                                                                                 |
| Method   | Independent re-audit (grep sweeps of every `do.*` API, read of all touched sources, verification of samber/do v2.1.0 + otter v2.3.0 dispatch behavior in module cache) followed by **execution and verification** of the fixes prescribed in [the 01:07 review](2026-11-01_01-07_samber-do-di-and-healthcheck-architecture-review.md), plus two additional findings fixed on sight |
| Baseline | Build OK, full `go test ./... -race` green, `golangci-lint run` 0 issues, `golangci-lint config verify` OK — **before** the fixes, reproduced independently                                                                                                                                                                                                                        |
| Result   | All HIGH and MED findings from the baseline review **fixed and tested**; DI + health architecture now matches its own documentation                                                                                                                                                                                                                                                |

---

## 1. Executive Verdict

**Question 1: Are we using samber/do v2 in combination with Health Checks superbly?**

**Now: yes, for the core contract.** The baseline review's verdict ("DI superb, health not wired, self-healing broken by one hardcoded 200") was confirmed by independent re-audit and then _repaired_:

- `/health` is now **truthful**: 503 + `unhealthy` when the repository probe fails; the Docker `HEALTHCHECK` → `healthcheck` subcommand → restart chain (ADR-0005) finally has teeth.
- The container now **owns its lifecycle**: blob bucket and otter cache are shut down through samber/do's `ShutdownerWithError`/`Shutdowner` dispatch, enforced by compile-time guards in the composition root.
- The dormant DI health sweep is **wired and tested**: `Container.HealthCheck(ctx)` sweeps the injector and returns the server's verdict.
- Shutdown has a **single owner**: the injector's DAG wave; the manual double-shutdown of the server is gone.

**Question 2: Do we have a PROPER service-oriented, composable, resilient, self-healing architecture?**

| Property             | Before | After  | What changed                                                                                                                                                                   |
| -------------------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Service-oriented     | A      | **A**  | Unchanged; composition root purity intact, no DI imports leaked into internal packages                                                                                         |
| Composable           | A−     | **A**  | Server registered via `do.Provide` + eager invoke — the graph now records real dependency edges (was a detached `ProvideValue` leaf)                                           |
| Resilient (shutdown) | B−     | **A−** | Blob + cache lifecycle owned by the container; single shutdown owner; SSE clients released before drain (was: open tab blocked shutdown 30 s in dev, endpoint exposed in prod) |
| Self-healing         | C−     | **A−** | Truthful `/health` (503 on dead repo); container health sweep live; Docker restart contract effective                                                                          |

---

## 2. What Was Implemented

### Fix 1 — Truthful `/health` (was HIGH-1)

`internal/server/handlers.go` — `handleHealth` now derives the aggregate from the single canonical probe (`Server.HealthCheck(r.Context())`, which satisfies `do.HealthcheckerWithContext`):

- healthy → `200` + `"status":"healthy"` (payload shape unchanged for existing consumers)
- probe failure → **`503`** + `"status":"unhealthy"` + top-level `error` detail; the repository dependency entry carries the wrapped error
- `checkRepository()` was removed; the repository detail (`repositoryHealth`) is _derived from_ the probe, so `/health` performs exactly **one** repository check per request (was two)
- stale doc comment ("no in-process sweep invokes this method today") replaced with the new truth
- **A test was codifying the bug**: `TestHealthEndpoint`/`TestHealthEndpointStructure` asserted `200 healthy` against a `FailingRepository`. Both now use a healthy fixture; the failing-repo behavior is asserted by `TestHealthEndpointReportsUnhealthyRepository` (503 + unhealthy + repository error).

Impact chain repaired: repository dies → `/health` 503 → `healthcheck` subcommand exits 1 → Docker `HEALTHCHECK` (retries=3) marks unhealthy → orchestrator restarts. ADR-0005 needed no amendment — it already keyed off non-200.

### Fix 2 — Container-owned resource lifecycle (was HIGH-2 + MED-2)

- `internal/content/blob.go`: `BlobRepository.Shutdown() error` added (delegates to `Close()`; the idiomatic name stays for direct callers). Previously `Close() error` matched no samber/do interface, so **the gocloud bucket was never released on shutdown**.
- `internal/cache/html.go`: `HTMLCache.Shutdown()` added (delegates to `Close()`). Previously the cache was invisible to the injector and closed only as a side effect of `Server.Shutdown()`.
- `internal/container/container.go`: compile-time guards added in the **composition root**, not next to the types — a deliberate improvement over the baseline prescription, keeping `internal/cache` and `internal/content` free of DI imports while making a lost lifecycle method a build failure where wiring happens:

```go
var (
    _ do.ShutdownerWithError      = (*content.BlobRepository)(nil)
    _ do.Shutdowner               = (*cache.HTMLCache)(nil)
    _ do.HealthcheckerWithContext = (*server.Server)(nil)
)
```

### Fix 3 — Live container health sweep (was MED-1)

`Container.HealthCheck(ctx) map[string]error` delegates to `injector.HealthCheckWithContext(ctx)`. Empirically verified against v2.1.0 (`scope.go` `asyncHealthCheckWithContext` iterates **every** invoked service; non-healthcheckers report `nil` — the godoc's "services that implement the Healthchecker interface" phrasing is misleading). The accessor's doc states the real contract: _any non-nil value means unhealthy; all nil means healthy._ Two tests cover both sides (`TestHealthCheck_SweepsInjectedServices`, `TestHealthCheck_ReportsUnhealthyRepository` with a `Root`-only failing repository double).

### Fix 4 — Single shutdown owner (was MED-3)

`main.go` `shutdownServices` no longer calls `svc.server.Shutdown()` before `container.Shutdown()`. The injector dispatches `Server.Shutdown()` itself (it matches `do.Shutdowner`). Correctness no longer rides on undocumented third-party idempotency (`sync.Once` in the rate limiters, otter's once-guarded `StopAllGoroutines` — both verified in module source).

### Fix 5 — Honest eager server registration (new, improves the graph)

`container.New()` previously hand-ran `provideServer(injector)` and registered the result with `do.ProvideValue` — a **detached leaf**: samber/do held no dependency edges for the server, so the DAG-wave shutdown ordered nothing and the graph lied about the architecture. Now the server is registered with `do.Provide(injector, provideServer)` and materialized eagerly via `do.Invoke` in `New()`:

- boot-critical fail-fast preserved (a bad `-storage-url` still dies at boot, 10 s blob timeout)
- the graph records server → {repo, searcher, logger, cache, renderer, config} edges, so the shutdown wave orders consumers before providers
- **bonus DO-2 edge fixed**: if construction fails mid-graph, `New()` now shuts the partially-built injector down instead of leaking resolved resources (the old code returned `nil, err` with the injector dangling)

### Fix 6 — SSE shutdown stall + prod-exposed dev endpoint (new finding, fixed)

Found during the resilience review, absent from the baseline report:

- `GET /api/live-reload` was registered **unconditionally**; `handleSSE` blocks until the client disconnects. `http.Server.Shutdown` waits for active connections, so any SSE client — an open browser tab in dev, or anyone on the internet in prod — held graceful shutdown until the full 30 s deadline, after which `main` exited with an error. An open dev tab made every Ctrl-C take 30 s and end in "application failed".
- Fixes: (a) the route is now gated on `devMode` (matching what FEATURES/AGENTS always claimed — "SSE live reload (dev mode)"), so production cannot accumulate SSE clients; (b) `LiveReload.Close()` disconnects all clients (idempotent, `sync.Once`-guarded, mutex-safe against `Notify`) and `handleSSE` exits via a `closed` channel; (c) `main.gracefulShutdown` closes the hub _before_ `httpServer.Shutdown` — drain coordination belongs to the composition root that owns the `http.Server`; (d) `Server.Shutdown()` also closes the hub as belt-and-braces.
- Tests: `TestLiveReloadCloseReleasesHandler` (the handler's request context is never cancelled and the recorder implements `Flusher`, so **only** `Close` can release it — the exact trap documented in AGENTS.md gotcha 9), `TestLiveReloadCloseIsIdempotentAndSafeAfterClose` (no panic sending to closed channels), `TestLiveReloadEndpointDisabledOutsideDevMode` (404 in prod).

---

## 3. Verification Matrix

| Gate               | Command                                                 | Result                                                                                                                                        |
| ------------------ | ------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Build              | `env -u GOTOOLCHAIN GOEXPERIMENT=jsonv2 go build ./...` | ✅ OK                                                                                                                                         |
| Full suite         | `go test ./... -race -cover`                            | ✅ all packages ok (server 90.3%, cache 92.6%, content 77.7%, container 53.5% — container coverage up from baseline with the new sweep tests) |
| Lint               | `golangci-lint run ./...`                               | ✅ 0 issues                                                                                                                                   |
| Config (strict)    | `golangci-lint config verify`                           | ✅ OK (gotcha 18)                                                                                                                             |
| DO-1→DO-6 re-sweep | grep `MustInvoke                                        | Override                                                                                                                                      |
| samber/do dispatch | module source v2.1.0 (`service_eager.go`, `scope.go`)   | ✅ shutdown + healthcheck dispatch semantics verified empirically (probe test), not assumed                                                   |
| otter double-close | module source v2.3.0 `StopAllGoroutines`                | ✅ returns `false` on second call — container + server close paths coexist safely                                                             |

Behavioral checks after the fix: healthy container sweep → all `nil`; unhealthy repo sweep → server entry wraps `repository unreachable`; `/health` on failing repo → 503/`unhealthy`/error detail; `/api/live-reload` → 404 when `devMode=false`, registered when `true`; double container shutdown still clean (`TestContainerLifecycle`).

---

## 4. Remaining Open Work (deliberately not done here)

| Item                                                                                                                                                                | Why deferred                                                                                                                                                     | Where          |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| Typed health payload (status enum, typed dependency structs) — "degraded" is still unrepresentable; renderer's silent D2 degradation (container.go) stays invisible | API-shape change touching consumers; the wire contract just changed once (503) — let it settle                                                                   | baseline LOW-3 |
| Repository health gauge in `/metrics`                                                                                                                               | Prometheus-format metric addition; orthogonal to the DI wiring                                                                                                   | baseline LOW-4 |
| Blob-provider timeout leaks the racing goroutine's repo if it completes after the 10 s deadline                                                                     | Prod impact bounded by process exit; fix is a `context`-aware or closable race                                                                                   | baseline LOW-1 |
| `do.InjectorOpts{HealthCheckGlobalTimeout}` bound for the sweep                                                                                                     | Single service today; meaningful when more services implement `Healthchecker`                                                                                    | baseline §7.5  |
| Dev-mode drain latency after the fix                                                                                                                                | SSE clients are released instantly on shutdown now, but the _browser_ reconnect loop can race the 30 s drain; acceptable in dev, observe before engineering more | new, residual  |

---

## 5. Scorecard (after execution)

| Dimension                               | Grade  | Evidence                                                                                                      |
| --------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------- |
| Composition root & wiring purity        | **A**  | 6/6 DO rules clean; DI imports confined to `internal/container`; guards live in the root                      |
| Composability & service boundaries      | **A**  | Full graph edges recorded; interface-first consumption unchanged                                              |
| Resilience (drain, fail-fast, timeouts) | **A−** | SSE no longer stalls drain; single shutdown owner; fail-fast boot preserved                                   |
| Lifecycle completeness                  | **A−** | 4/4 resource-holders container-visible or explicitly owned (http.Server by main, watcher ctx-scoped dev-only) |
| Health-check integration                | **A−** | One canonical probe, served over HTTP and swept by the DI container; typed payload remains                    |
| Self-healing (orchestrator contract)    | **A−** | Dead repo → 503 → restart; residual gap is metrics-only                                                       |

**Overall: B → A−.** The architecture now does what its documentation claims. The remaining distance to A is polish (typed payload, metrics), not correctness.
