# Status Reports Index

Point-in-time session reports. **Resolved reports are archived** and carry inline `~~strikethrough~~` verdicts (`done` / `not-do` / `docs-health pass` markers). Unmarked rows in living reports below are **open work** — verify against the code before acting.

- Living reports: this directory (`docs/status/`)
- Archived reports: [`archived/`](./archived/) — fully resolved, immutable historical record
- Planning documents: [`../planning/`](../planning/)

## Living reports (open items remain)

| Report | Topic |
| --- | --- |
| [2026-09-27 docs-health AUDIT pass + self-critique](./2026-09-27_04-30_docs-health-audit-pass-and-self-critique.md) | Full docs audit; §b harvest gaps, §f follow-ups, §g decisions |
| [2026-09-13 pipeline fix + findings gate](./2026-09-13_14-52_pipeline-fix-and-findings-gate.md) | BuildFlow gate diagnosis; remaining advisories |
| [2026-07-27 ratelimit cleanup follow-up + self-review](./2026-07-27_16-41_ratelimit-cleanup-followup-self-review.md) | Burst-only test semantics |
| [2026-07-27 ratelimit flakiness cleanup completion](./2026-07-27_12-17_ratelimit-flakiness-cleanup-completion.md) | Burst-only helper + determinism |
| [2026-07-27 flaky ratelimit test fix](./2026-07-27_11-52_flaky-ratelimit-test-fix.md) | Off-by-one flake root cause |
| [2026-07-26 go-filewatcher v2 adoption](./2026-07-26_18-59_go-filewatcher-v2-adoption.md) | Watcher migration; watcher-test follow-up |
| [2026-07-13 firebase DNS + hosting setup](./2026-07-13_22-22_firebase-dns-hosting-setup.md) | Website deploy pipeline |
| [2026-07-13 docs-health audit + self-critique](./2026-07-13_22-37_docs-health-audit-and-self-critique.md) | Earlier audit pass |
| [2026-07-13 buildflow jsonv2 fix + self-critique](./2026-07-13_21-35_buildflow-jsonv2-fix-and-self-critique.md) | GOEXPERIMENT=jsonv2 adoption |
| [2026-07-13 readme/website/github overhaul](./2026-07-13_21-18_readme-website-github-overhaul.md) | Public presence rebuild |
| [2026-06-18 buildflow green / vendorhash / testfix](./2026-06-18_09-54_buildflow-green-vendorhash-testfix.md) | Pipeline repair session |
| [2026-06-18 goreleaser deprecations status](./2026-06-18_09-04_goreleaser-deprecations-status.md) | Release config migration |
| [2026-06-13 comprehensive status + library audit](./2026-06-13_10-40_comprehensive-status-dependency-update-and-library-audit.md) | Dependency sweep |

Most executed items land in [TODO_LIST.md](../../TODO_LIST.md) (open) or [CHANGELOG.md](../../CHANGELOG.md) (done) — those two files stay canonical; the reports carry the reasoning.
