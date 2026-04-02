# Comprehensive Reflection & Execution Plan

**Date:** 2026-04-02 17:01  
**Status:** Post-cache-corruption recovery planning

---

## 1. What I Forgot / Could Have Done Better

### Critical Oversights

1. **Did not add cache-clean to justfile first** — The very first thing I should have done when cache issues appeared was add a `just cache-clean` command with proper macOS handling
2. **Did not verify tests immediately after types_test.go fix** — I fixed the test but didn't run it right away to confirm
3. **Did not use smaller test scope** — Running `go test ./internal/domain/...` first would have been safer than full suite
4. **Did not document the renderer interface changes** — The commit `4233fdc` changed `NewRenderedFile` → `NewRenderedFileWithContent` but I didn't verify all callers

### Process Failures

5. **Gave up on cache cleaning too quickly** — Should have tried `sudo rm -rf` or restarting gopls more aggressively
6. **Did not check if justfile had cache-clean already** — Assumed it didn't exist, didn't verify
7. **Did not commit the types_test.go fix immediately** — That was a real fix that should have been its own commit
8. **Over-relied on background jobs** — Should have used foreground commands for critical verification

### Communication Gaps

9. **Status report was too verbose** — Could have been more concise, focused on blockers
10. **Did not explicitly ask about sudo permissions** — Might have been able to fix cache with elevated permissions

---

## 2. Multi-Step Execution Plan (Sorted by Impact/Work Ratio)

### Phase 1: Unblock Everything (Highest Impact, Low Work)

| Step | Task                                   | Impact       | Work   | Command/Action   |
| ---- | -------------------------------------- | ------------ | ------ | ---------------- |
| 1.1  | Add `cache-clean` to justfile          | **CRITICAL** | 2 min  | Edit justfile    |
| 1.2  | Run `just cache-clean`                 | **CRITICAL** | 1 min  | just cache-clean |
| 1.3  | Verify `go build ./...`                | HIGH         | 30 sec | go build ./...   |
| 1.4  | Verify `go test ./internal/domain/...` | HIGH         | 30 sec | go test          |

### Phase 2: Code Quality Verification (High Impact, Medium Work)

| Step | Task                          | Impact | Work   | Verification      |
| ---- | ----------------------------- | ------ | ------ | ----------------- |
| 2.1  | Run `go test ./...`           | HIGH   | 5 min  | All packages pass |
| 2.2  | Run `just lint`               | HIGH   | 3 min  | Zero lint errors  |
| 2.3  | Fix any remaining lint issues | MEDIUM | varies | Commit each fix   |

### Phase 3: Type System Architecture Improvements (High Impact, High Work)

| Step | Task                                         | Impact | Work   | Notes                              |
| ---- | -------------------------------------------- | ------ | ------ | ---------------------------------- |
| 3.1  | Review `domain.Renderer` interface           | HIGH   | 30 min | Check if it belongs in domain      |
| 3.2  | Consider `Result[T]` type for error handling | MEDIUM | 1 hr   | Use github.com/samber/mo or custom |
| 3.3  | Add `Option[T]` type for optional values     | MEDIUM | 1 hr   | Draft field, optional metadata     |
| 3.4  | Review URLPath validation                    | MEDIUM | 30 min | Could use net/url more             |

### Phase 4: Established Libraries Integration (Medium Impact, Medium Work)

| Step | Task                                | Impact | Work | Library                                   |
| ---- | ----------------------------------- | ------ | ---- | ----------------------------------------- |
| 4.1  | Replace custom cache with ristretto | MEDIUM | 2 hr | github.com/dgraph-io/ristretto            |
| 4.2  | Use lo for functional operations    | MEDIUM | 1 hr | github.com/samber/lo (already in go.mod!) |
| 4.3  | Add structured logging with slog    | MEDIUM | 1 hr | Use charm.land/log properly               |
| 4.4  | Use validator for config validation | LOW    | 1 hr | github.com/go-playground/validator        |

### Phase 5: Testing Infrastructure (Medium Impact, Low Work)

| Step | Task                                     | Impact | Work   | Notes                   |
| ---- | ---------------------------------------- | ------ | ------ | ----------------------- |
| 5.1  | Add `just test-watch` command            | MEDIUM | 10 min | File watching for tests |
| 5.2  | Add `just test-short` for quick feedback | MEDIUM | 5 min  | Skip integration tests  |
| 5.3  | Add test fixtures for markdown files     | LOW    | 30 min | Reusable test content   |

### Phase 6: Documentation & Tooling (Low Impact, Low Work)

| Step | Task                                        | Impact | Work   | Notes                      |
| ---- | ------------------------------------------- | ------ | ------ | -------------------------- |
| 6.1  | Update AGENTS.md with cache troubleshooting | LOW    | 20 min | Document what we learned   |
| 6.2  | Add `just doctor` command                   | LOW    | 15 min | Check prerequisites        |
| 6.3  | Clean up old status reports                 | LOW    | 5 min  | Archive or delete old ones |

---

## 3. Code Review: What We Already Have That Fits

### Existing Good Patterns

- `domain.HTML` type for safe HTML strings ✅
- `RenderedContent` struct bundling related data ✅
- `domain.Renderer` interface for abstraction ✅
- `samber/do` for dependency injection ✅
- `samber/lo` already in go.mod! (not fully utilized) ✅

### Gaps to Fill

- No `Result[T]` type for error handling
- No `Option[T]` type for optional values
- Cache uses otter, but ristretto is more battle-tested
- Not using `lo` for functional operations
- Config validation is manual

---

## 4. Type Architecture Improvements

### Current State

```go
type Renderer interface {
    Render(source []byte) (RenderedContent, error)
}

type RenderedContent struct {
    HTML       HTML
    TOC        []TOCItem
    Metadata   Frontmatter
    HasMermaid bool
}
```

### Proposed Improvements

#### A. Add Result[T] type

```go
type Result[T any] struct {
    value T
    err   error
}

func (r Result[T]) IsOk() bool { return r.err == nil }
func (r Result[T]) Value() (T, bool) { return r.value, r.IsOk() }
func (r Result[T]) Unwrap() T {
    if r.err != nil { panic(r.err) }
    return r.value
}
```

#### B. Add Option[T] type

```go
type Option[T any] struct {
    value T
    valid bool
}

func Some[T any](v T) Option[T] { return Option[T]{v, true} }
func None[T any]() Option[T] { return Option[T]{} }
```

#### C. Improve Frontmatter

```go
type Frontmatter struct {
    Title       Option[string]
    Description Option[string]
    Author      Option[string]
    Date        Option[time.Time]
    Tags        []string  // empty slice = none
    Draft       bool      // default false
}
```

---

## 5. Established Libraries We Should Use

### Already in go.mod (Underutilized)

1. **github.com/samber/lo** — Functional programming utilities
   - Use `lo.Map`, `lo.Filter`, `lo.Reduce` instead of custom loops
   - Use `lo.Must`, `lo.Try` for error handling
2. **github.com/cockroachdb/errors** — Better error handling
   - Use `errors.Wrap`, `errors.Newf` everywhere
   - Stack traces automatically

### Should Add

3. **github.com/samber/mo** — Monads (Option, Result, Either)
   - OR implement our own (simple enough)
4. **github.com/dgraph-io/ristretto** — Production cache
   - More battle-tested than otter
   - Better metrics and eviction policies

5. **github.com/go-playground/validator** — Struct validation
   - Tag-based validation
   - Better than manual checks

---

## 6. Questions I Cannot Figure Out

### Primary Question

**How do I reliably fix the Go build cache corruption when:**

- `rm -rf` fails with "Directory not empty" even after killing all processes
- Some directories appear to have invisible files
- `go clean -cache` doesn't fully clean
- macOS APFS might be caching directory metadata?

**Options I haven't tried:**

- Reboot the system (nuclear option)
- Use `sudo` for rm (need permission)
- Use disk utility to check filesystem
- Change GOCACHE to a completely different directory permanently

**What I need:**
A command or approach that will 100% fix this without requiring system reboot.

---

## 7. Immediate Next Actions (If Cache Fixed)

1. **Verify types_test.go fix** — `go test ./internal/domain/...`
2. **Add cache-clean to justfile** — Document the fix
3. **Run full test suite** — `go test ./...`
4. **Run linter** — `just lint`
5. **Commit any fixes** — One per logical change
6. **Push to origin** — `git push`

---

_Plan generated: 2026-04-02 17:01_
