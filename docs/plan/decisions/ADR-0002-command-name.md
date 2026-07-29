# ADR-0002: Ship the CLI as `ekko`

- Status: Accepted
- Scope: Echo CLI v0.1
- Supersedes: nothing. Narrows the command surface named in `project-overview.md`.

## Context

The planning documents invoke the CLI as bare `echo`:

```powershell
echo doctor
echo new "My Idea"
echo transcribe --model large-v3
```

`echo` is not an available command name on any platform Echo targets or is
developed on:

- **PowerShell** — `echo` is an alias for `Write-Output`. Command resolution
  order is alias, function, cmdlet, then external command, so the alias wins
  over an executable on `PATH`.
- **zsh and bash** — `echo` is a shell builtin. Builtins are resolved before
  `PATH` is consulted.

In every case `echo version` prints the string `version` and exits zero. The
failure is silent: a script cannot distinguish it from the CLI working
correctly, and neither can a user reading their terminal. Every Definition of
Done across the seven milestone documents is written in this form, so the
project's entire verification story was unrunnable as specified.

Discovered while reviewing the milestone documents before implementing
Milestone 1, and recorded here because `SPARK-WORKFLOW.md` requires an ADR
before an architectural or contract change.

## Decision

Ship the executable as `ekko` (`ekko.exe` on Windows).

"Echo" remains the product name in all prose. Only the command a user types
changes. The Python worker keeps its `echo-worker` entrypoint: it is always
invoked as `uv run echo-worker …`, never as a bare `echo`, so no collision
exists.

The command name lives in exactly one Go constant so a future rename is a
one-line edit rather than a sweep.

## Alternatives considered

**Keep `echo` and document `.\echo.exe`.** Rejected. It makes the documented
happy path a workaround, and a user who types the obvious thing gets silent
wrong behavior rather than an error. Shell aliases and functions can shadow the
builtin, but requiring every user to edit their shell profile before the tool
works is a worse first run than a different name.

**Keep `echo` and rely on `PATH` precedence.** Rejected as impossible: builtins
and aliases are resolved before `PATH` by design.

**Name it `echoctl`.** Rejected as a poor fit. The `-ctl` convention signals a
control plane for a running service; Echo is a local single-user tool.

**Name it `ec`.** Rejected. Two letters is collision-prone with local aliases
and conveys nothing.

## Consequences

### Positive

- The documented commands actually run the program on Windows, macOS, and Linux.
- Failures become real errors instead of a shell echoing its arguments back.
- Milestone Definitions of Done become executable as written.

### Negative

- The command no longer matches the product name, so documentation must state
  the relationship plainly.
- The planning documents needed a sweep to correct every invocation.
- Anyone who read the earlier drafts learned the wrong command.

## Guardrails

- No document or help text may instruct a reader to run bare `echo <subcommand>`.
- The command name is referenced through the single Go constant, never
  hardcoded in a second place.
- Any future rename updates this ADR rather than silently diverging from it.
