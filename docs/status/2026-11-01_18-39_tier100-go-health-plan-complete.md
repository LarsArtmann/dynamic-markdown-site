# Tier-100% Execution: Go-Health Plan Completed (2026-11-01 18:39)

**Verdict:** the 2026-11-01 02:13 plan is fully executed. Every medium task M8–M17
is done; the Tier 1% partials (F13/F14) are closed to the extent a local
environment allows. All gates green at write time.

## Completed this session

| Item | Outcome |
| ---- | ------- |
| F13 | `TestHealthcheckSubcommandTracksProbeReadiness` pins the subcommand against the *real* `WithHealthProbes` wiring: exit 0 on probe pass, `errHealthcheckFailed` on critical fail. Prior tests used hand-rolled muxes and proved nothing about the alias. |
| F14 (substance) | Dockerfile-identical HEALTHCHECK command run against a live binary: exit 0 with healthy root, exit 1 after `chmod 000` root (CI's probe-e2e asserts the red direction; green now asserted too). **The Docker-image run itself is blocked locally: no container runtime exists** (no docker/podman/nerdctl/socket). CI builds the image from git, so directive validity is CI-checked. |
| F39 | Coverage truth repaired: subprocess tests now forward the parent's `-test.gocoverdir` flag (test binaries ignore the `GOCOVERDIR` env var — measured 44.3% while true coverage was 87.2%). `internal/container` now measures **87.2%**; `Probe()` accessor contract pinned. Gap list for remaining <70% files: `cmd/main.go` process wiring (covered by CI probe-e2e at binary level), `runHealthcheck` (4-line glue over the tested `runHealthcheckWithArgs`), `watchForChanges` 60% (dev watcher). |
| M9 (F40–F41) | Blob-repository boot deadline race fixed: the abandoned goroutine's result is drained and its repository shut down (`buildBlobRepository` + `drainAbandonedBlobRepository`); construction is injectable for tests. Four race-detected tests cover on-time, late-success (Shutdown asserted via channel), late-error, and no-Shutdown-interface paths. |
| M13 (F46–F47) | `website/src/content/docs/guides/kubernetes.mdx` (sidebar-registered, site builds): three-probe table with **verified** go-health semantics (`/startupz` evaluates critical checks per request then latches green — read from go-health source, not guessed), sample Deployment manifest, startup-budget sizing table, Docker `--start-period` ≥ 10s blob guidance. |
| M8 (F42–F45) | go-health-dashboard research → decision (ADR-0007): **embedded SDK, dev builds only**, at `/debug/health-dashboard`; federation (`health-hub` polling `/health` documents) recorded in ROADMAP as the production answer. Pin **v0.10.2**: compiles against go-health v0.8.1 and its templ v0.3.1020 matches ours (v0.12.x would force templ v0.3.1070 + regeneration). Live spike verified: dev binary serves dashboard HTML (200, datastar), `/health` alias still JSON `pass`, prod binary 404s and links zero dashboard code (`go version -m` compared). Collision guard test pins the alias against the dashboard's default `/health` HTML route. |
| M14 (F48–F51) | templ drift: 0 files. TODO_LIST refreshed against reality: **otel already at stable v1.47.0** (re-pin item closed), GHSA item superseded — see next row. Social-preview note re-verified (og/home.png present). markdown-lint exit 0. |
| (bonus) | Website audit found 10 advisories (5 high, 5 moderate) — all cleared via `pnpm-workspace.yaml` overrides (`undici >=8.12.2`, `zod >=4.6.6`, `postcss-selector-parser >=6.1.5`; pnpm v11+ reads overrides from the workspace YAML, not `package.json.pnpm` — same ignore-class trap as `allowBuilds`). `pnpm audit --prod` = **0 vulnerabilities**; `astro build` green. |
| M15 (F52–F53) | `/health` benchmark annotated (cached-readiness path; 2947 ns/op measured). Metrics honesty fixed at root: `health_check` gauges expose **only** services with real checks (repository, renderer) — go-health's sweep synthesizes `pass` for every non-checker service (config/logger/server/searcher/cache), and the container now declares the checked set from its compile-time lifecycle guards. The F53 "cache sub-block" rename became unnecessary: the fake row is gone entirely. |
| M16 (F54–F56) | Dev SSE: 15s heartbeat comment frames + 64-client cap with actionable 503 (check+register share one critical section — no TOCTOU). Tests race-clean via a mutex-guarded recorder (plain `httptest.ResponseRecorder` races when the test polls while the handler streams). |
| M17 (F57–F59) | AGENTS gotchas 25 (exit-code-atomic gates — the trap fired in-session) and 26 (fixtures-are-specs). Pipeline-masking lesson committed to crush-config `references/lessons.md` (separate repo/commit `b4c518a`). |

## Gates at close

- `go build ./...` + `-tags dev` build: OK
- `go test ./... -race` + `go test -tags dev ./internal/server/ -race`: OK
- `golangci-lint run ./...`: **0 issues** (exit recorded unmasked); `config verify`: OK
- `nix flake check --impure`: vendorHash repaired via `buildflow -s nix-hash-fix --fix`; flake check result recorded in the session log
- Website: `astro build` green (13 pages); `pnpm audit --prod`: 0 findings

## Honest caveats

1. **Docker-image verification (F14) and the "dev compose" spike surface (M8/F44) ran as bare-binary e2e** — no container runtime exists on this machine. The dashboard-in-compose rendering claim is therefore NOT verified; what is verified is the same wiring served by a locally running dev binary.
2. The dashboard pusher runs on a background context for the dev process lifetime (documented in ADR-0007); if the dashboard ever graduates beyond dev, lifecycle moves into `Server.Shutdown()` first.
3. CI must be watched once for the dev-tag + probe-e2e legs landing green (written in the previous session; not yet observed on GitHub).

## Next candidates

- Watch next CI run; CHANGELOG carries the CI-leg addition note if a follow-up lands.
- `health-hub` federation smoke test whenever a second service exists to point at.
- templ bump → re-evaluate go-health-dashboard v0.12.x (empty-route disabling).
