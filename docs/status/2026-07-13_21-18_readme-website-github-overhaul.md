# Status Report: README + Website + GitHub Metadata Overhaul

**Date:** 2026-07-13 21:18
**Session scope:** Make the public repo presentable — README.md, public wiki website, GitHub description/topics/URL

---

## A) FULLY DONE

### 1. README.md — Rewritten

- **Fixed Gin -> net/http**: Tech stack falsely claimed [Gin](https://gin-gonic.com/). The codebase uses standard `net/http` with Go 1.22+ method-based routing. Zero Gin dependency in `go.mod`.
- **Removed justfile section**: README documented `just build`, `just run-dev`, etc. No `justfile` exists in the repo. Replaced with Nix + direct Go commands.
- **Added cloud storage**: S3, GCS, Azure Blob support via `gocloud.dev` (`-storage-url` flag) was entirely absent. Now a headline feature with usage examples.
- **Fixed Docker section**: Claimed "multi-stage build" with `golang:1.26-alpine` builder stage. The actual `Dockerfile` is a single-stage distroless copy. Rewrote to match reality.
- **Added missing endpoints**: `/sitemap.xml`, `/robots.txt`, `/metrics`, `/cache/stats` were missing from the API table.
- **Added admonition blocks**: GitHub-style `> [!NOTE]` feature was undocumented.
- **Added release.yml**: CI section only mentioned 2 workflows; there are 3 (test, docker, release).
- **Fixed architecture tree**: Added `blob.go`, `admonition_extension.go`, `healthcheck.go`, `metrics.go`, `sitemap.go`. Removed non-existent `pkg/errors/`.

### 2. Website — `website/` (40 files)

Full Astro + Starlight documentation site, following the exact pattern from `go-atomic-write/website/` and `gogenfilter/website/`:

- **Landing page** (`src/pages/index.astro`): Hero with terminal code preview, 8-feature grid, 4-step how-it-works, 8-row comparison matrix (vs Static Site Gens and Wiki Engines), 3 use case cards, CTA section
- **10 doc pages**: Installation, Quick Start, Configuration, Docker, Cloud Storage, Markdown Features, API Endpoints, Changelog, Contributing, Related Tools
- **14 Astro components**: Header, Footer, Hero, FeatureGrid, HowItWorks, Comparison, UseCases, CTA, Section, SectionHeader, Card, Icon, Logo, Sections
- **Visual identity**: Indigo accent (#6366f1), Space Grotesk + JetBrains Mono fonts, dark/light theme toggle, scroll animations
- **Infra**: Firebase hosting config with security headers, Nix flake, tsconfig, htmlvalidate, sitemap, robots.txt, manifest.json, favicon.svg
- **Build result**: 0 errors, 0 warnings, 0 hints, 12 HTML pages generated, Pagefind search index built

### 3. GitHub Metadata

| Field        | Before                                                                                                 | After                                                                                                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Description  | "A blazing-fast markdown site generator with live reload, built in Go"                                 | "A Go server that turns markdown files into a navigable website with search, diagrams, live reload, and S3/GCS support"                                                  |
| Homepage URL | empty                                                                                                  | `https://dynamicmarkdown.lars.software`                                                                                                                                  |
| Topics       | 15 topics including wrong ones: `gin`, `static-site-generator`, `blog`, `hot-reload`, `site-generator` | 20 topics: removed 5 wrong ones, added 10 accurate ones (`net-http`, `s3`, `gcs`, `d2`, `mermaid`, `wiki`, `dynamic-site`, `full-text-search`, `blob-storage`, `docker`) |

---

## B) PARTIALLY DONE

### Website content depth

- Docs are accurate but could be deeper. The reference sites (go-atomic-write) have guide pages for benchmarks, error handling, platform support. This site has guides but they're more overview-level.
- No architecture diagram page (the project has D2 support — would be a great dogfooding opportunity).
- No benchmark/performance page.

### Git workflow

- Changes are uncommitted (`git status` shows `M README.md` and `?? website/`). Nothing was committed or pushed.
- No branch was created — everything is on `master`.

---

## C) NOT STARTED

- Committing the changes
- Deploying the website to Firebase
- Adding the website to the main project's CI (e.g., a workflow that builds the website on PRs)
- Adding a `.github/CODEOWNERS` file
- Creating GitHub release with proper release notes
- Adding Open Graph images (`og:image`) for social sharing — the website has OG tags but no actual OG image
- Adding a CONTRIBUTING.md link from the main repo README to the website's contributing page
- Favicon for the GitHub repo (social preview image)

---

## D) TOTALLY FUCKED UP

### License inconsistency — PRE-EXISTING, NOT FIXED

The project has a **proprietary** LICENSE file (`PROPRIETARY LICENSE ... Unauthorized copying ... strictly prohibited`), but:

1. `.goreleaser.yaml` declares `license: MIT` on **5 separate lines** (homebrew_casks, nfpms, nix, scoops sections)
2. `flake.nix` correctly says `licenses.unfree`
3. The website `package.json` correctly says `"license": "UNLICENSED"`
4. The website footer correctly says "Proprietary"

**This was not introduced by this session** — it's a pre-existing bug. But I noticed it and did NOT fix it. The `.goreleaser.yaml` is actively lying about the license. This could cause legal issues and confuse package repositories (Homebrew, Scoop, Nix) that rely on the declared license.

### bun.lock committed to git history risk

The `.gitignore` ignores `bun.lock`, but I used `bun install` which created `bun.lock`. If someone runs `git add website/` carelessly, it won't be added (gitignored). But the reference sites note "CI uses pnpm" — there's no pnpm-lock.json. The lockfile situation is unclear.

---

## E) WHAT WE SHOULD IMPROVE

1. **Fix `.goreleaser.yaml` license claims** — Change all `license: MIT` to the correct proprietary/unfree designation. This is actively misleading.
2. **Add OG image** — The website has `<meta property="og:image">` tags but no actual image. Social shares will have no preview.
3. **Dogfood D2 diagrams** — The project supports D2 diagram rendering. The website should include architecture diagrams rendered as D2 on the docs pages. This proves the product works.
4. **Add a "Live Demo" link** — The GitHub repo has no way to see the server running. Consider linking to a live instance or adding screenshots/GIFs to the README.
5. **Website CI workflow** — Add a GitHub Action that runs `astro check` and `astro build` on the website directory to catch breakage before merge.
6. **pnpm lockfile** — Decide on pnpm vs bun and commit a lockfile for reproducible CI builds.
7. **Landing page screenshot in README** — The reference repos don't do this, but for a "website generator" project, showing what the output looks like is critical.
8. **Test all internal links** — The website build succeeded, but there could be broken relative links in the docs content.
9. **Add a "Comparison" section to README** — The website has a great comparison matrix vs SSGs and wiki engines. This could be a strong selling point in the README too.
10. **AGENTS.md still references justfile** — The project AGENTS.md says "justfile is deprecated ... should be migrated to flake.nix" but also the old README had a just section. Check if any justfile references remain in other docs.

---

## F) Up to 50 Things We Should Get Done Next

### Immediate (this session's loose ends)

1. Commit the README changes
2. Commit the website directory
3. Fix `.goreleaser.yaml` license from MIT to proprietary/unfree
4. Add `website/` mention to the main README.md
5. Verify the website deploys to Firebase successfully

### Website polish

6. Add an OG/social preview image (1200x630 PNG)
7. Add a favicon set (apple-touch-icon, 32x32, 16x16 ICO)
8. Add screenshots of the actual dynamic-markdown-site UI to the website
9. Add a D2 architecture diagram to the docs (dogfooding)
10. Add a Mermaid flowchart to the quick-start guide
11. Write a "Performance" guide page with benchmark numbers
12. Write an "Architecture" guide page explaining the repository pattern, domain types, DI
13. Add search analytics or feedback mechanism
14. Add a copy-to-clipboard button on all code blocks in Starlight docs (may need expressiveCode config)
15. Add "Edit on GitHub" links to each doc page
16. Add "Last updated" dates to doc pages
17. Add a table of contents to the landing page (sticky sidebar?)
18. Improve mobile responsive testing
19. Add print styles for the docs
20. Add a 404 page that matches the landing design (not just Starlight default)

### README improvements

21. Add a "Screenshots" or "Demo" section with animated GIF
22. Add a "Comparison with alternatives" section (Hugo, MkDocs, BookStack)
23. Add badges (CI status, Go version, license, Docker pulls)
24. Add a "Quick Start" one-liner at the very top before the highlights
25. Link to the website documentation from the README
26. Add a "Projects using this" section (if any)
27. Add installation via Homebrew instructions (once the tap is set up)

### GitHub repo polish

28. Create a GitHub Social Preview image (1280x640)
29. Set up GitHub Pages as a redirect to the Firebase-hosted site
30. Add issue templates (bug report, feature request)
31. Add a SECURITY.md policy
32. Add a `.github/CODEOWNERS` file
33. Add a `FUNDING.yml` if applicable
34. Pin issues or create a welcome discussion
35. Create releases with proper release notes from CHANGELOG.md
36. Add branch protection rules for `master`
37. Enable GitHub Discussions for community Q&A

### Content accuracy

38. Audit all code examples in the docs for correctness
39. Verify every CLI flag documented in the website matches `config.go`
40. Add docs for the `-site-name` flag and `DYNAMIC_MARKDOWN_SITE_NAME` env var
41. Verify the Homebrew cask instructions work (`.goreleaser.yaml` has `skip_upload: true`)
42. Verify the Nix run command works with the proprietary license
43. Add docs for the `healthcheck` subcommand
44. Document the `/api/live-reload` SSE event format
45. Add a migration guide for users coming from Hugo/MkDocs

### Technical debt

46. Fix the `.goreleaser.yaml` `homebrew_casks` section (AGENTS.md notes it's deprecated since v2.16)
47. Fix the `archives.format_overrides` deprecation in `.goreleaser.yaml`
48. Add `release.yml` to the website's CI to deploy on tag pushes
49. Set up Firebase hosting for `dynamicmarkdown.lars.software` (DNS, SSL)
50. Add monitoring/uptime for the documentation site

---

## G) Top 2 Questions

### 1. Should the `.goreleaser.yaml` license be changed to proprietary/unfree, or should the LICENSE file be changed to MIT?

The `.goreleaser.yaml` says `license: MIT` in 5 places. The `LICENSE` file says proprietary. The `flake.nix` says `licenses.unfree`. One of them is wrong. **I cannot determine which is the intended license** — this is a business/legal decision. If the project should be open-source (MIT), the LICENSE file needs rewriting and the flake.nix needs updating. If it's genuinely proprietary, the goreleaser config is actively harming the project by publishing wrong metadata to Homebrew, Scoop, and Nix repositories.

### 2. What is the actual deployment plan for the website?

I configured the website for Firebase hosting at `dynamicmarkdown.lars.software` and updated the GitHub homepage URL to point there. But:

- The Firebase project/target `dynamicmarkdown` may not exist yet in the `lars-software` Firebase project
- The DNS record for `dynamicmarkdown.lars.software` may not be configured
- The `.firebaserc` assumes the same Firebase project structure as `go-atomic-write` and `gogenfilter`

Should I attempt a `firebase deploy`, or does the `lars-software` Firebase project + DNS need to be set up first by you?
