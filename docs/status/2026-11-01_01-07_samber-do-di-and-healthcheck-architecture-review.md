# samber/do v2 + Health Check Architecture Deep Review

> **RESOLVED 2026-11-01 01:30** — all HIGH/MED findings below were fixed and verified the same
> night; see [2026-11-01_01-30_samber-do-health-architecture-fixes-executed.md](2026-11-01_01-30_samber-do-health-architecture-fixes-executed.md)
> for the implementation record, verification matrix, and post-fix scorecard. The text below is
> the untouched point-in-time baseline.

|         |                                                                                                                                                                                                                                                                                   |
| ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Date    | 2026-11-01 01:07                                                                                                                                                                                                                                                                  |
| Scope   | DI container (`internal/container`), lifecycle wiring (`cmd/dynamic-markdown-site`), health checks (`internal/server`), shutdown behavior, samber/do v2.1.0 API utilization                                                                                                       |
| Method  | Full read of container/main/healthcheck/watcher/server/cache/blob sources; grep sweeps for every `do.*` API; verified dispatch behavior against samber/do v2.1.0 source (`service_eager.go`, `scope.go`) and pkg.go.dev; build + `go test -race` on container/server/cmd packages |
| Verdict | **DI fundamentals: superb. Health-check integration: not wired. Self-healing: broken by one hardcoded 200.**                                                                                                                                                                      |

---

## 1. Executive Verdict

**Question 1: Are we using samber/do v2 in combination with Health Checks superbly?**

**No.** The DI layer itself is exemplary (clean DO-1 → DO-6 sweep, zero findings). But the health-check story is disconnected from the container:

- `Server.HealthCheck(ctx)` (internal/server/handlers.go:101) **exactly matches** samber/do's `HealthcheckerWithContext` interface, so `injector.HealthCheckWithContext(ctx)` would sweep it — **but that sweep is never invoked anywhere**. The method's own doc comment admits it (handlers.go:99-100: "no in-process sweep invokes this method today").
- The `/health` HTTP endpoint **hardcodes** overall status `healthy` and HTTP 200 (handlers.go:129-137) even when `checkRepository()` reports an error. The orchestrator-facing probe (Docker `HEALTHCHECK` via the `healthcheck` subcommand, ADR-0005) exits 0 on any 200 — so **a wedged instance with an unreachable repository is never restarted**. This is the single most consequential finding.

**Question 2: Do we have a PROPER service-oriented, composable, resilient, and self-healing architecture?**

| Property             | Grade  | One-line justification                                                                                                                                                     |
| -------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Service-oriented     | **A**  | Single composition root, constructor injection everywhere, interface consumers (`content.Repository`, `domain.Renderer`, `content.Searchable`), zero service-locator smell |
| Composable           | **A−** | Graph is flat but clean; provider functions are pure and independently testable; `do.Package`/scopes unused but not needed at this size                                    |
| Resilient (shutdown) | **B−** | Graceful drain works and is tested; two resource-lifecycle leaks (blob bucket never closed, cache not container-owned) and a fragile double-shutdown                       |
| Self-healing         | **C−** | Restart-on-failure contract is defeated by the always-200 `/health`; DI health sweep dormant                                                                               |

Overall: **strong skeleton, weak nerve system.** The wiring to be superb already exists — three small, surgical changes (§7) close every gap.

---

## 2. Inventory: What Actually Exists

### 2.1 Composition root

`internal/container/container.go` — one `do.New()` (line 29), private injector, typed error-returning accessors:

| Service                          | Registration                                               | Lifetime        | Materialized at boot?          |
| -------------------------------- | ---------------------------------------------------------- | --------------- | ------------------------------ |
| `*config.Config`                 | `do.Provide` (container.go:32)                             | lazy singleton  | yes (everything depends on it) |
| `*slog.Logger`                   | `do.Provide` (container.go:33)                             | lazy singleton  | yes                            |
| `*cache.HTMLCache`               | `do.Provide` (container.go:34)                             | lazy singleton  | yes (server needs it)          |
| `*renderer.GoldmarkRenderer`     | `do.Provide` (container.go:35)                             | lazy singleton  | yes                            |
| `content.Repository` (interface) | `do.Provide` (container.go:36)                             | lazy singleton  | yes                            |
| `*content.Searcher`              | `do.Provide` (container.go:37)                             | lazy singleton  | yes                            |
| `*server.Server`                 | `provideServer()` + `do.ProvideValue` (container.go:42-46) | **eager** value | **yes — deliberately**         |

The eager server construction (container.go:39-46) is a **documented, correct** boot-criticality decision: main starts listening immediately, and the health sweep must see a constructed service from boot rather than reporting green for a never-resolved service. It also makes startup fail-fast — a bad `-storage-url` surfaces at boot (10s blob timeout, container.go:171), not at first request.

`main.go` resolves exactly four services via the container accessors (all `do.Invoke`-backed, error-propagating: main.go:79-116), builds the stdlib `http.Server`, and owns signal handling (main.go:65) + drain (main.go:176-190).

### 2.2 Lifecycle ownership map (as implemented)

| Resource                                 | Who closes it                                                                                             | samber/do-aware?                                         | Verified                                                              |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- | --------------------------------------------------------------------- |
| `http.Server` (listener, in-flight reqs) | `main.gracefulShutdown` → `httpServer.Shutdown` (main.go:180)                                             | n/a (not in container)                                   | shutdown_integration_test.go:100                                      |
| `*server.Server` (rate limiters + cache) | `main.shutdownServices` manually (main.go:193) **and** `injector.Shutdown()` via `do.Shutdowner` dispatch | yes (matches `Shutdowner`)                               | service_eager.go:133-139 dispatches `Shutdown()` for eager services   |
| `*cache.HTMLCache` (otter goroutines)    | only transitively via `Server.Shutdown()` (handlers.go:95)                                                | **no** (`Close()` ≠ `Shutdown()`)                        | otter `StopAllGoroutines` is `sync.Once`-guarded (cache_impl.go:1712) |
| `*BlobRepository` (gocloud bucket)       | **nobody in production code** — `Close()` at blob.go:286-287 is test-only                                 | **no** (`Close() error` matches no Shutdowner interface) | grep: only tests call it                                              |
| rate limiter sweep goroutines            | `Server.Shutdown()` → `rateLimiter.Stop()` (handlers.go:93-94)                                            | transitively                                             | `Stop()` is `sync.Once`-guarded (ratelimit.go:111-114)                |
| file watcher                             | `watchForChanges` defer `watcher.Close()` (watcher.go:40-44), ctx-cancelled                               | no (deliberately outside container, dev-only)            |                                                                       |
| `LiveReload` SSE clients                 | request-context cancellation only (livereload.go:96-103)                                                  | no (state-holding service unknown to container)          |                                                                       |

### 2.3 Health-check layers (as implemented)

| Layer                                         | Exists                    | Truthful                  | Wired to DI                                                  |
| --------------------------------------------- | ------------------------- | ------------------------- | ------------------------------------------------------------ |
| In-process DI sweep (`injector.HealthCheck*`) | ❌ never invoked          | —                         | would work today (Server matches `HealthcheckerWithContext`) |
| `Server.HealthCheck(ctx)` method              | ✅ (handlers.go:101-111)  | ✅ checks ctx + repo      | ⚠️ used only by its own unit tests                            |
| `/health` endpoint                            | ✅ (handlers.go:121-138)  | ❌ **always 200/healthy** | ❌ hand-rolled, ignores `HealthCheck()`                      |
| `healthcheck` subcommand (ADR-0005)           | ✅ (healthcheck.go:33-71) | trusts `/health`'s 200    | inherits the lie                                             |
| `/metrics`                                    | ✅ cache + uptime only    | n/a                       | no health metrics (metrics.go)                               |

---

## 3. samber/do Best-Practice Audit (DO-1 → DO-6)

Grep sweep over all `do.*` API usage (only two files import samber/do: `internal/container/container.go`, `container_test.go`).

| ID   | Rule                                         | Verdict      | Evidence                                                                                                                                                                                                                                                                                                                          |
| ---- | -------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DO-1 | No `Must*` in runtime paths                  | ✅ **Clean** | Zero occurrences of `MustInvoke`/`MustInvokeAs`/`MustShutdown` in the repo. Accessors use `do.Invoke` + `errors.Wrap` (container.go:52-69).                                                                                                                                                                                       |
| DO-2 | Every `do.New()` has a matching `Shutdown()` | ✅ **Clean** | Single `do.New()` (container.go:29); `Container.Shutdown()` (container.go:72-74) called on both the error path (main.go:71) and the graceful path (main.go:185); tests clean up via `t.Cleanup` (container_test.go:47). ⚠️ Caveat: the container _reports_ shutdown but cannot _own_ two of its resources — see §4 HIGH-2 / MED-2. |
| DO-3 | No `do.Override*` outside tests              | ✅ **Clean** | Zero `Override*` calls anywhere; tests compose fresh containers with `ProvideValue` instead (container_test.go:64-67) — the more honest pattern.                                                                                                                                                                                  |
| DO-4 | No global injector                           | ✅ **Clean** | Injector is a private struct field; no package-level `do.Injector`/`RootScope` variables; no `do.DefaultRootScope` usage.                                                                                                                                                                                                         |
| DO-5 | No `do.Invoke` inside loops                  | ✅ **Clean** | All invocations are hoisted provider-body/accessor calls.                                                                                                                                                                                                                                                                         |
| DO-6 | `Shutdown()` is self-contained               | ✅ **Clean** | `Server.Shutdown()` touches only its own fields (handlers.go:92-96); no injector access inside any service; no cross-service shutdown calls.                                                                                                                                                                                      |

Structural smells:

| Smell                                  | Verdict                                                                                                                                                                                                                             |
| -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Service-locator (injector passed deep) | ✅ absent — `do.Injector` appears only in provider signatures                                                                                                                                                                       |
| Injector stored inside a service       | ✅ absent                                                                                                                                                                                                                           |
| Compile-time interface guards          | ✅ `var _ Repository = (*BlobRepository)(nil)` (blob.go:25); ⚠️ no `var _ do.Shutdowner = …` / `var _ do.HealthcheckerWithContext = …` guards on Server/HTMLCache/BlobRepository — the lifecycle contracts are implicit and untested |
| Error propagation quality              | ✅ every provider wraps with context; sentinel `errBlobTimeout` defined at package level (container.go:20)                                                                                                                          |

**Score: 6/6 anti-pattern rules clean, 0 findings.** This is rare and worth saying plainly: the DI discipline in this repo is textbook.

---

## 4. Findings

Severity-ranked, each with evidence and concrete impact.

### 🔴 HIGH-1 — `/health` hardcodes success; self-healing is defeated

**Where:** internal/server/handlers.go:121-138

```go
dependencies := map[string]any{
    "repository": s.checkRepository(), // may report jsonStatusError
    "cache":      s.checkCache(),
}
writeJSON(w, http.StatusOK, map[string]any{
    jsonKeyStatus: jsonStatusHealthy, // ← hardcoded, ignores the checks above
    ...
```

`checkRepository()` (handlers.go:141-154) correctly reports `{"status": "error", ...}` when `repo.Root()` fails — but the overall status is the literal `jsonStatusHealthy` and the response code is unconditionally `http.StatusOK`.

**Impact chain:** Docker `HEALTHCHECK` runs the `healthcheck` subcommand (ADR-0005) → it exits 0 on any 200 (healthcheck.go:66-68) → Docker/K8s treat a repository-dead instance as healthy → **no restart, ever, for the only dependency that can fail**. `Server.HealthCheck(ctx)` — written precisely to answer this question — is bypassed.

**Also:** the cache sub-check (handlers.go:157-167) is not a health check at all; it always returns `healthy` plus utilization stats. Honest, but it labels stats as health.

### 🔴 HIGH-2 — Blob bucket is never closed; container shutdown is blind to it

**Where:** internal/content/blob.go:286-287 (`Close() error`); grep: no production caller.

samber/do dispatches shutdown **only** through its four lifecycle interfaces (`Shutdowner`, `ShutdownerWithError`, `ShutdownerWithContext`, `ShutdownerWithContextAndError` — verified in service_eager.go:114-139). `Close() error` matches none, so `injector.Shutdown()` skips the BlobRepository entirely: S3/GCS HTTP clients, credentials, and connection pools are never released. In production the process exits anyway (masking it); in long-lived/tested processes it accumulates open buckets.

This also means the DI container does **not** own a resource it created — a DO-2 caveat: the graph knows the repository exists but cannot tear it down.

### 🟡 MED-1 — The container health sweep is dormant plumbing

**Where:** missing `Container.HealthCheck(ctx)` accessor; `injector.HealthCheck*` never called (grep-verified).

samber/do v2.1.0 ships exactly the primitive this project needs: `Scope.HealthCheckWithContext(ctx) map[string]error` sweeps every service implementing `Healthchecker`/`HealthcheckerWithContext` (scope.go:289-310, verified). `*server.Server` already satisfies `HealthcheckerWithContext` (handlers.go:101: `HealthCheck(ctx context.Context) error`). One accessor + one call in the handler would unify health under the container. Instead, health logic is re-implemented ad hoc in the HTTP handler with a different (lying) aggregate. The method doc comment (handlers.go:98-100) documents its own dormancy.

### 🟡 MED-2 — Cache lifecycle is owned by the wrong service

**Where:** internal/cache/html.go:74-76 (`Close()`), handlers.go:95.

`HTMLCache` is a top-level container service with background goroutines, but it exposes no samber/do lifecycle method — its teardown depends entirely on `Server.Shutdown()` happening to hold a reference. Consequences: (a) the container cannot shut the cache independently; (b) any future wiring change that constructs a server without the container's cache leaks otter goroutines; (c) test helpers must remember `t.Cleanup(htmlCache.Close)` manually (container_test.go:50, shutdown_integration_test.go:25) — they already do, which is the smell.

### 🟡 MED-3 — Server is shut down twice; correctness rides on third-party idempotency

**Where:** main.go:192-199 calls `svc.server.Shutdown()` and then `svc.container.Shutdown()`, which dispatches `Shutdowner` on the same `*server.Server` again (verified: eager services dispatch through service_eager.go:133-139).

Safe **today** only because both callees are once-guarded: `rateLimiter.Stop()` uses `sync.Once` (ratelimit.go:111-114) and otter's `StopAllGoroutines()` uses `sync.Once` (cache_impl.go:1712-1720, verified in v2.3.0). Nothing in this repo documents or tests that invariant. The manual call also defeats samber/do's dependency-aware wave ordering — irrelevant in today's two-edge graph, but it means the shutdown sequence's real owner is "whatever idempotency happens to hold," not the composition root.

### 🟢 LOW-1 — Blob-provider timeout leaks the racing goroutine and its repository

**Where:** container.go:159-173. On timeout the provider returns `errBlobTimeout` while the goroutine may still complete `NewBlobRepository` — creating a bucket nobody closes (compounding HIGH-2). Bounded by process exit in prod; unbounded across repeated `container.New` attempts in tests/harnesses.

### 🟢 LOW-2 — Test helper type-lies: `*do.ShutdownReport` assigned to `err`

**Where:** container_test.go:34-41 (`assertNoShutdownError`). `err := c.Shutdown()` compiles because `*ShutdownReport` has `Error() string`, but the variable is not an `error`. Cosmetic; the check itself is correct.

### 🟢 LOW-3 — Health payload is `map[string]any`; "degraded" is unrepresentable

**Where:** handlers.go:124-137, 141-167. The status vocabulary is two free-string constants (`healthy`/`error`) scattered through ad-hoc maps. There is no type making invalid states unrepresentable — e.g. the renderer's silent graceful degradation (container.go:133-136, D2 unavailable) cannot be surfaced as `degraded`; it's invisible. `Server.HealthCheck()` returning a typed report (status enum + per-dependency results) would fix both this and HIGH-1 with the same edit.

### 🟢 LOW-4 — No health signal in `/metrics`

**Where:** metrics.go (cache counters + uptime only). A repository-unreachable counter or last-sweep-status gauge would let dashboards see what `/health` should be reporting.

### 🟢 LOW-5 — Observations (acceptable, recorded for completeness)

- `LiveReload` is a state-holding service (client map, livereload.go:19-23) invisible to the container; dev-only, request-context lifetime makes this fine. If it ever moves out of dev mode, it needs a lifecycle method.
- The file watcher deliberately lives outside the container (main.go:68, ctx-driven) — reasonable for a dev-only loop; the container graph therefore doesn't describe 100% of the process, which is a documented tradeoff, not an accident.
- `config.Load()` calls `flag.Parse` (forces the subprocess test pattern, container_test.go:18-31) — orthogonal to DI but worth knowing when composing containers in tests.

---

## 5. What Is Genuinely Good (keep exactly as-is)

1. **Fail-fast eager composition.** The 10s bounded blob timeout (container.go:150-173) converts an un-cancellable gocloud credential discovery into a boot-time, context-rich failure (`storage_url=` appended to the error).
2. **Graceful drain is real and tested.** In-flight request completion + post-shutdown refusal are asserted end-to-end (shutdown_integration_test.go:17-132), including the subtle `StateNew` vs in-handler race (comment at lines 87-89).
3. **Error-returning accessors all the way down.** `main` never panics on wiring; every resolution failure is wrapped with what was being resolved.
4. **The healthcheck subcommand design (ADR-0005)** is the right pattern for distroless — it only stops working because `/health` lies, not because the probe is wrong.
5. **Interface-first consumption.** Server depends on `content.Repository`, `content.Searchable`, `domain.Renderer` — not concrete impls — so filesystem/blob/memory swap without touching the server.

---

## 6. Architecture Snapshot (target vs current)

```
current                                     target (§7)
──────────────────────────────────────────  ──────────────────────────────────────────
/health ── hand-rolled checks ── 200 always   /health ── Container.HealthCheck(ctx)
   │                                             │        = injector sweep (Server ✓)
   └── healthcheck subcmd: 200 ⇒ healthy         └── 503 on any failure ⇒ restarts
injector.Shutdown()                             injector.Shutdown() (DAG wave)
   ├─ Server.Shutdown()  ← called TWICE          ├─ Server.Shutdown()   (once)
   ├─ HTMLCache          ← invisible             ├─ HTMLCache.Shutdown() (owned)
   └─ BlobRepository     ← invisible             └─ BlobRepository.Shutdown() (owned)
```

## 7. Prescribed Fixes (small, surgical, in order)

1. **Make `/health` truthful (HIGH-1).** In `handleHealth`, call `s.HealthCheck(r.Context())`; on error respond `http.StatusServiceUnavailable` with the wrapped error and `status: "unhealthy"`; keep version/uptime/dependency detail for observability. ~10 lines. This alone repairs the orchestrator contract (ADR-0005 subcommand needs no change — it already keys off non-200).
2. **Give the container lifecycle ownership (HIGH-2, MED-2).** Add `func (r *BlobRepository) Shutdown() error { return r.Close() }` (or rename, updating the interface doc); add `func (c *HTMLCache) Shutdown() { c.Close() }`. Add compile-time guards: `var _ do.ShutdownerWithError = (*BlobRepository)(nil)`, `var _ do.Shutdowner = (*HTMLCache)(nil)`, `var _ do.HealthcheckerWithContext = (*server.Server)(nil)`. samber/do then closes both during its DAG-wave shutdown with zero new orchestration code.
3. **Unify health under the container (MED-1).** Add `Container.HealthCheck(ctx) map[string]error` delegating to `c.injector.HealthCheckWithContext(ctx)`; have `handleHealth` consume it and merge per-dependency detail. Update the now-stale doc comment at handlers.go:98-100.
4. **Single shutdown owner (MED-3).** Delete the manual `svc.server.Shutdown()` in `shutdownServices` (main.go:193) — the injector already calls it in dependency order; the error path (main.go:71) keeps `shutdownServices` for the not-yet-served case.
5. **Optional polish:** type the health payload (status enum + typed dependency structs, LOW-3); bound the sweep with `do.InjectorOpts{HealthCheckGlobalTimeout: …}`; add a `repository_health` gauge (LOW-4); wrap the blob provider timeout path so the abandoned goroutine's repo, if constructed, is closed (LOW-1).

Estimated total diff for 1-4: well under 100 lines including tests; no interface breaks (Repository interface untouched — lifecycle stays out of the domain interface by design, which is correct).

## 8. Verification Performed

| Check                                                                             | Result                                                     |
| --------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `env -u GOTOOLCHAIN GOEXPERIMENT=jsonv2 go build ./...`                           | ✅ BUILD_OK                                                |
| `go test ./internal/container/... ./internal/server/... ./cmd/... -race -count=1` | ✅ ok (2.274s / 1.233s / 6.667s)                           |
| Grep: `MustInvoke                                                                 | do.Override                                                |
| Grep: `injector.HealthCheck` callers                                              | ✅ zero — sweep dormant (MED-1 confirmed)                  |
| Grep: `BlobRepository.Close` callers                                              | ✅ zero production hits (HIGH-2 confirmed)                 |
| samber/do v2.1.0 eager-shutdown dispatch (service_eager.go:114-139)               | ✅ verified: only `Shutdown*` method names trigger         |
| samber/do v2.1.0 `Scope.HealthCheckWithContext` (scope.go:289-310)                | ✅ verified: sweeps all `Healthchecker*` services          |
| otter v2.3.0 `StopAllGoroutines` (cache_impl.go:1712)                             | ✅ `sync.Once` — double-shutdown safe today (MED-3 caveat) |

## 9. Scorecard

| Dimension                               | Grade  | Weight of evidence                                                               |
| --------------------------------------- | ------ | -------------------------------------------------------------------------------- |
| Composition root & wiring purity        | **A**  | §3: 6/6 DO rules clean                                                           |
| Composability & service boundaries      | **A−** | §5.5: interface-first consumers; flat but honest graph                           |
| Resilience (drain, fail-fast, timeouts) | **B+** | §5.1-5.2; minus double-shutdown fragility                                        |
| Lifecycle completeness                  | **C+** | §4 HIGH-2, MED-2, MED-3: 2 of 4 resource-holders invisible to the container      |
| Health-check integration                | **D**  | §4 HIGH-1, MED-1: plumbing exists, endpoint bypasses it                          |
| Self-healing (orchestrator contract)    | **C−** | §4 HIGH-1: always-200 defeats ADR-0005's own intent                              |
| **Overall**                             | **B**  | Excellent DI skeleton; the health nerve system needs ~100 lines to become superb |

## 10. Open Work

Prescribed fixes §7.1-§7.4 are ready to execute as a single bounded task (suggest title: "Wire DI health sweep + truthful /health + container-owned cache/blob shutdown"). Track in `TODO_LIST.md` per docs-health convention; ADR-0005 needs no amendment — the subcommand contract is correct and becomes fully effective the moment `/health` stops hardcoding 200.
