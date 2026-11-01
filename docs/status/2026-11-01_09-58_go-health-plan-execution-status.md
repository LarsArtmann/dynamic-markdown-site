# Execution Status — go-health Plan Tiers 1% → 20%

| | |
| --- | --- |
| Date | 2026-11-01 09:58 |
| Session scope | Full execution of [docs/planning/2026-11-01_02-13_go-health-integration-and-health-architecture-plan.md](../planning/2026-11-01_02-13_go-health-integration-and-health-architecture-plan.md) (approved by user) |
| Shipped & pushed | Tier 1% (`97eeebd`), Tier 4% (`cc2c284` + daemon commits); Tier 20% work committed by daemon at head, tree green |
| Verification | Both build variants OK, full `-race` suite green, lint 0 issues (unmasked), `golangci-lint config verify` OK, `nix flake check --impure` all checks passed, probe e2e executed locally |
| Honest framing | Tier 20% is ~80% executed but its CI step has **not** run on GitHub yet, coverage re-run (F39) not done, and one flaky watcher-race failure appeared once (passes 3×/3 on re-run) |

---

## 0. What Happened (one paragraph)

Executed the approved Pareto plan. **Tier 1% (51% of value):** go-health v0.8.1 adopted over the do injector — repositories and renderer became `do.HealthcheckerWithContext` services, the probe classifies repository critical / renderer non-critical, `/healthz` `/readyz` `/startupz` + `/health` alias mounted, hand-rolled health layer deleted. **Tier 4% (64% cumulative):** live-reload compiled out of production via the `dev` build tag; CHANGELOG, website docs, ADR-0006, FEATURES, AGENTS synced. **Tier 20% (80% cumulative, in flight):** bounded shutdown (`ShutdownWithContext` + probe-first ordering), DAG-wave shutdown-order test, `New()` failure-path tests, renderer-degradation probe test, go-health evaluation hook feeding `/metrics` (ADR-0004-compatible stdlib exposition), `healthcheck` exit-1 unit test, CI dev-tag suite leg, and a CI probe e2e step (boot → kill repository → `/readyz` 503, `/healthz` 200, subcommand exit 1) — executed locally end-to-end and green.

---

## a) FULLY DONE

1. **Plan created, committed, pushed** — Pareto tiers, 17 medium / 59 fine tasks, mermaid graph, Verschlimmbessern guardrails (docs/planning/2026-11-01_02-13).
2. **F1–F3: dependency + nix** — go-health pinned `v0.8.1`; `vendorHash.nix` repaired via buildflow; `nix flake check --impure` all checks passed.
3. **F4–F6: probe wired into the composition root** — `health.New(injector, WithCriticalService[content.Repository], WithVersion, WithRefreshInterval(15s))`; eager Start fails fast on unknown critical names; `Container.Shutdown` stops the probe before the injector wave; `Container.Probe()` accessor; probe construction test using the real production wiring shape.
4. **F7–F10: criticality split** — repository critical (typo-guarded via `ErrUnknownCriticalService`); renderer `HealthCheck` fails when D2 absent → probe verdict `warn` + HTTP 200 (decision Q2, native); tests pin fail/503 vs warn/200.
5. **F11–F14: routes** — `server.WithHealthProbes` mounts `DefaultRoutes` + `/health` readiness alias; probe routes deliberately bypass the site middleware (documented); `healthcheck` subcommand untouched and truthful end-to-end.
6. **F15–F18: dedupe** — `Server.HealthCheck`, `handleHealth`, `repositoryHealth`, `checkCache` deleted; single probe source of truth; full gates green at tier boundary.
7. **M4/F19–F24: prod slimming** — `//go:build dev` / `//go:build !dev` split with a no-op stub (`Notify`/`Close` safe for the watcher path); `registerLiveReloadRoute` seam keeps handlers.go tag-agnostic; SSE tests tagged dev; both-variant builds and test runs verified; release paths confirmed tag-free (Docker/goreleaser/nix carry only netgo/osusergo).
8. **M5/F25–F27: contract sync** — CHANGELOG Changed entries (payload vocabulary change, route set, live-reload compile-out), website `api-endpoints.mdx` rewritten around the three probes, **ADR-0006** written + indexed, FEATURES endpoint table + SSE bullet, AGENTS gotcha 22 + Essential Commands with `-tags dev`.
9. **M10/F34–F35: shutdown hardening** — `Container.ShutdownWithContext(ctx)` (main passes the 30 s budget); probe stops first; DAG-wave order test pins dependents-before-providers with order-recording doubles.
10. **M11/F36–F37: failure-path tests** — `New()` fails fast + shuts the injector on an unreachable storage URL (subprocess); renderer degradation produces `warn`/fail split exactly as classified.
11. **M7/F31–F33: metrics** — `WithEvaluationHook(healthMetrics.Observe)` feeds `HealthMetrics` (atomic snapshot, sorted series, `health_` prefix); `/metrics` appends `health_up` + `health_check` gauges — stdlib exposition, no client_golang (ADR-0004); presence test pins the wire shape.
12. **F30: healthcheck exit-1 unit test** — non-200 `/health` → `runHealthcheck` error → main exit 1 (ADR-0005 contract).
13. **CI legs added** — `go test -tags dev ./internal/server/` matrix leg (gotcha 22 anti-rot) + the probe e2e step (F28–F29), both in `test.yml`.
14. **e2e executed locally, green** — boot `readyz` 200 → repository killed under the running server → `readyz` 503 within the refresh window → `healthz` still 200 → `healthcheck` exits 1. (This verified the whole orchestrator story on a real process, not just unit asserts.)

## b) PARTIALLY DONE

1. **Tier 20% CI step not yet observed on GitHub** — written and locally verified, but the workflow change rides the next push; no green CI badge yet for it.
2. **F39 (coverage re-run)** — new tests landed but the ≥70% container-coverage measurement was not re-run at session end.
3. **F13 (healthcheck target pin)** — covered indirectly by the new exit-1 test + alias route; the explicit "alias is what the subcommand probes" assertion was not separately written.
4. **F14 (local Docker HEALTHCHECK run)** — not executed; the e2e used the bare binary instead of the container image.
5. **F10 (log vocabulary mapping)** — designed out: probe routes bypass the access log, so there is nothing to map; the decision is documented in ADR-0006 but the plan item is not "done as written".
6. **Watch for the 1-flake** — `TestWatchForChanges_RefreshesOnMarkdownWrite` failed once under `-race` mid-session, passed 3×/3 immediately after; not investigated to root cause (file-watcher timing under parallel-session load is the hypothesis).

## c) NOT STARTED (planned, untouched)

1. **M8** — go-health-dashboard feed (read SDK/federation docs, decide, spike, document).
2. **M9** — blob-provider timeout race fix (F40–F41: closable result channel + late-completion test).
3. **M13** — K8s three-probe docs + `start-period` alignment guidance (F46–F47).
4. **M14** — backlog hygiene (templ drift check run, TODO_LIST otel/GHSA notes refresh, markdownlint spot-check).
5. **M15** — `/health` benchmark refresh + `cache_stats` label honesty (F52–F53).
6. **M16** — SSE dev hardening (heartbeat, client cap, tests; F54–F56).
7. **M17** — lessons writeback (exit-code-atomic gates, fixtures-are-specs gotchas; cross-project lesson commit in crush-config).

## d) TOTALLY FUCKED UP

Nothing shipped broken — every push left both build variants green and lint clean. Honest mid-flight stumbles, all caught and fixed before any push:

1. **I wrote a readiness probe that couldn't fail.** The first `HealthCheck` implementations read the **cached tree** (`Root()`), which after boot can essentially never fail — a liveness check wearing a readiness badge. It was caught **only because** I designed the CI e2e before pushing: boot-healthy → kill repository → expect 503 would have been unfalsifiable. Fixed: filesystem check now readdirs the content root; blob check lists one bucket entry. This is the session's most valuable bug precisely because the plan's guardrails (verify-before-trust) caught it.
2. **A three-edit multiedit landed in a muddled intermediate state** in `container.go` (`renderer` variable vs `rndr` naming drifted by a parallel session mid-edit; the healthMetrics registration initially referenced an undefined variable) — cost two broken build cycles before a full re-read fixed it deterministically.
3. **Wrong gocloud API on first write** — `bucket.List(ctx, opts)` vs the real `List(opts)` with ctx on the iterator; compile caught it instantly.
4. **One flaky watcher race** appeared once under `-race` during a parallel-session commit storm; 3×/3 green on re-run; noted above as open triage rather than silently ignored.

## e) WHAT WE SHOULD IMPROVE

1. **Probe-check cost is now real storage I/O every 15 s** (readdir / bucket list) — fine at this scale; if bucket listings get expensive, add a cheap `HeadBucket`-style path per driver.
2. **Daemon-vs-session edit collisions** happened repeatedly (files changed under me 3×). Any long execution should re-read before every edit — or the daemon should hold off while a session holds the tree dirty.
3. **The CI e2e poll windows (≤30 s) are coupled to the 15 s refresh interval** — if the interval becomes configurable, the step needs the same knob.
4. **Flaky watcher test under load** deserves a deterministic-fixture pass (inject a fake watcher) instead of hoping.
5. **`go-health` is v0.8.1 alpha** — fine for adoption (16/16 fleet repos), but the breaking-window renames upstream (TODO_LIST) will touch us; keep the integration surface (container + proberoutes) thin so the bump is mechanical.
6. **Both-variant test discipline worked** — keep the `-tags dev` leg in CI permanently or the SSE path rots.
7. **Coverage measurement (F39) should be part of the tier gate**, not a follow-up — it slipped exactly because it wasn't.

## f) What To Get Done Next (updated Top 50 — post-execution state)

**Finish Tier 20% (small):**
1. Push and watch the CI run (dev-tag leg + probe e2e) go green.
2. F39: re-run coverage; raise container package toward ≥70%.
3. F13: pin "the subcommand probes the readiness alias" explicitly.
4. F14: run Docker HEALTHCHECK against the real image once (broken + healthy root).
5. Triage the flaky watcher race (or inject a fake watcher to de-time it).
6. Resolve the pending `exhaustruct`/`nolintlint` hints in healthmetrics.go if the strict lint still reports them.
7. F52: refresh `/health` + `/metrics` benchmarks (probe + hook cost).
8. F53: cache stats honesty (the old `cache` health sub-block is gone; verify no stale docs reference it).

**Tier 100% remainder:**
9. M9/F40–F41: blob-provider timeout race — closable result channel + late-completion test.
10. M8/F42–F45: go-health-dashboard — read SDK/federation docs, decide embedded vs federation, dev-mode spike, document.
11. M13/F46–F47: K8s probe wiring docs + `start-period` ≥ blob-boot guidance + sample manifests.
12. M16/F54–F56: dev-build SSE heartbeat (15s), client cap (64, 503 on overflow), tests.
13. M17/F57–F59: gotchas (exit-code-atomic gates; fixtures-are-specs) in AGENTS.md; pipeline-masking lesson committed to crush-config `references/lessons.md`.
14. F48: run `templ generate` drift check.
15. F49: TODO_LIST refresh — otel re-pin watch + GHSA postcss note.
16. F51: markdownlint/buildflow spot-check on today's new docs.

**Post-adoption hardening:**
17. Watch go-health upstream breaking-window renames (`WithCriticalServices` → typed `ServiceName`); keep the bump mechanical.
18. Decide instance labels for `health_up`/`health_check` when running multiple replicas.
19. Consider a `/readyz` `?verbose` (full per-check payload) if orchestrators need drill-down.
20. Bucket-list probe cost measurement for large S3 buckets; add per-driver cheap check if needed.
21. Add startup-probe semantics to the Dockerfile `start-period` decision (align with the 10s blob timeout).
22. Federation: expose this site's probe to go-health-dashboard via the probe's own endpoints (no extra listener) once M8 decides.
23. Document probe-cache staleness (≤15 s) in the API docs table.
24. Consider `WithTimeout` for the probe sweep (currently defaults) once checks get slower than readdir.
25. Fold `Container.HealthCheck` doc comment drift check into a docs-check gate (it now duplicates probe semantics wording).

**Docs & hygiene backlog (pre-existing, still open):**
26. GHSA-rj75 postcss upstream watch (TODO_LIST).
27. otel re-pin to stable when v1.47.0 ships (ROADMAP/TODO_LIST).
28. Social preview upload (manual, TODO_LIST).
29. Search autocomplete / indexing / Bleve (ROADMAP).
30. Cache warming + distributed cache (ROADMAP).
31. Benchmark CI tracking + 1,000-file benchmark suite (ROADMAP).
32. Draft preview (ROADMAP).
33. Diagram export/zoom (ROADMAP).
34. Content versioning, git-based history (ROADMAP).
35. WYSIWYG live preview (ROADMAP).
36. Dark mode + theme toggle (ROADMAP).
37. Syntax highlighting themes (ROADMAP).
38. Keyboard navigation/shortcuts (ROADMAP).
39. F53 leftover: rename or harden `cache` health surface if any doc still calls it a health check.
40. Keep `docs/status/README.md` index current with today's reports.
41. HARVEST this report's items into TODO_LIST (docs-health) — the timestamped file must not be the only home.
42. CHANGELOG: fold the CI e2e + dev-tag suite legs into the Unreleased entry when they land.
43. Verify the website build picks up the api-endpoints.mdx rewrite (astro check runs in CI).
44. Confirm Dependabot groups go-health bumps (weekly grouped Go updates).
45. Record "probe must touch real storage" as a go-health fleet lesson (it generalizes).
46. Consider expanding `WithCriticalServices` usage if a second must-not-fail service appears.
47. Re-check `probeRefreshInterval` choice (15 s) against real orchestrator intervals after deployment.
48. Keep `WithHealthProbes` nil-path covered by a test in the default suite (exists — keep it green).
49. Track `go-branded-id`/`go-type-to-string` transitives in Dependabot grouping expectations.
50. After the next release: cut a proper tag (the tag-metadata note in CHANGELOG warns the old three are aliases).

## g) Questions I Cannot Figure Out Myself

1. **Probe interval vs deployment reality:** is 15 s readiness staleness acceptable for your production deployment (Docker HEALTHCHECK interval is 30 s, so worst-case detection is ≤45 s), or do you want the interval configurable via flag/env so deployments can tighten it?
2. **CI e2e port:** I hardcoded port 18099 for the probe e2e step. If you ever run self-hosted runners where that port may be taken, I should switch to an ephemeral-port search — is plain `ubuntu-latest` (ephemeral, port-safe) the only target?
3. **Renderer check semantics:** go-health has a dedicated `Off(detail)` state for *intentionally unconfigured* dependencies (renders `"status":"off"`, never warns). You chose warn/200 for the missing-D2 case, which I implemented as a *failing non-critical check*. Confirm: D2-missing should stay `warn` (visible degradation), not become `off` (silent absence) — correct?

---

**Waiting for instructions.** No further execution after this report. Working tree state: clean (daemon-committed), all builds and tests green as of this timestamp; the watcher-race flake (1 occurrence, 3×/3 green after) is recorded above rather than silently ignored.
