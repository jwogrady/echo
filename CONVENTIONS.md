# Project Conventions

> **This document is yours.** Spark seeded it once from shipped defaults and
> will never overwrite it — edit it freely so it describes how this repository
> actually works. It is the readable, reviewable contract collaborators read
> before they open a pull request.

**How to read this file**

- **Default guidance** — shipped starting points. Keep them or change them.
- **`[ADAPT]`** — a project decision: replace the placeholder with what is
  true for this repo.
- **`TODO(decision)`** — a human decision Spark cannot make for you. Resolve
  it, then delete the marker.

Lines that end with a `<!-- spark:pref key=value -->` comment are
**machine-backed**: the same fact is resolved from `.spark/preferences.json`
(shipped defaults, then operator overrides, then this repo's committed project
facts). Editing only the prose does **not** change what Spark's automation
does — change the matching preference too, or the two will drift. Prose without
a marker is guidance, never silently configuration. The prose/config boundary
is documented in the Spark reference `project-standards.md`.

---

## Branching

- Branch model `github-flow`: short-lived feature branches off the trunk;
  never commit to the trunk directly. <!-- spark:pref branch.model=github-flow -->
- Name branches by type: `feat/…`, `fix/…`, `docs/…`, `chore/…`.

## Commits

- Commit convention: `conventional-commits`. <!-- spark:pref commit.convention=conventional-commits -->
- Subject line in the imperative mood, under `72` characters,
  no trailing period. <!-- spark:pref commit.subject-max=72 -->
- The body explains *why*, not *what*. One logical change per commit.

## Pull requests & review

- Open a pull request for every change, even small ones. One concern per PR.
- `[ADAPT]` What "approved and mergeable" requires for this repo (passing
  checks, review count, up-to-date branch).
- `TODO(decision)`: required reviewers and any review SLA.

## Issue tracking

- Track work across these categories: `feature bug documentation chore tech-debt research infrastructure`. <!-- spark:pref issue.taxonomy=feature bug documentation chore tech-debt research infrastructure -->
- `[ADAPT]` Any project-specific labels or milestone rules layered on top.

## Collaboration boundaries

- Permission trust tier `delivery` — the baseline of what
  automation may do in this repo without a per-command prompt. <!-- spark:pref permissions.preset=delivery -->
- Do not close or comment on issues/PRs, and do not cut releases or tags,
  without explicit human instruction.
- `[ADAPT]` Team boundaries: who owns merges, deploy windows, and who to ask
  when a decision is unclear.

## Documentation

- Documentation describes reality. Update it in the same change that changes
  behavior.
- `[ADAPT]` Where this project's docs live and who keeps them current.
