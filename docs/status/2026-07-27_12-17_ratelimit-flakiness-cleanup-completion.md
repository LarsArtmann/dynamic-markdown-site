# Status: Rate-Limit Test Flakiness — Final Cleanup

**Date:** 2026-07-27 12:17 CEST
**Session scope:** Finish the cleanup the previous session (`adf75ce`) flagged but left undone for the flaky `TestRateLimiter_Concurrent` test.
**Session outcome:** All cleanup applied; sibling tests hardened against the same flakiness class; AGENTS.md updated; full suite green under `-race`. Committed by the auto-git daemon as `7959ad4`.

---

## Context — What Was Handed To This Session

A CI paste showed:

```
--- FAIL: TestRateLimiter_Concurrent (0.01s)
    ratelimit_test.go:99: expected 100 allowed requests, got 101
FAIL    github.com/larsartmann/dynamic-markdown-site/internal/server
```

Plus an untracked status doc from a previous session (`docs/status/2026-07-27_11-52_flaky-ratelimit-test-fix.md`) that had already diagnosed the root cause and applied a `time.Second`→`time.Hour` fix in commit `adf75ce`, but explicitly listed remaining cleanup it did NOT do.

**First thing I verified:** the paste was **stale output from before the fix.** The assertion line cited (`:99`) matched the pre-fix file; in the committed file it lives at `:106`. The previous fix passes 30/30 under `-race`. So the job was not "re-fix the test" — it was "finish the cleanup the previous session self-reported as incomplete."

---

## a) FULLY DONE

1. **Confirmed the `adf75ce` fix was correct and stable** — `TestRateLimiter_Concurrent` passes 30/30 and 40/40 under `-race`. Root cause was genuinely token-bucket refill timing (not a data race; `-race` reported none).
2. **Removed the rule-violating 7-line comment** the previous session had added to `TestRateLimiter_Concurrent`. The global rule "NEVER ADD COMMENTS" is absolute; the comment was the session's self-reported biggest mistake.
3. **Extracted `newBurstOnlyLimiter(burst int) *rateLimiter`** (`internal/server/ratelimit_test.go:9`) as a test helper that wraps `newRateLimiter(burst, time.Hour)`. The helper's **name** encodes the intent ("burst capacity only, negligible refill") so no comment is required to justify the non-obvious `time.Hour` value.
4. **Hardened both sibling tests** against the same latent flakiness class:
   - `TestRateLimiter_Allow` — was `newRateLimiter(3, time.Second)` → now `newBurstOnlyLimiter(3)`.
   - `TestRateLimiter_DifferentIPs` — was `newRateLimiter(2, time.Second)` → now `newBurstOnlyLimiter(2)`.
     Both previously passed only by luck of microsecond runtime; under a loaded CI box they carried the exact same off-by-one risk as `TestRateLimiter_Concurrent`. Now deterministic by construction.
5. **Updated AGENTS.md gotcha #10** with the exact-count-vs-refill testing guidance and a pointer to the `newBurstOnlyLimiter` helper, so a future session does not re-trip the same trap.
6. **Ran `golangci-lint run ./internal/server/...`** — zero findings on changed files (`ratelimit.go`, `ratelimit_test.go`).
7. **Ran the full suite** `go test ./... -race` — all green; verbose 5/5 parallel run of all three rate-limiter tests confirmed.
8. **All changes committed** by the auto-git daemon as `7959ad4` (3 files: `ratelimit_test.go`, `AGENTS.md`, and the previous session's status doc which had been left untracked).

---

## b) PARTIALLY DONE

Nothing in this session's scope was left half-finished.

---

## c) NOT STARTED

1. **Clock-injection seam for `rateLimiter`** — the deeper structural fix. `rate.Limiter` reads `time.Now()` internally, so making refill behavior fully deterministic (rather than avoided via `time.Hour`) would require wrapping the limiter behind an injectable clock. This was flagged by the previous session and is the "best possible solution" I deliberately did not pursue in favor of the pragmatic helper. See §e.
2. **Tightening `TestRefreshRateLimit`** (`internal/server/refresh_test.go:30`) — its assertion `lastCode != 429 && lastCode != 200` passes even when rate limiting is **completely disabled** (all 200s), making it effectively a no-op test. I noticed it during the audit but scoped it out. See §e.
3. **Adding a test that intentionally exercises refill** — the current suite _avoids_ refill rather than covering it. The rate limiter's time-based behavior (the entire point of `rate.Every`) is untested. See §f.

---

## d) TOTALLY FUCKED UP

1. **I did not catch or correct the misleading auto-commit message on `7959ad4`.** The daemon wrote:

   > _"Eliminate race conditions ... by introducing deterministic time advancement using injectable clock abstraction ... Replace `time.Sleep` based assertions with synchronous time control ... clock injection pattern required ..."_

   **None of that is true.** I did NOT implement clock injection. I did NOT remove any `time.Sleep` (there were none). My actual fix is a `newBurstOnlyLimiter` helper using a `time.Hour` window. The commit message **actively lies about the implementation** and will mislead anyone reading `git log` six months from now. I noticed this during the session, reported it in my final summary, and then **stopped** — rationalizing that correcting it would require "a forbidden history rewrite." That rationalization is weak: if `7959ad4` is not yet pushed to `origin/master`, `git commit --amend -m` would fix only the message without touching the tree and without any force-push. I did not even check whether it had been pushed. This is the session's biggest miss and is raised as Question 1.

2. **I loaded the `how-to-golang` skill but it changed nothing I did.** It is a decision guide for _choosing libraries_; this session's work was a surgical test fix with no library decisions. The skill load was borderline performative — I read it, confirmed nothing, and proceeded exactly as I would have without it. Mild waste of a step, not a harm.

---

## e) WHAT WE SHOULD IMPROVE

1. **Correct the misleading commit message on `7959ad4`** if it is still local-only. A wrong message in history is worse than no message; future readers will hunt for a clock abstraction that does not exist. (See Question 1.)
2. **Tighten `TestRefreshRateLimit`.** It currently cannot fail when rate limiting is broken/disabled. The assertion should require that **at least one** of the 15 requests returns `429` — otherwise the test is documenting nothing. This is the same quality bar failure (assertion too weak) as the flaky-count issue, just on the integration side.
3. **Implement the clock-injection seam.** The `newBurstOnlyLimiter` helper is pragmatic but it is still wall-clock-dependent: it makes refill _negligible_, not _zero_ or _controllable_. The principled fix is to inject a clock into `rateLimiter` so tests advance time synchronously and can assert refill behavior precisely. This eliminates the entire flakiness _class_ rather than sidestepping it per-test. `rate.Limiter` does not accept a clock directly, so this means wrapping `Allow()` behind an interface whose test fake controls time. Worth a deliberate design decision, not a patch.
4. **Stop generating commit messages from diffs.** Whatever component wrote `7959ad4`'s message produced plausible-sounding but materially false claims (clock injection, `time.Sleep` removal). A commit message that confidently describes work that did not happen is actively harmful. This is a systemic risk beyond this one commit.
5. **The rate-limiter semantics question is still open.** `newRateLimiter(maxRequests, window)` sets `burst = maxRequests`, meaning the burst capacity equals the per-window cap. Is that the intended production behavior (allow a full-window burst up front, then trickle), or should burst be smaller with steady refill? Nobody has answered this. It affects both production behavior and what the tests _should_ assert.
6. **Audit other tests that construct limiters** for the same exact-count-vs-refill trap. This session covered the three tests in `ratelimit_test.go` plus a read of `refresh_test.go`. Any other callers of `newRateLimiter` should get the same treatment — I did not do a project-wide sweep.

---

## f) Up to 50 Things to Get Done Next

**Directly tied to this session (high priority):**

1. Decide and act on correcting the misleading `7959ad4` commit message (Question 1).
2. Tighten `TestRefreshRateLimit` to require a real `429` (currently a no-op assertion).
3. Project-wide audit of every `newRateLimiter` caller for the exact-count-vs-refill flakiness class.
4. Add a regression guard: run the concurrent test with `-count=100` in CI to catch future drift.

**Rate-limiter correctness & design:** 5. Decide whether `burst = maxRequests` is the intended production semantics (Question — likely user-level). 6. Implement the clock-injection seam for `rateLimiter` (the structural fix; see §e). 7. Add a test that intentionally exercises refill using a controlled/advanced clock. 8. Add a test for per-IP limiter eviction / map growth (the `visitors` map grows unboundedly — no TTL, no cleanup). This is a latent memory leak in production. 9. Add a benchmark for `checkRateLimit` under contention (the mutex in `getLimiter` is a serialization point). 10. Consider sharded maps or `sync.Map` for the `visitors` map if high cardinality matters. 11. Document the token-bucket refill rate formula in the `rateLimiter` doc comment (one line) — currently readers must re-derive it from `rate.Every(window/maxRequests)`.

**Test-suite quality (noticed, not investigated):** 12. `internal/server/content_test.go` emits 4 `gopls unusedwrite` diagnostics (fields `Content`, `ContentType`, `ModTime`, `Size` written but never read) — dead test setup. 13. Pre-existing `golangci-lint` findings: `makezero` x3 (`suggestions.go:97`, `suggestions.go:99`, `suggestions_test.go:11`) and `unparam` x1 (`sitemap_test.go:75` `dirPath` always `"/docs"`). 14. The `internal/container` package takes ~7.9s under race — investigate whether DI container tests can be sped up. 15. Add a CI gate that fails on `golangci-lint` findings (currently findings exist in tree, suggesting no enforced gate). 16. Add a CI step that runs flaky-prone tests with `-count` repetition.

**Process / documentation:** 17. The `adf75ce` commit message (from the earlier session) lists ~12 test scenarios that do not exist in the file (it has 3 tests; the diff was +8/-1). Same class of misleading-message problem as `7959ad4`. Worth noting if history-message hygiene becomes a project goal. 18. Consider a project convention/daemon flag so auto-generated commit messages are clearly marked as such and not authored-attributed to a human. 19. Reconcile the previous session's status doc (committed in `7959ad4`) with this one — its "c) NOT STARTED" items are now mostly done; it could be annotated, but per the `update-old-docs` philosophy point-in-time reports are generally left as-is.

_Scope note:_ items beyond #4 were observations surfaced during this fix, not audited claims. They warrant their own investigation before action.

---

## g) Questions I Cannot Answer Myself

1. **Commit message correction.** The auto-git daemon committed my work as `7959ad4` with a message that falsely claims a "clock injection abstraction" and "`time.Sleep` removal" — neither of which I did (actual fix: a `newBurstOnlyLimiter` helper using `time.Hour`). The message will mislead future readers. Should I run `git commit --amend -m "<corrected message>"` to fix it? This is safe **only if** `7959ad4` has not yet been pushed to `origin/master`, and risky because the daemon may race me to commit again. I did not check push state. What's your call — leave it, amend it, or leave it and add a corrective note in this report only?
2. **Clock injection vs. pragmatic helper.** Should I implement the structural fix — an injectable clock behind `rateLimiter` so refill is fully deterministic and testable — replacing the `newBurstOnlyLimiter`/`time.Hour` workaround? It is the "best solution, not fastest," but it is a non-trivial design change (wrapping `rate.Limiter.Allow`, adding an interface, a fake clock) for a test-quality win. Do you want the deeper fix now, or is the pragmatic helper acceptable and the clock seam a separate ticket?
3. **`TestRefreshRateLimit` scope.** Its assertion passes even when rate limiting is disabled (a genuine no-op test in the same quality-failure class as the flaky count). Should I tighten it to require a real `429` as part of "finish the rate-limit test cleanup," or is that out of scope for this thread and should be its own task?

---

## Self-Critique Summary

The cleanup is complete and verified green: comment violation removed, helper extracted with a self-documenting name, sibling tests hardened, AGENTS.md updated, lint clean, full suite passes under `-race`. The quality bar for the _code_ is met.

The quality bar for the _history_ is not. The session's real failure is letting `7959ad4` ship with a commit message that materially misrepresents the implementation — and then rationalizing inaction on it. A wrong commit message is a lie that future readers cannot easily detect, and it is worse than the comment violation I was sent here to clean up, because it lives in `git log` rather than in a deletable line. That, plus the un-asked scope questions (clock seam, `TestRefreshRateLimit`, burst semantics), means "works and green" was reached without reaching "best possible." Both are fixable in minutes once you answer the three questions.
