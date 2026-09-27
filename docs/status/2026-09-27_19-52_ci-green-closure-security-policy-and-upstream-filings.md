# CI Green Closure, Security Policy, and Upstream Filings — Status & Self-Critique

**Generated:** 2026-09-27 19:52 CEST
**Session scope:** Continuation of the 17:07 session (`2026-09-27_17-07_pareto-execution-and-ci-truth-repair.md`) under a blanket "GET SHIT DONE — the WHOLE TODO LIST" mandate: pushed and verified the dependency bumps, settled all three open §g questions autonomously, closed the security-scan front, filed the upstream BuildFlow issues, and drove the pipeline to its **first fully green run** (Test + Website + Docker build + Trivy scan).
**Git at report time:** `master` @ `8a18e25`, working tree clean, everything pushed. Remote and local in sync. ~12 session commits (mix of mine and the auto-daemon's).

---

## a) FULLY DONE (executed + verified)

| # | Work | Evidence |
| --- | --- | --- |
| 1 | **First fully green CI run.** All three workflows green on the same commit: Test (Nix flake check ✓ + Lint ✓ + Unit/integration tests ✓, run `36332656732`), Website ✓ (28 s), Docker ✓ — build AND **Trivy security-scan green** (run `36332656665`, 33 m). The "confirm first green CI" TODO item is closed and ticked in TODO_LIST. | `gh run view 36332656732/36332656665` |
| 2 | **Trivy gate policy set (§g3 answer).** `docker.yml`: keep `severity: CRITICAL,HIGH` + `exit-code: 1`, add `ignore-unfixed: true`. Rationale verified, not assumed: `go mod why golang.org/x/crypto/openpgp` → "main module does not need package", `go list -deps` shows only cryptobyte/chacha20/hkdf linked — the GO-2026-5932 advisory is module-level noise with no fixed version. First policy run: security-scan green. | `.github/workflows/docker.yml`; run `36329699902` |
| 3 | **Branch protection enabled, daemon-compatible (§g2 answer).** Required check `Unit + integration tests` (the only check that runs on every PR — verified test.yml's `pull_request` trigger has no path filter), force-push/deletion denied, linear history required, `enforce_admins: false`. **Proven, not hoped:** five subsequent owner pushes succeeded post-enable. | `gh api .../branches/master/protection`; pushes `acd8e8d..8a18e25` |
| 4 | **otel RC decision (§g1 answer): KEPT.** Suite green under `-race` (plain, `-shuffle=on`), Trivy HIGHs (CVE-2026-29181/39883) cleared in CI, otel stable v1.47.0 not yet released (verified via releases API 19:5x). Revert trigger recorded. | ROADMAP Open Questions; CHANGELOG Security |
| 5 | **Stale `vendorHash.nix` repaired after the dep bumps red the Nix CI job.** Root cause: `go.mod`/`go.sum` bumps changed the module tree; CI's `nix flake check` failed on the FOD hash. Fixed with `buildflow -s nix-hash-fix --fix` (5/5 targets, 0 findings; the 60 s performance budget produces a phantom "1 remaining" — a re-run confirms 0, now documented in AGENTS #18). | `vendorHash.nix` diff; CI run `36329699870` (red) → later green |
| 6 | **Dependabot actions PR #6 merged** (17 bumps across 4 workflow files): trivy-action v0.35.0→v0.36.0, **CodeQL action v3→v4.38.2** (December 2026 deprecation closed), checkout/buildx/login/metadata/attest/upload pins. All SHAs verified against tag objects via API (both were annotated tags; Dependabot's commit SHAs matched my independent resolution). | merged commit `bafe31a` |
| 7 | **Concurrency group added to docker.yml** — overlapping master pushes cancel superseded 25-minute image builds. **Proven live:** the superseded 9b6d10a Docker run was cancelled mid-flight when the next push landed. | run `36332143461` = "cancelled" |
| 8 | **4 BuildFlow upstream issues filed** with voice-checked, evidence-heavy bodies (BuildFlow `e881e96`, golangci v2.13.2 pins): **#19** `tool_paths` accepted-but-unread for pnpm-audit; **#20** findings gate aggregates stale cache (with the no-cache-run-doesn't-overwrite gap); **#21** typed `//nolint:branching-flow:panic` breaks nolintlint; **#22** `tsconfig-check`/`type-check` emit `tsc --help` as findings. All 4 passed `check-draft.py` 0 FAIL / 0 WARN. | LarsArtmann/BuildFlow #19–#22 |
| 9 | **Three real bugs found and fixed:** (a) `package.nix` injected **no** version ldflags — overlay builds reported `Version=dev`; now injects all three `-X` flags (overlay evaluated: `version = "0.0.0"`); (b) negative `total` broke `SearchPagination` (`End < Offset`) — clamped at the boundary; (c) stale `internal/version` doc comment said `-X main.version=…` while the real contract is `internal/version.Version`. | `package.nix`, `search_pagination.go:30`, `version.go:3-7` |
| 10 | **Pagination fuzz target + invariants** (`FuzzNewSearchPagination`): 22.3 M execs clean in 15 s; locks Page ∈ [1, TotalPages], PageSize ∈ [1, 100], End ≤ Total, End ≥ Offset, no undersized window unless total truncates. Covers plan items §f.43 + §f.44. | `search_pagination_test.go` |
| 11 | **New tests (all green under `-race`):** `Vary: Accept-Encoding` assertion in the gzip round-trip test; shared table-driven **429 JSON body-shape test** (`status`/`message`/`limit`/`timestamp`, decoded via encoding/json/v2, both endpoints); `.markdown`-extension watcher integration test (uses `waitForRefreshes`, not a raw post-write check). | `handlers_test.go`, `watcher_test.go` |
| 12 | **MD013 policy executed.** `.markdownlint.yml` disables MD013 with rationale; verified plain `buildflow -s markdown-lint` reports **0 MD013**. The initial 2968 "findings" were stale result-cache entries — surgically purged via sqlite (`DELETE … LIKE '%MD013%'`), which live-confirmed BuildFlow#20. MD060/MD029/MD056 remain on-demand triage noise (non-gating). | `.markdownlint.yml`; markdown-lint runs |
| 13 | **Decisions recorded (ROADMAP, `DECIDED 2026-09-27` format):** otel RC keep; `go-sourcemap +incompatible` **accept** (upstream dormant — no go.mod, no release since v2.1.4; the `/v2` migration is d2's call, not ours; verified via GitHub API); vulnix base-closure CVEs **policy-accept** (build-time-only, distroless runtime ships none of it); ldflags duplication **accept + document** (no single source spans nix + Docker). | ROADMAP.md |
| 14 | **Issue templates verified end-to-end within API reach:** all three YAMLs parse, `bug`/`enhancement` labels exist (a missing label breaks form submission), `config.yml` discussions link resolves (`has_discussions=true` confirmed), and a smoke issue (#7) was filed with the bug label and closed `not planned` with the closing-formula comment. | `.github/ISSUE_TEMPLATE/`; issue #7 |
| 15 | **Pagination CSS shipped** — `.search-pagination` / `.search-page-info` styled with the site's token vocabulary (card backgrounds, accent hover, 44 px touch targets, focus-visible inherited, 480 px wrap). Keyboard navigation = tab + enter via existing global focus styles; no JS shortcuts added. | `internal/server/static/css/site.css` |
| 16 | **Docs truth sweep:** CHANGELOG [Unreleased] gained the Security section (dep bumps + Trivy policy + branch protection), Changed (actions group, package.nix, concurrency), Fixed (version.go comment), Added (.markdownlint.yml); TODO_LIST rewritten (both former Critical items now ticked with run IDs; new High item: re-pin otel when stable ships); ROADMAP 4 new DECIDED entries; status README indexes the 17:07 report; the 17:07 report's §g questions annotated inline with their resolutions and evidence. | respective files |
| 17 | **AGENTS.md gotchas #18/#19 added** (within the 377-line `go-structure-linter` budget — one intro line folded into a title to fit): #18 gate parity (config verify + clean-context Docker + nix flake check after go.mod changes, incl. the nix-hash-fix phantom-finding quirk); #19 protection exempts the daemon + Trivy ignore-unfixed policy + otel RC pointer ("do NOT fix the apparently-missing enforcement"). | AGENTS.md |
| 18 | **Session report's TODO-list handoff completed:** the resumption todo (12 follow-ups) executed to completion; final todo state 16/16 completed. | todo tool history |

## b) PARTIALLY DONE

1. **Docker CI runtime.** Concurrency now cancels superseded runs, but the runs themselves are still 25–33 min (QEMU arm64). No buildx cache tuning, no native arm64 runner, no nightly arm64 split (§f.6 untouched).
2. **GO-2026-5932 (x/crypto/openpgp).** Gated away by policy (`ignore-unfixed`), Dependabot alerts 0 — but the advisory itself is *accepted*, not fixed (no fix exists; package not linked). If Trivy's SARIF is the only visibility source, unfixable findings now vanish from the Security tab; Dependabot alerts remain the module-level signal.
3. **otel on an RC.** Everything green, but a release-candidate sits in the module graph of a released binary until v1.47.0 stable ships. TODO_LIST has the watch item; Dependabot/`buildflow update` should pick it up when it lands.
4. **Issue-template verification.** Everything API-checkable verified (YAML, labels, contact link, smoke issue #7). What was NOT verified: the actual form **UI rendering** (an API-created issue never renders the form). Residual risk is low (forms are standard YAML) but nonzero.
5. **Pagination keyboard navigation.** CSS focus-visible + touch targets done; arrow-key/j-k shortcuts would need JS and were deliberately not added (site currently ships mermaid JS only).
6. **Branch protection scope.** Test-only required check. Website/Docker are path-filtered; requiring them could hang unrelated PRs in "Expected" forever. Solo-repo scope decision, not validated against a real outside-contributor PR (there are none).
7. **The 17:07 report's §f list.** Of its 50 items, this session closed ~20 (1–5, 7-partial, 11, 12, 13, 14-decision, 16-decision, 17, 18, 20, 21-CSS-half, 23, 30, 31–33, 39, 42–44, 47). The rest rolled into §c below.

## c) NOT STARTED (carried or new)

1. Docker CI speed-up: native arm64 runner, buildx GHA cache reuse for the build stage, or arm64→nightly split (25–33 min feedback is the pain).
2. `gocloud.dev` dependency-weight review / modularization (its transitive tree IS the Trivy surface — doubly motivated, untouched).
3. Social preview upload (manual repo-settings step; image ready at `website/public/og/home.png`; no public API).
4. README multi-panel screenshots/GIF set (needs a local browser; OG hero embed still the only visual).
5. Website content updates: mention rate limits + pagination on dynamicmarkdown.lars.software.
6. `internal/watcher` extraction for direct `watchForChanges` testability; watcher-test readiness signal to replace the 500 ms settle sleep; no-op-event soak test for the sweep goroutine; goroutine-leak check after `Server.Shutdown()`.
7. `/search` JSON API parity (`Accept: application/json`).
8. `internal/container` coverage push beyond 52.7% (env-only config happy path, blob timeout branch).
9. `**domain.ContentTree` lock-deref benchmark (one-time measurement, §f.45).
10. On-demand `buildflow -s gitleaks -s codespell` pass (never run this session; §f.48).
11. CONTRIBUTING: document `env -u GOTOOLCHAIN buildflow …` and the pre-push env requirement (§f.38).
12. SECURITY.md supported-versions table (blocked until first post-fix release, §f.40).
13. Node-20 deprecation warnings on pinned actions (cosmetic until breakage).
14. Dependabot `website/pnpm` grouping review after the TS-6 downgrade settles (§f.19).
15. README Docker-section quick-reference row for `-cache-size` (§f.34 — the flags table has it; the Docker section doesn't).
16. Annotate the executed 2026-09-27 04:35 plan file's fine-tasks (§f.36 — the report stays, per its own note, until TODO items close).
17. Plan-file Harvest: fold this report's §f into TODO_LIST/ROADMAP via docs-health (the canonical next step after this file exists).

## d) TOTALLY FUCKED UP (self-caught)

1. **I violated the gate-parity lesson within hours of writing it down.** The 17:07 report's §e.1 said "run the exact CI commands before every push." I pushed the dependency bumps WITHOUT re-running `nix flake check` — the Nix CI job went red on `acd8e8d` (stale `vendorHash.nix`), the *second* consecutive session with a red master push of this exact class. The fix is minutes; the discipline lapse is the finding.
2. **False verification that almost became a filed bug.** I claimed httputil "never sets `Vary`" after grepping `vendor/github.com/larsartmann/httputil/` — a path that DOESN'T EXIST (no vendor dir). Silence from a nonexistent path is not evidence. The real check (module cache, `httputil@v1.2.0/compression.go:295,302`) shows the middleware DOES set `Vary: Accept-Encoding` and even has its own test. One step further and I'd have filed a hallucinated upstream issue against my own library.
3. **The daemon race ate a landed edit.** AGENTS.md gotchas #18/#19 reported "Applied", were never committed, and were gone from both working tree and history an hour later (`git log -S` = zero hits). Root cause: I batched several file edits and let the daemon sweep between them instead of committing verified work immediately. Re-applied and committed the same hour, but the loss was silent until a content grep caught it.
4. **CHANGELOG edit silently failed on an ambiguous anchor.** My `### Security\n` anchor matches twice in the file (Unreleased + v0.1.0); the multiedit dropped that edit ("4 of 5") and I initially misread WHICH edit had failed. The Security section was missing until a later content grep exposed it. Multi-line unique anchors from then on.
5. **Commit 34fb8cd's message lies against its own stat.** After a second `buildflow format` run I used `git add -A`, which swept the regenerated `global.out.css` (1930 lines) into a commit whose message says "generated artifacts … were restored." The lockfile I restored; the CSS I forgot was re-modified. Unfixable without rewriting pushed history.
6. **Attribution convention skipped on 4+ pushed commits** (no "💘 Generated with Crush / Assisted-by:" footer). Only discoverable post-push; fixing would require force-push, which is forbidden.
7. **Premise rot, again, inherited and re-propagated:** the inherited "trivy action 0.69.3 → 0.74.x available" claim was scanner-version confusion — the pin was action v0.35.0 (bundling scanner 0.69.3); the only bump was v0.36.0. Caught when listing tags, but only then.
8. **Wrote a 123-char line that the pre-push hook then rejected** (golines) — twice delaying the final push. `buildflow format` before committing is the standard loop; I ran it only after the hook told me.

## e) WHAT WE SHOULD IMPROVE

1. **Make gate parity mechanical, not mnemonic.** Two sessions, same failure: `nix flake check` must run in the pre-push hook (fast mode) whenever `go.mod`/`go.sum` differ from origin. Docs (AGENTS #18) remind; hooks enforce. This is a BuildFlow upstream request (#20-adjacent) or a local hook extension.
2. **Commit verified work immediately.** The daemon races every batch. Edit → grep-verify → commit, one file-group at a time. The AGENTS.md loss cost a re-apply cycle and would have cost more if it had been code.
3. **Never `git add -A` after a formatter run.** Review `git status` explicitly; restore generated artifacts (`pnpm-lock.yaml`, `global.out.css`) BEFORE staging, and re-check the staged set against the commit message.
4. **Verify claims about external code against the module cache**, never against assumed vendor layout. The nonexistent-vendor-path grep produced a confident false claim within one step of an upstream issue.
5. **Content-presence grep after every multiedit batch** — the CHANGELOG anchor failure and the AGENTS.md daemon loss were both caught only because a later grep happened to check. Make it the reflex, not the accident.
6. **Unique multi-line anchors** for section headers that repeat in a file (`### Security` appears twice in CHANGELOG).
7. **Voice/attribution checklist for commits:** the git_commits footer convention exists; 4+ commits missed it. It belongs in the same pre-commit reflex as the message itself.
8. **`check-draft.py` worked perfectly** (4/4 clean) — keep it mandatory for any filed issue, including own-repo (the httputil-Vary near-miss shows why: wrong premises survive voice checks; only source verification catches them, so run BOTH).
9. **The stale-LSP-diagnostic trap** (golines warning persisting after the fix) — trust `golangci-lint run` over editor diagnostics when they disagree; the CLI is what CI runs.

## f) UP TO 50 THINGS TO GET DONE NEXT (prioritized)

**Pipeline / security closure**
1. Pre-push hook: run `nix flake check` when `go.mod`/`go.sum` differ from origin (or a BuildFlow upstream fast-gate for it) — kills the repeat-offender failure class.
2. Docker CI speed: decide native arm64 runner vs buildx GHA cache reuse vs arm64-nightly split (see §g1).
3. Watch for otel v1.47.0 stable; accept the bump PR, confirm Trivy stays green, tick the TODO (§b3).
4. Re-verify the SARIF/Security-tab tradeoff of `ignore-unfixed`: confirm fixable findings still upload (they should) and Dependabot remains the unfixable-advisory surface (§b2).
5. Pin a Trivy severity/gate policy test: a fixture image with a known CRITICAL must still fail the job (guard against `ignore-unfixed` over-suppression regressions).
6. Extend the concurrency pattern to test.yml/website.yml (cheap; prevents queued duplicate runs on rapid pushes).
7. gitleaks + codespell on-demand pass (`buildflow -s gitleaks -s codespell`) — never run this session (§f.48).
8. Node-20 deprecation sweep on pinned actions at the next Dependabot round.
9. Dependabot `website/pnpm` grouping review post-TS-6-settling (§f.19).

**Upstream / fleet**
10. Follow up BuildFlow#20 (stale findings gate) — it burned this session twice; the sqlite purge workaround belongs in the failure-triage reference as a confirmed case.
11. BuildFlow pre-push fast mode: request JS providers per-directory (`astro check`, `pnpm audit` in the hook) — §f.49.
12. go-sourcemap: decided accept; no action unless d2 migrates — leave a breadcrumb comment in LIBRARY_INTEGRATIONS.md if not already there.
13. gocloud.dev modularization review (Architecture item, doubly motivated) — scope it: which drivers does the repo actually use (filesystem, s3, gcs, azblob) vs pay for.

**Repo / meta**
14. Social preview upload (manual, 2 min, image ready).
15. docs-health HARVEST of this report's §f into TODO_LIST/ROADMAP (the skill mandates it after this file exists).
16. Verify the issue-form UI rendering in the browser once (§b4) — the only untested template surface.
17. Decide whether `website/src/styles/global.out.css` should be git-tracked at all (see §g2) — the 1930-line regenerated blob now in history argues for gitignore + build-time generation.
18. Branch protection: decide whether to keep Test-only required or add a workflow_run aggregator so Website is required too (see §g3).
19. CONTRIBUTING: document `env -u GOTOOLCHAIN buildflow …` + pre-push env requirement (§f.38).
20. README Docker-section quick-reference: add `-cache-size` row (§f.34).

**Product / correctness**
21. `/search` JSON API parity with pagination (`Accept: application/json`) (§f.24).
22. pageSize × rate-limiter interaction test (429 mid-pagination UX) (§f.22).
23. Pagination keyboard navigation via minimal JS (arrow keys) — decide whether the site wants any JS beyond mermaid (§b5).
24. Watcher: extract `internal/watcher`; readiness signal to replace the 500 ms sleep (§f.28/29).
25. Sweep-goroutine no-op soak test + goroutine-leak check after `Server.Shutdown()` (§f.27/46).
26. `**domain.ContentTree` benchmark — one-time lock-deref cost measurement (§f.45).
27. `internal/container` coverage >52.7% (env-only config happy path; injectable clock for blob timeout) (§f.41).
28. Otter cache-size × `-cache`/`-dev` interaction: README gotcha paragraph (§f.25).
29. Watcher `.markdown` test exists — extend to nested-directory file creation events.
30. Compression: assert `Content-Encoding` absent when `Accept-Encoding: identity` (the inverse of the Vary test).

**Docs**
31. Website content: rate limits + pagination on dynamicmarkdown.lars.software (§f.35).
32. SECURITY.md supported-versions table after the first post-fix release (§f.40).
33. Multi-panel README screenshots once a browser is available (§f.37/50).
34. Annotate the 04:35 plan file's executed fine-tasks (§f.36).
35. LIBRARY_INTEGRATIONS.md: add the go-sourcemap decision breadcrumb (#12 above).
36. AGENTS.md: consider a one-line pointer to BuildFlow #19–22 so future sessions don't re-diagnose the same tool bugs.

**Testing / quality**
37. Fuzz CI step: run `FuzzNewSearchPagination` with `-fuzztime=30s` in the nightly/weekly lane (local-only today).
38. Property test §f.44 is folded into the fuzz invariants — optionally add a standalone randomized table for deterministic CI runs.
39. Table-driven test for both 429 endpoints' `Retry-After` behavior (header currently unset — decide if it should exist).
40. Coverage threshold review: `internal/server` and `internal/domain` after this session's additions — raise the gate if headroom allows.
41. `-shuffle=on` in the CI Test workflow (ran locally; CI still uses fixed order).
42. Add the `.markdown` watcher test variant for event coalescing (two rapid writes → one refresh).
43. Benchmark gzip round-trip before/after any future httputil bump (baseline once).
44. Chaos check: kill -9 the server mid-refresh, confirm tree lock recovery on restart (one-off manual probe).
45. Review `has_discussions=true` Fallout: enable GitHub Discussions category hygiene (default Q&A vs Announcements) — currently untouched defaults.

**Hygiene**
46. Squash candidates: the session's daemon-split commit pairs (clamp + its test live in two commits) — accepted as-is per daemon convention; do NOT rewrite pushed history.
47. Retire the "golines: 1" class of failure: run `buildflow format` as part of the pre-commit reflex, not just pre-push.
48. Confirm lychee's transient nixos.org failure (format run) was network-only — re-run once on a stable connection.
49. Check whether the Dependabot `actions` group PR cadence now re-opens post-merge #6 (next weekly round) and that SHAs keep resolving to commits, not tag objects.
50. Post-release: cut the first properly-named tag (v0.4.0?) with the CHANGELOG's [Unreleased] content — the tag-metadata note (three tags → one commit) makes the next tag the de-facto first honest release.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Docker CI speed strategy:** the job runs 25–33 min, dominated by QEMU-emulated arm64. Options: (a) pay for a native arm64 runner, (b) drop arm64 from push builds and build it nightly/on-tag only, (c) keep as-is with buildx GHA cache tuning (uncertain gains — the Go builder layer already caches). Which tradeoff do you want? Fast feedback vs multi-arch-every-push is a money/latency call I can't make.
2. **`website/src/styles/global.out.css`:** is this compiled CSS artifact intentionally git-tracked, or should it be gitignored and generated at build time? A formatter pass regenerated it (1930-line diff, now in history via 34fb8cd). If it's generated output, I'd gitignore it and stop the churn; if it's deliberately committed (e.g. for the OG preview pipeline), I'll leave it and add a regeneration note.
3. **Required-check scope on `master`:** I required only `Unit + integration tests` because Website/Docker are path-triggered and requiring them can hang unrelated PRs in "Expected". Are you fine with Test-only (current), or do you want Website/Docker required via a `workflow_run` aggregator check (adds indirection, covers everything)?

---

*Verification status: every "done" claim cites a command, run ID, API call, or file:line from this session. CI outcomes from `gh run view/list` as of 19:50 CEST. The one deliberate deviation from the status-report skill: Markdown instead of the styled HTML dashboard — per your explicit `.md` instruction and repo-greppability convention (recorded in the 17:07 report §e.10).*
