# Session Status: Verification of the 01:56 Fixes — Panic Found, Fixed, Process Honesty

|                    |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Date               | 2026-11-01 02:11                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| Session scope      | Continuation of the 01:55 status session: asked the 3 (g) questions, then independently verified the parallel session's 01:56 fixes per the user's answer                                                                                                                                                                                                                                                                                                                                                                                            |
| Deliverables       | 1 defect found + fixed (`LiveReload` double-close panic), 2 report annotations, index row                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Verification       | Affected tests ×5 under `-race` ok; full `go test ./... -race` 8/8 packages ok; `golangci-lint run ./internal/server` 0 issues                                                                                                                                                                                                                                                                                                                                                                                                                       |
| Status             | Verification verdict: **8/9 claim areas confirmed, 1 critical defect found and fixed**                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ✅ Decisions 02:20 | User answered §g: **(1) Indefinite stale** — any cached content may serve indefinitely during a repository outage; never-cached paths → 503. **(2) Keep 503** — `/health` keeps failing during outages (orchestrator restart contract unchanged); stale serving is user-facing mitigation only. **(3) Re-verify default** — sessions always re-verify other sessions' green-gate claims; no lock protocol. These three answers are the seed content for ADR-0006 (Tier-1 #1) and bound design #2: stale layer is additive to the truthful-503 probe. |

---

## The three questions answered first

### What did I forget?

1. **To run `git status` BEFORE asking my (g) questions.** The status-report skill puts git status inside the report step; I followed that order literally. Result: my 01:55 report and my 3 questions were written while a parallel session was actively landing the fixes — Q2 ("fix now or harvest only?") was **obsolete at delivery time**, and my "all defects open" framing needed a correction annotation within 3 minutes of writing. Cheap ordering fix, real cost.
2. **Two of their artifacts went unread during verification.** The parallel diff contains a second report (`…samber-do-health-architecture-fixes-executed.md`, 128 lines) plus AGENTS.md/FEATURES.md edits and full test diffs — I reviewed the core source changes and the new tests' heads, but never opened that second report or the doc edits. My "8/9 verified" verdict therefore covers code paths, not their documentation claims.
3. **The `TestLiveReloadCloseReleasesHandler` test is timing-dependent.** The handler goroutine may not have registered its channel before `Close()` runs, in which case the test passes vacuously (it never exercised the interleaving that panics). My fix is correct, but the test does not _deterministically_ cover the bug I fixed — it caught it under `-race` scheduling luck, and would pass again even if someone reintroduced the channel-closing loop. Deterministic coverage is unregistered work.
4. **Full-package lint after my fix.** I re-linted only `./internal/server/`; the full `./...` lint ran before my one-file change, so the combination is almost certainly fine but the invocation chain isn't airtight.
5. **The user's Q2 answer ("200 degraded + stale cache") is an unregistered architecture decision.** It now lives only in this conversation. No ADR, no TODO entry — exactly the memory-loss pattern AGENTS.md warns about.

### What could I have done better?

1. **Check for concurrent sessions before producing anything.** One `git log --since=...` at the start would have revealed the 01:56 landing and changed my report, my questions, and my framing from the first line. Ordering discipline over cleverness.
2. **Made the regression test deterministic while I was in there.** I fixed the bug but left its only test racy — half the job. The right move: force registration-before-Close (test hook or a registration signal) so the interleaving is guaranteed, not probabilistic.
3. **Bound the verification claim.** "8/9 verified" was informal numbering over their 14-item list. A verification report should table claim → evidence → verdict one-to-one, so gaps (the unread report, unchecked doc claims) are visible instead of averaged away.
4. **Run the binary, not just the tests.** The 503 contract is unit/integration-verified; a real process boot with a broken repository and `curl /health` would close the loop end-to-end. Not done — the test pyramid's top step is missing.

### What could I still improve (ongoing)?

1. Adopt **claim→evidence tables** as the standard shape for any verification verdict.
2. Treat **any "gates green" claim from any session (including mine) as unverified until the exact invocations are re-run** — this incident is the third gate-masking event in this repo's history (gotcha 18 lineage) and deserves a cross-project lesson entry.
3. When a user decision arrives mid-session (stale-cache preference), **register it immediately** (ADR draft + TODO) instead of letting it ride in conversation memory.

---

## a) FULLY DONE

| Item                                                                                                                                                                                                                          | Evidence                                                                                                                                                                                                                                           |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 3 blocking questions asked and answered (deployment target, self-healing policy, verify-or-trust)                                                                                                                             | User: "not deployed yet", "200 degraded + stale cache", "yes, verify"                                                                                                                                                                              |
| Full diff review of the parallel session's core fixes (18 files, +991/−42)                                                                                                                                                    | handlers.go (503 via single probe), container.go (guards, sweep accessor, eager `do.Provide` + error-path `injector.Shutdown`), blob.go/cache.go (`Shutdown()` delegation), main.go (single shutdown owner, SSE-first drain), livereload.go, tests |
| **Critical defect found:** `LiveReload.Close()` closed client channels that `handleSSE`'s own `defer close(clientChan)` also closes → `panic: close of closed channel`, crashing the server package under the full race suite | Panic trace: livereload.go:149 via `TestLiveReloadCloseReleasesHandler.func1`; reproduced in `go test ./... -race` run at ~02:00                                                                                                                   |
| Defect fixed with the same single-owner principle the parallel session applied to shutdown                                                                                                                                    | internal/server/livereload.go:40-56 — `Close()` now only signals `closed` + clears the map; each handler's deferred close is the sole closer of its channel                                                                                        |
| Fix verified: stress ×5 on affected tests under `-race`; full suite `go test ./... -race` 8/8 packages ok; `golangci-lint run ./internal/server` 0 issues                                                                     | shell outputs 02:05-02:10                                                                                                                                                                                                                          |
| Both status reports reconciled non-destructively: my 01:55 report annotated (fixes landed, items superseded), their 01:56 report annotated (verification failure + fix), both indexed                                         | header rows in `docs/status/2026-11-01_01-55…` and `…_01-56…`; README.md index                                                                                                                                                                     |

## b) PARTIALLY DONE

1. **Independent verification** — code paths of all 9 claim areas reviewed and executed; open: the second fixes-report (`…fixes-executed.md`), AGENTS.md/FEATURES.md edits, full test-diff bodies, and the "one repository probe per request" claim's dedicated assertion were not individually checked. Effort: S.
2. **Regression coverage for the double-close bug** — bug fixed and stress-covered, but the covering test remains interleaving-dependent (passes vacuously when registration loses the race). Effort: S.
3. **Full-lint chain** — full `./...` lint green pre-fix, package-level post-fix; the strict chain (full lint after last change) wasn't re-run. Effort: S.
4. **CI confirmation** — daemon commits (including my fix) not watched through CI; no `go.mod` change so no `nix flake check` obligation per gotcha 18. Effort: S.
5. **User-decision registration** — "stale-cache self-healing" answered but not yet written into ADR/TODO. Effort: S.

## c) NOT STARTED (observed this round, untouched)

1. Stale-cache serving design + implementation (user's chosen self-healing strategy).
2. ADR for the health-semantics evolution: 503-truthful probes today vs 200-degraded + stale content later; how the subcommand/orchestrator contract survives that change.
3. Deterministic regression test for the LiveReload close interleaving.
4. E2E process probe: boot with broken repo → `curl /health` → expect 503.
5. HARVEST of both 01:55 and 01:56 (f) sections into `TODO_LIST.md`/`ROADMAP.md` (now three timestamped reports carry overlapping open work).
6. Cross-project lesson entry: "never trust claimed-green without re-running the exact invocations" (crush-config `references/lessons.md`, by commit).
7. Re-run of the 01:07 scorecard once the dust settles (D→A claim check for Health/Self-healing grades).

## d) TOTALLY FUCKED UP

1. **The parallel session shipped a shutdown panic behind a green-gates claim.** `golangci-lint` + a partial test run masked a defect that the full `-race` suite catches immediately — the same gate-masking failure mode as gotcha 18's "three red pushes", now with a fourth instance. Severity: dev-facing (panic in handler goroutine on shutdown with an open dev tab; net/http recovers it, so noisy logs + killed connection, not a crash — but the fix's own test couldn't deterministically prove the fix). Mitigation: **fixed and verified this session** (livereload.go:40-56).
2. **My own miss, named honestly:** I produced a 166-line status report plus a 3-question batch that a single `git log` would have materially changed. The report needed a correction annotation before the daemon even committed it. Not data-loss-grade, but it is exactly the "stale-on-arrival" output this workflow is supposed to prevent.
3. **Decision entropy:** user decisions and open work now live in three timestamped reports + one conversation, with overlapping 50-item lists and no deduped backlog. Until HARVEST runs, "what should we do next" has three inconsistent answers. Mitigation: item (c)5.

## e) WHAT WE SHOULD IMPROVE

1. **Ordering: git-first.** Run `git log`/`git status` before writing any report or asking any questions — concurrent-session awareness is now a demonstrated failure mode, not hypothetical.
2. **Verification verdicts as claim→evidence tables.** Every claimed item gets its own row with the exact invocation or file:line that proves it; unverifiable claims get "not checked" rows instead of being silently dropped.
3. **Deterministic test for deterministic bugs.** Any time a race detector catches an interleaving bug, the accompanying regression test must force that interleaving, not hope for it.
4. **Register mid-session user decisions instantly** (ADR draft + TODO row) — conversation memory is not project memory.
5. **Gate claims must cite invocations.** Extend gotcha 18's discipline to status reports: "green" without the exact command list is treated as "unverified".
6. **Deduped backlog over per-report lists.** One TODO_LIST fed by HARVEST beats N timestamped 50-item brainstorms that drift apart.

## f) Up to 50 things we should get done next

Carry-forwards from 01:55/01:56 remain open unless marked done by the 01:56 landing. Tier 1 = actionable now; Tier 2 = roadmap fuel. Impact / Effort (S<30m, M<2h, L>2h) / Category.

**Tier 1 — actionable now**

| #  | Task                                                                                                                                                                       | Impact   | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Write ADR-0006: health truth ownership **including** the user's stale-cache decision (probe vs content semantics)                                                          | High     | M      | Documentation |
| 2  | Design stale-cache serving on repository failure: which paths, how stale, what never-cached paths return                                                                   | Critical | L      | Feature       |
| 3  | Make `TestLiveReloadCloseReleasesHandler` deterministic (force registration before `Close`)                                                                                | High     | S      | Bug           |
| 4  | Add a regression test asserting no panic when `Close()` races an active SSE handler (the exact fixed bug)                                                                  | High     | S      | Bug           |
| 5  | HARVEST all three reports' (f) sections into `TODO_LIST.md`/`ROADMAP.md`, deduped                                                                                          | High     | M      | Documentation |
| 6  | Register the "re-verify claimed-green gates" lesson into crush-config `references/lessons.md` (by commit)                                                                  | Medium   | S      | Process       |
| 7  | Read + fact-check the unread artifacts of the 01:56 diff: `…fixes-executed.md`, AGENTS.md/FEATURES.md edits                                                                | Medium   | S      | Quality       |
| 8  | Verify the "exactly one repository probe per request" claim has a dedicated assertion                                                                                      | Medium   | S      | Quality       |
| 9  | Blob-provider 10s-timeout leak: close the abandoned repo if the racing goroutine succeeds (open since 01:07 review)                                                        | Medium   | M      | Bug           |
| 10 | Failure-path test: `healthcheck` subcommand exits non-zero on non-200 (still zero coverage)                                                                                | High     | S      | Bug           |
| 11 | Integration test: `injector.Shutdown()` closes the blob bucket                                                                                                             | Medium   | M      | Quality       |
| 12 | Integration test: `injector.Shutdown()` stops otter goroutines                                                                                                             | Medium   | M      | Quality       |
| 13 | Rename `err` → `report` in `assertNoShutdownError` (container_test.go)                                                                                                     | Low      | S      | Cleanup       |
| 14 | Watch CI green on the daemon commits incl. the LiveReload fix                                                                                                              | Medium   | S      | Process       |
| 15 | Re-run full `golangci-lint run ./...` after the livereload fix (strict chain)                                                                                              | Low      | S      | Quality       |
| 16 | E2E probe: boot binary with broken repo, `curl /health` → 503                                                                                                              | Medium   | M      | Quality       |
| 17 | Document the LiveReload single-owner channel contract in AGENTS.md gotcha 21 (extend the existing sentence)                                                                | Low      | S      | Documentation |
| 18 | Emit `Retry-After` on `/health` 503                                                                                                                                        | Low      | S      | Feature       |
| 19 | `Notify` after `Close`: decide + document behavior for late SSE registrations during drain (currently immediate exit via closed channel — likely fine, just write it down) | Low      | S      | Documentation |
| 20 | Check EventSource auto-reconnect behavior against `Close()` (residual noted by 01:56 session, still unobserved)                                                            | Low      | M      | Quality       |

**Tier 2 — roadmap fuel**

| #  | Task                                                                                                                                                    | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 21 | Typed health payload: status enum incl. `degraded` (prerequisite for the stale-cache contract)                                                          | High   | M      | Feature       |
| 22 | `/healthz` vs `/readyz` split once a deployment target exists (user: none today — deprioritized)                                                        | Medium | M      | Feature       |
| 23 | `/metrics`: repository health gauge + last-sweep status/timestamp                                                                                       | Medium | M      | Feature       |
| 24 | `do.InjectorOpts{HealthCheckGlobalTimeout}` bound                                                                                                       | Medium | S      | Quality       |
| 25 | Bounded `ShutdownWithContext` aligned with the 30s drain window                                                                                         | Medium | M      | Quality       |
| 26 | Per-service `ShutdownReport` errors logged structured                                                                                                   | Low    | S      | Quality       |
| 27 | Startup health sweep before `ListenAndServe`; abort boot on unhealthy deps                                                                              | Medium | M      | Feature       |
| 28 | Sweep coverage test: assert exactly the expected healthcheckers are probed                                                                              | Medium | S      | Quality       |
| 29 | Golden-file test for `/health` JSON contract                                                                                                            | Low    | S      | Quality       |
| 30 | Retry/backoff around `repo.Root()` in the sweep for transient blob errors                                                                               | Medium | M      | Feature       |
| 31 | Soak test: random handler/Notify/Close churn under `-race` for LiveReload                                                                               | Low    | L      | Quality       |
| 32 | Confirm `FileSystemRepository` needs no closer; document in Repository interface doc                                                                    | Low    | S      | Documentation |
| 33 | Dockerfile `--start-period=10s` vs blob 10s open timeout — verify margin                                                                                | Medium | S      | Bug           |
| 34 | E2E test: SIGTERM → SSE release → drain → injector shutdown ordering                                                                                    | Medium | M      | Quality       |
| 35 | Custom lint rule: forbid hardcoded health-status aggregates in handlers                                                                                 | Low    | L      | Quality       |
| 36 | Unify `jsonStatus*`/`jsonKey*` into a typed vocabulary                                                                                                  | Low    | M      | Cleanup       |
| 37 | otter v2.4.0 bump evaluation (v2.3.0 current)                                                                                                           | Low    | M      | Cleanup       |
| 38 | `do.Package` grouping of providers (config/logger vs core)                                                                                              | Low    | L      | Cleanup       |
| 39 | `samber-do-auditlog` hooks for lifecycle observability                                                                                                  | Medium | M      | Feature       |
| 40 | Dev-only debug endpoint exposing `do.ExplainInjector` graph                                                                                             | Low    | M      | Feature       |
| 41 | Per-service `do.HealthCheck[T]` probes in the debug surface                                                                                             | Low    | L      | Feature       |
| 42 | Website docs sync: `/health` 503 semantics on dynamicmarkdown.lars.software                                                                             | Low    | S      | Documentation |
| 43 | Post-stabilization: re-run the 01:07 scorecard; verify Health D→A / Self-healing C−→A actually hold                                                     | Medium | M      | Documentation |
| 44 | Retry-After/Retry hint in SSE stream on reconnect (pairs with #20)                                                                                      | Low    | S      | Feature       |
| 45 | Rate-limit policy review for `/health` (must stay unthrottled — document why)                                                                           | Low    | S      | Documentation |
| 46 | Assess `/health` info disclosure (version/commit on public deploys)                                                                                     | Low    | S      | Security      |
| 47 | Session-process rule: concurrent-session check (`git log --since`) becomes step 0 of every report/review                                                | Medium | S      | Process       |
| 48 | Claim→evidence table template for verification verdicts (skill or checklist)                                                                            | Medium | S      | Process       |
| 49 | Consider `context.Context` propagation into `Container.HealthCheck` callers from request scope (already done via `r.Context()` — verify all call sites) | Low    | S      | Quality       |
| 50 | Retire/merge superseded report sections once HARVEST lands (annotate done items in 01:55/01:56 (f) lists)                                               | Low    | S      | Documentation |

## g) Questions I cannot figure out myself

1. **How stale is acceptable?** When the repository is down and we serve cached HTML: bounded staleness (e.g. last N minutes) or indefinite-until-refresh? And for paths never cached: 503, 404, or empty shell? The product answer shapes fix #2's entire design.
2. **Under stale-serving mode, should `/health` stay 503 (orchestrators keep restarting) or flip to 200-`degraded` (instance stays up, in-process healing owns recovery)?** Your "not deployed yet" + "stale cache" answers make both defensible; the probe semantics under the new mode is a distinct call I can't derive.
3. **Parallel-session protocol:** should independently-running sessions (like today's 01:07/01:55/01:56 overlap) coordinate via a claim/lock file or a git convention, or do you accept the overlap cost and want each session to simply re-verify the other's green claims by default?

---

_Point-in-time snapshot. Supersedes the (g) answers of `01-55`; annotates `01-56`. Section (f) is HARVEST input — Tier 1 to `TODO_LIST.md`, Tier 2 to `ROADMAP.md`._
