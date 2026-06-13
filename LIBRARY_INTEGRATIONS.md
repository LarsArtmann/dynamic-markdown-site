# Library Integration Report — Dynamic Markdown Site

**Generated:** 2026-05-13 | **Updated:** 2026-06-13 | **Codebase:** `github.com/larsartmann/dynamic-markdown-site`

---

## Current Library Usage Analysis (2026-06-13)

### Version Status: All Direct Dependencies Updated

All 5 outdated direct dependencies have been updated to latest:

| Library              | Was       | Now     | Notes                                         |
| -------------------- | --------- | ------- | --------------------------------------------- |
| alecthomas/chroma/v2 | v2.23.1   | v2.26.1 | Syntax highlighting                           |
| cockroachdb/errors   | v1.12.0   | v1.13.0 | Error wrapping                                |
| fsnotify/fsnotify    | v1.9.0    | v1.10.1 | File watcher                                  |
| larsartmann/httputil | v0.0.0-\* | v0.2.0  | HTTP middleware (breaking: `Middleware` type) |
| gocloud.dev          | v0.40.0   | v0.46.0 | Blob storage                                  |

### Usage Depth Assessment

| Library                   | Usage Level      | What's Used                                                          | Untapped Potential                                                                                            |
| ------------------------- | ---------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| **charm.land/log/v2**     | Adequate         | Level setting, formatter (text/JSON), slog.Handler wrapping          | Child loggers with `With()` for component context (available via slog)                                        |
| **a-h/templ**             | Well used        | Component rendering, template-to-ResponseWriter, 13 components       | None significant                                                                                              |
| **chroma/v2**             | Minimal          | Monokai style, line numbers disabled                                 | `WithClasses(true)` (CSS-based, smaller HTML), `TabWidth()`, `HighlightLines()`, line linking, custom lexers  |
| **cockroachdb/errors**    | Adequate         | `Wrap`, `Wrapf`, `New`, `Is` in 14 files                             | `WithDetail()`, `WithHint()` for actionable user-facing error messages; `Assert()`/`Assertf()` for invariants |
| **fsnotify**              | Well used        | Watcher lifecycle, events, errors, recursive dirs, debounce          | None significant                                                                                              |
| **httputil**              | **Now improved** | Recovery, Compression, RequestID, ResponseRecorder, ClientIP, Chain  | `ETag()` (conditional GETs), `CORS()`, `Timeout()`, `Server` (graceful lifecycle), `HealthHandler`            |
| **otter/v2**              | Well used        | Get, Set, GetWithLoader, InvalidateAll, Stats, EstimatedSize, Close  | None significant                                                                                              |
| **samber/do/v2**          | Well used        | Provide, Invoke, Shutdown, ShutdownReport                            | `ProvideNamedValue`, `InjectableScope` (not needed at this scale)                                             |
| **samber/lo**             | Minimal          | `FilterMap`, `ContainsBy` (2 of 200+ functions)                      | `Map`, `Filter`, `Reduce`, `CoalesceOrEmpty`, `Ternary`, `Must` — many manual loops could use lo              |
| **testify**               | Well used        | `assert` + `require` extensively (~94 calls across 4 test files)     | None significant                                                                                              |
| **goldmark**              | Well used        | 7 extensions, 2 custom extensions, AST walking, TOC extraction       | `goldmark.WithRendererOptions()` for custom HTML rendering rules                                              |
| **goldmark-highlighting** | Minimal          | Monokai style, no line numbers                                       | `WithFormatOptions()` — same chroma opportunities as above                                                    |
| **goldmark-meta**         | Well used        | Frontmatter parsing, metadata extraction                             | None significant                                                                                              |
| **gocloud.dev**           | Adequate         | Blob: OpenBucket, List, NewReader, Exists, Attributes, Close         | `blob.SignedURL()` for pre-signed download URLs (if serving from S3/GCS)                                      |
| **x/time/rate**           | Adequate         | Token bucket: `NewLimiter`, `Allow`, per-IP map                      | `SetLimit()`/`SetBurst()` for dynamic rate adjustment                                                         |
| **yaml.v3**               | Minimal          | `Unmarshal` for draft detection only                                 | Could be replaced by goldmark-meta frontmatter parsing to eliminate the dependency entirely                   |
| **d2**                    | Well used        | Compile, render to SVG, textmeasure, 11 render options, dagre layout | Custom themes, multi-board diagrams, animations (tooling-level, not library-level)                            |

### Improvements Implemented This Session

1. **httputil.Recovery()** added — critical panic recovery middleware we were missing entirely
2. **httputil.Compression()** added — gzip response compression for all responses
3. **httputil.RequestID()** replaced custom `requestIDMiddleware()` — eliminated ~50 lines of duplicate code
4. **Middleware ordering fixed** — RequestID now runs BEFORE accessLog, so request_id is no longer empty in logs (was a bug)
5. **Depguard config cleaned** — removed unused `gin-gonic/gin`, added `httputil` and `x/time/rate`

### Remaining Opportunities (Not Implemented)

| Opportunity                     | Impact | Effort | Notes                                                                                          |
| ------------------------------- | ------ | ------ | ---------------------------------------------------------------------------------------------- |
| httputil.ETag()                 | Medium | Low    | Conditional GETs — 304 responses for unchanged content, saves bandwidth                        |
| httputil.Timeout()              | Low    | Low    | Request-level timeout via context (currently using http.Server timeout which kills connection) |
| chroma WithClasses(true)        | Medium | Medium | Switches from inline styles to CSS classes — smaller HTML, but requires generating monokai CSS |
| cockroachdb WithHint/WithDetail | Medium | Medium | Actionable error messages for API consumers (/refresh, /search)                                |
| yaml.v3 removal                 | Low    | Low    | goldmark-meta already parses frontmatter; draft check could use that instead                   |
| samber/lo expansion             | Low    | Low    | Many manual loops could be simplified, but risk of churn vs readability tradeoff               |

---

## Executive Summary

The project uses **17 direct dependencies** from the Go ecosystem (Gin, Goldmark, Chroma, Templ, Otter, fsnotify, samber/do, samber/lo, cockroachdb/errors, gocloud.dev, D2, YAML, charm.land/log). Of the **19 ecosystem libraries** documented in `LIBRARY_GUIDE.md`, **none are currently used**. This report assesses each library's fit and identifies where adoption would provide meaningful value vs. where the current approach is appropriate.

**Key finding:** 5 libraries offer clear, actionable improvements. 8 are irrelevant to this project's scope. 6 are borderline — nice-to-have but not worth the integration cost at current project scale.

---

## Quick Reference: Integration Decisions

| Library                                                          | Current State                                             | Verdict               | Priority | Rationale                                                                |
| ---------------------------------------------------------------- | --------------------------------------------------------- | --------------------- | -------- | ------------------------------------------------------------------------ |
| [go-filewatcher](#1-go-filewatcher)                              | Raw `fsnotify` with no debouncing                         | **Adopt**             | High     | Fixes real bug: no debouncing causes N refreshes on bulk file ops        |
| [go-error-family](#2-go-error-family)                            | `cockroachdb/errors` + sentinel errors, no classification | **Adopt**             | High     | Adds behavioral error classification the codebase needs at HTTP boundary |
| [smart-configs](#3-smart-configs)                                | Hand-rolled `flag` + env var parsing                      | **Adopt**             | Medium   | Eliminates ~150 lines of custom parsing, adds actionable error messages  |
| [templ-components](#4-templ-components)                          | 13 hand-built components                                  | **Adopt selectively** | Medium   | Replace 4 generic components; keep 9 domain-specific ones                |
| [cmdguard](#5-cmdguard)                                          | Raw `flag` package in `config.Load()`                     | **Consider**          | Low      | Overkill for single-command server; value if CLI grows subcommands       |
| [go-business-rules](#6-go-business-rules)                        | Constructor validation only                               | **Consider**          | Low      | Config validation would benefit; most validation is domain-constructors  |
| [go-output](#7-go-output)                                        | N/A                                                       | **Skip**              | —        | No CLI output formatting needed; web server serves HTML                  |
| [go-commit](#8-go-commit)                                        | N/A                                                       | **Skip**              | —        | Developer tooling, not a dependency                                      |
| [universal-workflow](#9-universal-workflow)                      | N/A                                                       | **Skip**              | —        | No multi-step orchestration in this project                              |
| [ActaFlow](#10-actaflow)                                         | N/A                                                       | **Skip**              | —        | No actor model needed; request-scoped HTTP server                        |
| [cqrs-htmx](#11-cqrs-htmx)                                       | Gin + hand-built handlers                                 | **Skip**              | —        | No CQRS/event sourcing; Gin + templ already working well                 |
| [go-composable-business-types](#12-go-composable-business-types) | N/A                                                       | **Skip**              | —        | No business data modeling; content files, not domain entities            |
| [project-discovery-sdk](#13-project-discovery-sdk)               | Custom `FileSystemRepository`                             | **Skip**              | —        | Different scope: project scanning vs. markdown serving                   |
| [go-cqrs-lite](#14-go-cqrs-lite)                                 | N/A                                                       | **Skip**              | —        | No event sourcing, no aggregates, no projections                         |
| [go-localfirst](#15-go-localfirst)                               | N/A                                                       | **Skip**              | —        | No offline/sync requirements                                             |
| [go-localsync](#16-go-localsync)                                 | N/A                                                       | **Skip**              | —        | No external API sync                                                     |
| [go-finding](#17-go-finding)                                     | N/A                                                       | **Skip**              | —        | No static analysis pipeline                                              |
| [go-branded-id](#18-go-branded-id)                               | N/A                                                       | **Skip**              | —        | No ID types in domain; paths are the identifiers                         |
| [gogenfilter](#19-gogenfilter)                                   | N/A                                                       | **Skip**              | —        | No generated code detection needed                                       |

---

## Detailed Analysis

### 1. go-filewatcher

**Current implementation:** `cmd/dynamic-markdown-site/watcher.go` uses raw `fsnotify` with:

- Manual recursive directory watching via `addDirectoriesRecursive()`
- Custom `shouldTriggerRefresh()` filter for `.md`/`.markdown` extensions
- **No debouncing** — every filesystem event triggers a `repo.Refresh()` + `liveReload.Notify("")`
- No retry/recovery if the watcher goroutine dies

**Why adopt:**

| Problem                   | Impact                                                                         | go-filewatcher fix                               |
| ------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------ |
| No debouncing             | Bulk operations (`git checkout`, IDE auto-save) trigger N sequential refreshes | Per-path or global debouncing built in           |
| Manual recursive watching | ~60 lines of hand-rolled add/filter/skip logic                                 | Automatic recursive watching with smart defaults |
| No recovery               | Watcher goroutine exits silently on error; no restart path                     | Middleware chains with recovery                  |
| Limited filtering         | Only `.md`/`.markdown` check                                                   | 15+ composable filters with AND/OR/NOT           |

**Integration plan:**

- Replace `cmd/dynamic-markdown-site/watcher.go` (~120 lines) with ~20 lines of go-filewatcher config
- Use `GlobalDebouncer` with 500ms delay to coalesce bulk events
- Use `FilterOr(HasExt(".md"), HasExt(".markdown"))` for file type filtering
- Keep existing `repo.Refresh()` + `liveReload.Notify()` as the handler

**Risk:** Low. Drop-in replacement for a self-contained module with no downstream dependencies.

**Effort:** ~2 hours.

---

### 2. go-error-family

**Current implementation:**

- `cockroachdb/errors` for wrapping everywhere (14 files)
- 8 sentinel errors, but only `ErrContentNotFound` is ever checked with `errors.Is()` at the handler level
- Error → HTTP status mapping is binary: 404 for `ErrContentNotFound`, 500 for everything else
- No structured error context, no retry hints, no error codes
- Mixed wrapping: `cockroachdb/errors` in most files, `fmt.Errorf(%w)` in `container.go`

**Why adopt:**

| Problem                           | Impact                                                                                      | go-error-family fix                                                                                           |
| --------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| No behavioral classification      | All non-404 errors are 500; no distinction between transient (retry) and permanent (bug)    | 5-level Family classification: Rejection→400, Conflict→409, Transient→503, Corruption→422, Infrastructure→500 |
| No error codes                    | Error messages are free-form strings; not machine-readable                                  | `Coded` interface: `ErrorCode() string` → `db.timeout`, `content.not_found`                                   |
| No retry signaling                | Clients (or internal retry logic) can't determine if an error is worth retrying             | `Retryable` interface + `Family.IsRetryable()`                                                                |
| No user-facing error presentation | Raw error strings leak in JSON responses (e.g., `/refresh` endpoint exposes `result.Error`) | `HandleError()` generates What/Why/Fix messages                                                               |
| Mixed error packages              | `cockroachdb/errors` aliased as `cockroachdberrors` to avoid collision                      | Protocol-based: import only interfaces, keep cockroachdb for wrapping                                         |

**Integration plan:**

1. Define domain error families:
   - `ErrContentNotFound` → `Rejection` (client error, 404)
   - `ErrInvalidPath` → `Rejection` (client error, 400)
   - `ErrInvalidRoot` → `Corruption` (config error, 500)
   - `errRefreshFailed` → `Transient` (retryable, 503)
   - `errBlobTimeout` → `Transient` (retryable, 503)
2. Map families to HTTP status in `handlers.go` using `family.ExitCode()` or custom mapping
3. Use `Classify(err)` for the generic error handler to get retry hints + proper status
4. Replace raw `result.Error` exposure in `/refresh` JSON with `HandleError()` presentation
5. Keep `cockroachdb/errors` for wrapping (go-error-family is a protocol, not a replacement)

**Risk:** Low. Additive — wraps existing error infrastructure without breaking `errors.Is()` checks.

**Effort:** ~4 hours.

---

### 3. smart-configs

**Current implementation:** `internal/config/config.go` (~280 lines):

- `flag` package for CLI flags
- Hand-rolled `applyEnvVar` functions with custom `parseBool`, `parseUint16` helpers
- Manual env var prefix (`DYNAMIC_MARKDOWN_`) handling
- `strconv.ParseBool` equivalent hand-rolled as `parseBool`
- No config file support, no actionable error messages

**Why adopt:**

| Problem                                  | Impact                                                                                        | smart-configs fix                                                 |
| ---------------------------------------- | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| ~80 lines of env var parsing boilerplate | Maintenance burden, subtle bugs (e.g., `parseBool` silently treats unknown values as `false`) | Multi-source resolution built in: CLI → env → `.env` → defaults   |
| Silent failure on invalid env values     | `DYNAMIC_MARKDOWN_CACHE=garbage` → silently disables caching                                  | Actionable error messages with copy-paste fix commands            |
| No `.env` file support                   | Developers must set env vars manually or use shell scripts                                    | Built-in `.env` file resolution                                   |
| No CI/CD context awareness               | Generic error messages regardless of environment                                              | Auto-detects GitHub Actions, Docker, K8s for tailored suggestions |

**Caveats:**

- smart-configs has its own API and patterns — would require rewriting the config package
- Current config is functional and tested; risk of regression during migration
- The `flag` integration may need adaptation since smart-configs likely has its own flag mechanism

**Integration plan:**

1. Replace `applyEnvOverrides()` + `applyEnvBool` + `applyEnvString` + `applyEnvDuration` (~80 lines) with smart-configs resolver chain
2. Keep `defineAndParseFlags()` for now; smart-configs can wrap flag sources
3. Replace `validate()` with smart-configs validation hooks
4. Add `.env` file support for local development

**Risk:** Medium. Core config is foundational; thorough testing required.

**Effort:** ~6 hours.

---

### 4. templ-components

**Current implementation:** 13 hand-built templ components in `templates/layout.templ`:

- `Layout`, `Header`, `Breadcrumbs`, `Footer` (structural)
- `DirectoryView`, `FileView`, `ErrorView`, `SearchView` (pages)
- `ContentCard`, `SearchResultCard` (cards)
- `TableOfContents`/`TOCItem` (TOC)
- `LiveReloadScript`, `MermaidScript` (utility)
- ~1061 lines of hand-written CSS with custom design system

**Why adopt selectively:**

| Component                    | Current                       | templ-components                                          | Worth replacing?                                                |
| ---------------------------- | ----------------------------- | --------------------------------------------------------- | --------------------------------------------------------------- |
| `Breadcrumbs`                | 20-line hand-built nav        | `navigation.Breadcrumbs` with typed props                 | **Yes** — standard pattern, better accessibility                |
| `ContentCard`                | Card with icon + title + meta | `display.Card` or `display.SimpleCard`                    | **Yes** — standard card, dark mode free                         |
| `SearchResultCard`           | Similar to ContentCard        | `display.Card` variant                                    | **Yes** — deduplicate card patterns                             |
| `ErrorView`                  | Custom error page             | Could use `display.EmptyState` or compose from primitives | **Maybe** — error pages are brand-specific                      |
| Search input + button        | Raw HTML form                 | `forms.Input` + button                                    | **Maybe** — small surface area                                  |
| `Layout`                     | Full HTML5 shell with SRI     | `layout.Base` with SRI, theme, meta                       | **Maybe** — but markdown site needs specific `<head>` structure |
| `Header` / `Footer`          | Custom nav/footer             | `navigation.Nav` / `navigation.Footer`                    | **Maybe** — brand-specific but would get dark mode free         |
| `TableOfContents`            | Recursive TOC from headings   | No direct equivalent                                      | **No** — domain-specific                                        |
| `LiveReloadScript`           | SSE dev script                | No equivalent                                             | **No** — dev tooling                                            |
| `MermaidScript`              | Mermaid.js loader             | No equivalent                                             | **No** — domain-specific                                        |
| `DirectoryView` / `FileView` | Page compositions             | No page-level components                                  | **No** — app-specific composition                               |

**Caveats:**

- templ-components brings Tailwind CSS dependency — the project currently has ~1061 lines of custom CSS. Adopting templ-components means maintaining **both** Tailwind and custom CSS, or migrating the entire stylesheet.
- Markdown content styles (`.markdown-content *`) are deeply custom and wouldn't benefit from a component library.
- Dark mode support is a nice-to-have but not a stated requirement.
- CSP nonce support in templ-components is valuable if CSP headers are planned.

**Integration plan:**

1. Add `templ-components` as dependency
2. Replace `Breadcrumbs` with `navigation.Breadcrumbs` (lowest risk, clearest win)
3. Replace `ContentCard` + `SearchResultCard` with `display.Card` variants
4. Keep `Layout`, `Header`, `Footer`, `ErrorView`, and all domain-specific components as-is
5. Assess Tailwind adoption separately — don't mix Tailwind utility classes with the existing custom CSS without a migration plan

**Risk:** Medium. Visual regression risk; requires visual QA after each component swap.

**Effort:** ~4-8 hours (including visual QA).

---

### 5. cmdguard

**Current implementation:** Single `main.go` with `run()` → `setupServices()` → `serveHTTP()` → `gracefulShutdown()` flow. Config via `flag` package in separate `config.Load()`.

**Why it's "Consider" not "Adopt":**

| Factor             | Assessment                                                       |
| ------------------ | ---------------------------------------------------------------- |
| Command complexity | Single command (server). cmdguard shines with multi-command CLIs |
| DI                 | Already using `samber/do/v2` which handles DI well               |
| Typed flags        | Config struct already has typed fields after parsing             |
| Health checks      | Already implemented via `/health` endpoint                       |
| Graceful shutdown  | Already implemented via signal.NotifyContext                     |
| Output formats     | N/A — this is a web server, not a CLI tool                       |

**When to reconsider:** If the project adds subcommands (e.g., `dynamic-markdown-site serve`, `dynamic-markdown-site migrate`, `dynamic-markdown-site validate`), cmdguard would eliminate significant boilerplate.

**Effort if adopted:** ~8 hours for full migration. Not justified for current scope.

---

### 6. go-business-rules

**Current implementation:** Constructor functions (`NewURLPath`, `NewDirectoryNode`, `NewFileNode`) return errors for invalid input. Config validation in `validate()` checks port, root dir, log level.

**Why it's "Consider" not "Adopt":**

| Factor             | Assessment                                                               |
| ------------------ | ------------------------------------------------------------------------ |
| Validation scope   | Most validation is in domain constructors (correct pattern)              |
| Severity levels    | No current need — validation errors are always fatal                     |
| Config validation  | 3 checks; too small to justify a library                                 |
| JSON serialization | Not needed — validation errors surface as HTTP errors, not API responses |

**When to reconsider:** If the project adds content validation rules (e.g., markdown linting, link checking, frontmatter schema validation) with severity levels (warn vs. error).

**Effort if adopted:** ~3 hours. Low risk but low value at current scope.

---

## Existing Integration Quality

### Libraries Used Well

| Library               | Usage                                        | Quality                                                                          |
| --------------------- | -------------------------------------------- | -------------------------------------------------------------------------------- |
| **a-h/templ**         | Type-safe templates with props structs       | Excellent — proper compile-time checks, clean props pattern                      |
| **yuin/goldmark**     | Markdown rendering with 8 extensions         | Excellent — custom extensions for diagrams and admonitions show deep integration |
| **maypok86/otter**    | HTML response caching with stats             | Good — proper TTL, stats tracking, cache invalidation                            |
| **samber/do/v2**      | Dependency injection container               | Good — lazy providers, graceful shutdown, clean separation                       |
| **charm.land/log**    | Structured logging via slog                  | Good — dev/production formatter switch, slog interface compliance                |
| **alecthomas/chroma** | Syntax highlighting via goldmark integration | Adequate — minimal direct usage (monokai style only)                             |
| **gocloud.dev/blob**  | Multi-backend storage (local, S3, GCS)       | Good — portable URL scheme, driver registration, timeout protection              |

### Libraries with Improvement Opportunities

| Library                    | Issue                                                                                                   | Recommendation                                                                   |
| -------------------------- | ------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| **cockroachdb/errors**     | Double-wrapping bugs (`Wrapf` + `%w` in same call); mixed with `fmt.Errorf` in container                | Audit all `Wrapf` calls; remove redundant `%w` verbs                             |
| **fsnotify**               | No debouncing, no recovery, no filters beyond file extension                                            | Replace with go-filewatcher                                                      |
| **gin-gonic/gin**          | Route handling split between middleware and routes (`staticAndContentMiddleware` duplicates routing)    | Refactor: move content routing to proper Gin handlers, simplify middleware chain |
| **samber/lo**              | Only 3 uses across codebase; `lo.Filter` in rate limiter allocates on every request                     | Acceptable for current scope; consider in-place filtering for hot paths          |
| **gopkg.in/yaml.v3**       | Minimal usage (draft detection only) but duplicates frontmatter parsing that goldmark-meta already does | Consolidate: use goldmark-meta for all frontmatter extraction                    |
| **oss.terrastruct.com/d2** | Graceful degradation on init failure (good), but fallback is silent                                     | Add explicit log warning when D2 support is disabled                             |

---

## Implementation Priority

```
Phase 1 (High value, Low risk):
├── go-filewatcher    → Fix debouncing bug in dev mode
└── go-error-family   → Add behavioral error classification at HTTP boundary

Phase 2 (Medium value, Medium risk):
├── smart-configs     → Replace hand-rolled env var parsing
└── templ-components  → Replace Breadcrumbs + Card components

Phase 3 (Future consideration):
├── cmdguard          → If CLI grows subcommands
└── go-business-rules → If content validation rules are added
```

---

## Dependency Impact

| Action           | New direct deps                                  | Removed deps | Net change                       |
| ---------------- | ------------------------------------------------ | ------------ | -------------------------------- |
| go-filewatcher   | go-filewatcher, gogenfilter (transitive)         | fsnotify     | -1 direct, +2 transitive         |
| go-error-family  | go-error-family                                  | none         | +1 direct (zero transitive deps) |
| smart-configs    | smart-configs, go-branded-id (transitive)        | none         | +1 direct, +1 transitive         |
| templ-components | templ-components, tailwind-merge-go (transitive) | none         | +1 direct, +1 transitive         |

**Total if all Phase 1+2 adopted:** +4 direct dependencies, +4 transitive dependencies, -1 direct dependency (fsnotify removed).

---

_This report is based on codebase analysis as of 2026-05-13. Library capabilities sourced from `/home/lars/projects/docs/LIBRARY_GUIDE.md`._
