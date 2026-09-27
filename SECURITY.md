# Security Policy

## Supported Versions

Only the latest `master` branch and the most recent tagged release receive security fixes.

## Reporting a Vulnerability

**Do not open a public issue for security reports.**

Use GitHub's private vulnerability reporting:
<https://github.com/LarsArtmann/dynamic-markdown-site/security/advisories/new>

Alternatively, email <lars@larsartmann.com> (include "dynamic-markdown-site security" in the subject).

Please include:

- A description of the issue and its impact
- Step-by-step reproduction or proof of concept
- Affected versions or commit range
- Any known workarounds

## What to Expect

- Acknowledgement within 7 days
- A fix or mitigation timeline within 30 days for confirmed issues
- Credit in the release notes unless you prefer to remain anonymous

## Scope

In scope: the Go server (`cmd/`, `internal/`), its HTTP API, the container image, and the website build pipeline.

Out of scope: vulnerabilities in the deployed hosting infrastructure (Firebase/CDN configuration), denial-of-service via bandwidth, and social engineering.
