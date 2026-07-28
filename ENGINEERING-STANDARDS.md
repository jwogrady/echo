# Engineering Standards

> **This document is yours.** Spark seeded it once from shipped defaults and
> will never overwrite it — edit it freely so it describes this project's real
> engineering choices. It is the reviewable contract for how this repository is
> built and shipped.

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

## Stack & tooling

- Default stack: `python-uv` — the runtime, dependency, and
  project-management toolchain. <!-- spark:pref stack.default=python-uv -->
- Frontend stack: `typescript-bun`. <!-- spark:pref stack.frontend=typescript-bun -->
- Infrastructure: `opentofu`. <!-- spark:pref stack.infra=opentofu -->
- `[ADAPT]` If this project's stack differs, commit the exception to
  `.spark/preferences.json` and record it under *Deviations* below.

## Quality gates

- Keep the repository buildable; validate changes continuously.
- `[ADAPT]` The formatter, linter, type checker, and test runner for this
  stack.
- `TODO(decision)`: the exact commands CI runs as the merge gate.

## Dependencies

- Minimize them. Prefer the standard library. Add only for meaningful value;
  remove unused dependencies promptly.

## CI

- CI provider: `github-actions` — validation on every push. <!-- spark:pref ci.provider=github-actions -->

## Release posture

- Release mechanism `release-please`: it owns versioning, the changelog,
  tags, and GitHub Releases; `ship` does the commit and PR and defers the
  release to it. <!-- spark:pref release.mechanism=release-please -->

## Security & configuration

- Never commit secrets. Configuration lives outside source control, via
  environment variables.
- Principle of least privilege. Document required config; keep local dev
  simple.

## Deviations from Spark defaults

- Record every deliberate deviation from the shipped defaults here, and mirror
  the machine fact in `.spark/preferences.json` so the prose and the
  configuration never drift.
- `TODO(decision)`: list this project's deviations, or state "none".
