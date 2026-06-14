# 1. Record architecture decisions

Date: 2026-06-14

## Status

Accepted.

## Context

The project accumulated many "why did we do it this way" questions that
were answered only in old Slack threads, commit messages, or the
heads of individual contributors. New contributors could not tell, on
arrival, whether a given pattern was a deliberate decision or an
accident of history.

## Decision

We adopt **Architecture Decision Records (ADRs)** as the canonical
record of non-obvious technical choices. Each ADR:

- Is a short Markdown file in `docs/adr/`
- Has a unique four-digit number prefix (`0001-...`)
- Captures **Context**, **Decision**, and **Consequences**
- Is **immutable** once accepted; superseded decisions are kept and
  pointed at by a new ADR
- Index lives in [`docs/adr/README.md`](./README.md)

The format follows the [MADR](https://adr.github.io/madr/) template.

## Consequences

Positive:

- New contributors can find rationale without spelunking Git history
- The set of accepted decisions becomes a living design document
- Reviewers can point at ADRs to settle recurring design debates

Negative:

- Writing an ADR is overhead. We will keep them light and only
  document decisions that are non-obvious or expensive to change.
- Discipline required to write them; we add a PR-template reminder.

## Alternatives considered

- **Wiki pages** — easy to author, but no review trail and easy to
  silently drift out of sync with code.
- **Comments in code** — discoverability is poor and they bloat the
  source.
- **Commit messages** — they are append-only, hard to find, and often
  describe the _change_ not the _decision_.
