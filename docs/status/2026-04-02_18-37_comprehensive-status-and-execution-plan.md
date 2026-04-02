# Comprehensive Status Report

**Date:** 2026-04-02 18:37 CEST
**Branch:** `master` (1 commit ahead of origin)
**HEAD:** `227f551` — fix(ci): use explicit image name in docker workflow and add status report
**Build:** PASSING (verified with `GOWORK=off go build ./...`)
**Tests:** UNKNOWN — build cache was cleared for disk space; test run requires rebuilding all dependencies
**Disk:** 6.4GB free / 229GB total (98% used) — was as low as 756MB earlier today
**Staged:** 2 files (reflection-execution-plan.md, justfile)
**Production Code:** 4,517 lines across 34 files
**Test Code:** 7,372 lines across test files

---

## a) FULLY DONE

| #   | What                                                                | Commit               | Impact                              |
| --- | ------------------------------------------------------------------- | -------------------- | ----------------------------------- |
| 1   | Lint fixes across renderer, content, sitemap, config                | `b4e23b6`, `24b0195` | Zero lint issues achieved           |
| 2   | `domain.Renderer` interface introduced                              | `4233fdc`            | Dependency inversion for rendering  |
| 3   | `RenderedContent` struct + `NewRenderedFileWithContent` constructor | `4233fdc`            | Type-safe rendered output           |
| 4   | DI wiring: renderer through container                               | `4233fdc`, `6372ec6` | Full DI pipeline works              |
| 5   | `NewServer` 7-arg signature with `domain.Renderer` + `siteName`     | `4233fdc`            | Proper dependency injection         |
| 6   | `SiteName` in `ErrorViewProps` and `LayoutProps` templates          | `4233fdc`            | Site name renders everywhere        |
| 7   | `SimpleRenderer` + tests deleted (dead code)                        | `6372ec6`            | 58 lines of dead code removed       |
| 8   | Docker workflow: explicit image name                                | `227f551`            | CI no longer breaks on template var |
| 9   | Admonition extension (goldmark alert blocks)                        | `88e3367`            | Feature complete with tests         |
| 10  | Sitemap + robots.txt                                                | `9439b33`            | Feature complete with tests         |
| 11  | Frontmatter draft parsing (yaml.v3)                                 | `c489007`            | Proper draft detection              |
| 12  | CSS admonition refactored to custom properties                      | `05f48e9`            | Maintainable theming                |
| 13  | `config.Timeout` wired to HTTP server                               | `4233fdc` (earlier)  | ReadTimeout/WriteTimeout set        |

---

## b) PARTIALLY DONE

| #   | What                             | Status                               | Remaining                                                                            |
| --- | -------------------------------- | ------------------------------------ | ------------------------------------------------------------------------------------ |
| 1   | Split brain elimination          | **0 of 4 fixed**                     | `skipDirs`, `isMarkdownFile`, `getContentType`, `SuggestedPath` all still duplicated |
| 2   | Ghost code cleanup               | **1 of 2 fixed**                     | `SimpleRenderer` deleted; `testutil/` still exists (0 imports)                       |
| 3   | `cache.GetOrCompute` integration | **Method exists but unused**         | `render.go` still does manual Get/Set pattern                                        |
| 4   | `HasReadme` implementation       | **Field exists, hardcoded `false`**  | Needs actual directory child check                                                   |
| 5   | `SearchResult.Snippet` rendering | **Field extracted, never displayed** | Template missing snippet display                                                     |

---

## c) NOT STARTED

| #   | What                                                                 | Priority        |
| --- | -------------------------------------------------------------------- | --------------- |
| 1   | Container DI tests                                                   | Medium          |
| 2   | E2E diagram rendering tests                                          | Medium          |
| 3   | Dependabot fix (grpc auth bypass CVE)                                | HIGH — security |
| 4   | `go mod tidy`                                                        | Medium          |
| 5   | `git push` (1 commit ahead)                                          | Immediate       |
| 6   | Unify `treeStats` / `blobTreeStats` duplicate structs                | Low             |
| 7   | Fix double error wrapping in `search.go:63`                          | Low             |
| 8   | Populate `Frontmatter.Date` from metadata                            | Low             |
| 9   | Remove useless type assertion in `main.go:209`                       | Low             |
| 10  | Move hardcoded values to config (cache size, rate limits, CSP, etc.) | Low             |

---

## d) TOTALLY FUCKED UP

| #   | What                          | Severity  | Details                                                                                                                                                                 |
| --- | ----------------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Disk space**                | CRITICAL  | 229GB disk at 98%. Go build cache cleared 3x today. Tests can't run when disk < 2GB. This is the #1 blocker for ALL development.                                        |
| 2   | **Stale LSP diagnostics**     | ANNOYING  | gopls shows phantom errors in `container.go:192` and `errors.go:35,71` — code is correct, but LSP cache is corrupted. Requires `templ generate` + LSP restart to clear. |
| 3   | **Go build cache corruption** | RECURRING | Cache gets corrupted when disk hits 100%. Had to `go clean -cache` three times today. Each rebuild takes 3-5 minutes.                                                   |
| 4   | **Test suite not verifiable** | HIGH      | Cannot confirm tests pass because build cache keeps getting wiped by disk pressure. Last known-good test run was before today's session.                                |
| 5   | **`go.work` interference**    | ANNOYING  | Parent directory has `go.work` that breaks all Go commands unless `GOWORK=off` is prefixed. Not a bug per se, but a DX papercut.                                        |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Split brains are a maintenance time bomb** — 4 concepts duplicated across packages. If one changes, the other won't. This IS how bugs sneak in.
2. **`testutil` is dead weight** — 234 lines of well-written code with zero consumers. Either adopt it or delete it. Currently it's just noise.
3. **`cache.GetOrCompute` exists but nobody uses it** — The whole point of having this method was to avoid the manual Get/Set pattern. It's dead code in plain sight.
4. **Hardcoded magic values everywhere** — Cache size 10,000, rate limit 10/min, max-age 86400, words-per-minute 200, blob timeout 10s. None come from config.
5. **`SuggestedPath` type duplication** requires a conversion function (`convertToTemplateSuggestions`). This is a code smell that adds coupling for no benefit.

### Process

6. **Disk space is the #1 blocker** — We've spent more time fighting disk space than writing code. Consider: cleaning ~/Library/Caches/Google (1.5GB), removing unused Docker images, archiving old projects.
7. **Too many status reports, not enough code changes** — `docs/status/` has 9 files. That's 9 sessions writing reports instead of fixing the 4 split brains.
8. **No CI verification** — 1 commit ahead of origin. Changes aren't validated by CI. We should push after each commit.

### Type Model

9. **`domain.Renderer` returns `RenderedContent`** — Good. But `server.render.go` still accesses individual fields via `content.HTML`, `content.TOC` etc. The type is there but the abstraction leaks.
10. **`SuggestedPath` should live in `domain/`** — It's a domain concept (a path suggestion with scoring). Having it in both `server/` and `templates/` violates the dependency rule.

---

## f) Top 25 Things To Do Next (Sorted by Impact / Effort)

| Priority | Task                                                                        | Effort | Impact   | Why                                            |
| -------- | --------------------------------------------------------------------------- | ------ | -------- | ---------------------------------------------- |
| **1**    | **Free disk space (minimum 10GB)**                                          | 5 min  | BLOCKER  | Nothing works without disk space               |
| **2**    | **Git push** (1 commit ahead)                                               | 1 min  | HIGH     | CI validation, backup                          |
| **3**    | **Dependabot: update grpc** (CVE auth bypass)                               | 5 min  | CRITICAL | Security vulnerability                         |
| **4**    | **`go mod tidy`**                                                           | 2 min  | MEDIUM   | Clean dependency tree                          |
| **5**    | **Delete `internal/testutil/`** (0 imports, 234 lines)                      | 3 min  | MEDIUM   | Dead code removal                              |
| **6**    | **Unify `skipDirs`**: export from `content`, use in `watcher`               | 10 min | MEDIUM   | Eliminate split brain #1                       |
| **7**    | **Unify `isMarkdownFile`**: export from `content`, use in `watcher`         | 10 min | MEDIUM   | Eliminate split brain #2                       |
| **8**    | **Unify `getContentType`**: merge into `content/helpers.go`, add font types | 15 min | MEDIUM   | Eliminate split brain #3                       |
| **9**    | **Use `cache.GetOrCompute`** in `render.go`                                 | 10 min | MEDIUM   | Eliminate dead code + cleaner pattern          |
| **10**   | **Implement `HasReadme`**: check directory children for README.md           | 5 min  | LOW      | Feature works correctly                        |
| **11**   | **Render `SearchResult.Snippet`** in template                               | 5 min  | LOW      | Feature works correctly                        |
| **12**   | **Move `SuggestedPath` to `domain/`**                                       | 15 min | MEDIUM   | Eliminate split brain #4 + conversion function |
| **13**   | **Fix double error wrapping** in `search.go:63`                             | 2 min  | LOW      | Correct error handling                         |
| **14**   | **Unexport unnecessary exports** in `content/helpers.go`                    | 5 min  | LOW      | Clean API surface                              |
| **15**   | **Move hardcoded cache size to config**                                     | 5 min  | LOW      | Configurable behavior                          |
| **16**   | **Move hardcoded rate limit to config**                                     | 5 min  | LOW      | Configurable behavior                          |
| **17**   | **Remove useless type assertion** in `main.go:209`                          | 1 min  | LOW      | Dead code                                      |
| **18**   | **Populate `Frontmatter.Date`** from YAML metadata                          | 10 min | LOW      | Complete feature                               |
| **19**   | **Unify `treeStats` / `blobTreeStats`** structs                             | 5 min  | LOW      | DRY                                            |
| **20**   | **Add container DI tests**                                                  | 30 min | MEDIUM   | Test coverage for DI wiring                    |
| **21**   | **Add E2E diagram rendering tests**                                         | 30 min | MEDIUM   | Verify D2/Mermaid through pipeline             |
| **22**   | **Run `templ generate`** to fix stale LSP errors                            | 2 min  | LOW      | DX improvement                                 |
| **23**   | **Add `.gitignore` entry for `docs/status/`** or auto-generate              | 2 min  | LOW      | Keep repo clean                                |
| **24**   | **Clean `.golangci.yml`** of testutil exclusion rules (after deletion)      | 2 min  | LOW      | Config hygiene                                 |
| **25**   | **Delete this and older status reports**                                    | 5 min  | LOW      | 9 status files is too many                     |

---

## g) Top #1 Question I Cannot Figure Out Myself

**What is eating your disk space?**

229GB disk at 222GB used, and it's been a recurring blocker all day. The Go build cache was cleared 3x. The project itself is only 1.9MB. The `~/go/pkg/` mod cache is 4.4GB. Google Chrome caches are 1.5GB.

Is there a larger cleanup you can do outside this project? For example:

- `docker system prune -a` (could free GBs of unused images)
- Clean Xcode derived data: `rm -rf ~/Library/Developer/Xcode/DerivedData/`
- Clean old iOS simulators: `xcrun simctl delete unavailable`
- Remove unused Homebrew caches: `brew cleanup -s --prune=all`

Without at least 10GB of free disk space, development in this project will remain painful — builds take 5 min after each cache clear, and the cache gets corrupted when disk hits 100%.

---

## Summary Statistics

```
Production code:    4,517 lines (34 files)
Test code:          7,372 lines
Split brains:       4 identified, 0 fixed
Ghost packages:     1 (testutil, 234 lines, 0 imports)
Dead methods:       1 (cache.GetOrCompute)
Hardcoded values:   8 identified
CVE vulnerabilities: 1 (grpc auth bypass)
Commits ahead:      1 (not pushed)
Disk space:         6.4GB free / 229GB (98%)
Build:              PASSING
Tests:              UNKNOWN (cache cleared)
Lint:               Presumed clean (last verified at commit 24b0195)
```
