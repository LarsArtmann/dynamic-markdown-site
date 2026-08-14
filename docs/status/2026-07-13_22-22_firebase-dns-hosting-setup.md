# Status Report: Firebase Hosting + DNS Setup for Dynamic Markdown Site

**Date:** 2026-07-13 22:22
**Session scope:** Configure `/home/lars/projects/domains/` DNS and Firebase hosting for `dynamicmarkdown.lars.software`

---

## A) FULLY DONE

### 1. Firebase Hosting Site Created

- Site `dynamicmarkdown` created in project `lars-software`
- Default URL live: `https://dynamicmarkdown.web.app`
- 63 files deployed (landing page + 10 doc pages + 404 + CSS/JS/fonts/sitemap/pagefind)
- Verified live by fetching the installation page — all content rendering correctly

### 2. Custom Domain Added in Firebase

- `dynamicmarkdown.lars.software` registered as a custom domain on the `dynamicmarkdown` site
- Domain status: `DOMAIN_ACTIVE`
- SSL cert provisioning: `CERT_PENDING` (will activate once DNS propagates)
- DNS verification: `DNS_MISSING` (waiting for Terraform apply)

### 3. Terraform DNS Records Written

Added two records to `/home/lars/projects/domains/lars.software.tf`:

- **CNAME** `dynamicmarkdown` -> `dynamicmarkdown.web.app.` — maps the subdomain to Firebase hosting
- **TXT** `_acme-challenge.dynamicmarkdown` -> `hlsIzj2ITXCZCBuCWJgskQL2mmvNKis54OYM-KLxBWs` — Firebase SSL cert verification (ACME DNS challenge)
- Terraform `fmt` and `validate` both pass

---

## B) PARTIALLY DONE

### Terraform apply NOT run

- The DNS records are written in Terraform but **not applied** to Namecheap
- Namecheap API rejected the apply because the current public IP (`89.65.239.240`) is **not whitelisted** in the Namecheap API access settings
- Error: `API Key is invalid or API access has not been enabled ... the public IP this provider calls from is whitelisted`
- The DNS change must be applied from a whitelisted machine

### Custom domain SSL cert pending

- Firebase shows `CERT_PENDING` and `DNS_MISSING`
- SSL will auto-provision once the CNAME and TXT records propagate after Terraform apply
- Until then, `https://dynamicmarkdown.lars.software` will not resolve

---

## C) NOT STARTED

- Committing changes in either repo (`dynamic-markdown-site` or `domains`)
- Adding a GitHub Actions workflow for the website (auto-deploy on push)
- Setting up the `firebase deploy` as part of CI/CD

---

## D) TOTALLY FUCKED UP

### 1. Left `firebase-tools` as a dependency in `website/package.json`

I ran `bun add -d firebase-tools` as a debugging step when `bunx firebase-tools` failed with a native module error (`re2.node: undefined symbol`). This added `"firebase-tools": "^15.23.0"` to `devDependencies` in `website/package.json`. This is **wrong** — the reference websites (`go-atomic-write`, `gogenfilter`) do NOT have `firebase-tools` in `package.json`. The deploy should use `bunx firebase-tools` or the Nix devShell's `firebase-tools` package. I need to remove this dependency.

### 2. Added TODO review comments to the domains repo that I didn't write

The `git diff` of `lars.software.tf` shows `# TODO(review):` comments on pre-existing records (bunq-splitter, sendgrid DKIM, em7370) and changes to other `.tf` files (`artmann-holding.com.tf`, `extract-metadata.tech.tf`, `issue-shield.com.tf`, `larsartmann.cloud.tf`, `locals.tf`). **These changes were NOT made by me** — they were already in the working tree when I started. The domains repo had uncommitted changes from a prior session. I should have checked `git status` before editing and noted these were pre-existing. I did not author them, but my Terraform apply (when run) would include them.

### 3. Did not check git status before editing the domains repo

The domains repo had **14 pre-existing uncommitted files**. I added my changes on top without verifying the working tree was clean. If someone applies this Terraform, they'll also apply all the pre-existing changes — some of which may be intentional, some may not.

---

## E) WHAT WE SHOULD IMPROVE

1. **Remove `firebase-tools` from `website/package.json`** — it was a debugging artifact. Deploy should use `bunx` or the Nix shell.
2. **Apply the Terraform DNS change** — the site is live at `.web.app` but `dynamicmarkdown.lars.software` won't work until DNS propagates. Must be done from a whitelisted IP.
3. **Set up CI/CD for the website** — add a GitHub Action that runs `astro check` + `astro build` + `firebase deploy` on push to `master` for the `website/` directory.
4. **Clean the domains repo working tree** — 14 files have pre-existing uncommitted changes. These should be reviewed, committed, or reverted before my DNS change is applied.
5. **Add a deploy script** — the website `flake.nix` already has a `deploy` app (`pnpm run build && firebase deploy --only hosting`). Document this in the README or CONTRIBUTING.

---

## F) Up to 50 Things We Should Get Done Next

### Immediate fixes

1. Remove `firebase-tools` from `website/package.json`
2. Run `bun install` to regenerate the lockfile without firebase-tools
3. Rebuild the website to confirm it still builds clean
4. Review the 14 pre-existing uncommitted files in the domains repo
5. Commit the DNS changes in the domains repo (just the `lars.software.tf` diff)
6. Apply the Terraform DNS change from a whitelisted IP
7. Verify DNS propagation with `dig dynamicmarkdown.lars.software`
8. Verify SSL cert provisioning in Firebase console
9. Verify `https://dynamicmarkdown.lars.software` loads correctly

### CI/CD

10. Add a GitHub Actions workflow for the website (build + deploy on push)
11. Add Firebase token as a GitHub secret (`FIREBASE_TOKEN`)
12. Add `astro check` as a required CI step
13. Consider auto-deploying on tags only (vs every push to master)

### DNS / Domains repo cleanup

14. Investigate the pre-existing TODO review comments in `lars.software.tf`
15. Fix the `bunq-splitter.me` hostname (likely should be `bunq-splitter`)
16. Fix the duplicate sendgrid DKIM records (`s1._domainkey.lars.software` etc.)
17. Fix the duplicate `em7370.lars.software` record
18. Fix `issue-shield.com.tf` CAA security_email (points to artmann.tech)
19. Fix `artmann-holding.com.tf` module naming (generic `email` vs convention)
20. Fix `locals.tf` baerenstein.ch split brain (listed but config disabled)
21. Commit or revert all pre-existing domains repo changes

### Website polish

22. Add OG image for social sharing
23. Add GitHub social preview image
24. Add screenshots of the actual dynamic-markdown-site UI
25. Add D2 architecture diagram to docs (dogfooding)
26. Add performance/benchmarks guide page
27. Add architecture guide page
28. Add "Edit on GitHub" links to doc pages
29. Test all internal links with a link checker
30. Add a copy-to-clipboard button check in Starlight docs
31. Add print styles
32. Test mobile responsiveness

### README / GitHub

33. Add badges (CI, Go version, Docker pulls)
34. Add a "Quick Start" one-liner at the very top
35. Add link to the live website from README
36. Add a comparison section vs Hugo/MkDocs/BookStack
37. Add screenshots/GIF to README
38. Fix `.goreleaser.yaml` license mismatch (MIT vs proprietary)
39. Add issue templates
40. Add SECURITY.md
41. Add CODEOWNERS
42. Enable GitHub Discussions
43. Create a GitHub release with release notes
44. Add branch protection for master

### Commit the original session work

45. Commit the README.md rewrite in `dynamic-markdown-site`
46. Commit the entire `website/` directory in `dynamic-markdown-site`
47. Verify `git status` is clean after committing
48. Push to remote (if instructed)
49. Squash or organize commits logically
50. Write a commit message that explains the README fixes (Gin -> net/http, etc.)

---

## G) Top 2 Questions

### 1. Should I commit the pre-existing uncommitted changes in the domains repo, or are those yours to review?

The domains repo (`/home/lars/projects/domains/`) had 14 files with uncommitted changes before I touched it — including TODO review comments, formatting changes, and module updates. These are NOT my changes. If I commit `lars.software.tf`, those 14 files will either need to be committed separately, reverted, or explicitly excluded. **Should I stash/commit only my `lars.software.tf` changes, or do you want to review the pre-existing changes first?**

### 2. From which machine should the Terraform apply happen?

The current public IP (`89.65.239.240`) is not whitelisted in Namecheap's API access settings. The Terraform apply cannot run from here. **Do you have a whitelisted machine to run `terraform apply` from, or should I add this IP to the Namecheap whitelist first?** Without the apply, `dynamicmarkdown.lars.software` will not resolve.
