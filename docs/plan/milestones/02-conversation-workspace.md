# Milestone 2 Prompt — Conversation Workspace

Read the project overview, ADR, and completed Milestone 1 implementation before changing code.

## Objective

Implement durable local conversation workspaces managed entirely by the Go CLI.

## Commands

Implement:

```text
ekko new <title>
ekko list
ekko use <conversation-id-or-prefix>
ekko status
```

Support a global `--conversation` option where appropriate so automation does not depend on active state.

## Requirements

- Generate stable, collision-resistant conversation IDs.
- Create the documented directory tree.
- Store versioned `conversation.json` metadata.
- Store an empty `outline.md` and `resources.json` for future compatibility without implementing resource features.
- Maintain active-conversation state outside conversation directories.
- Accept unique ID prefixes but reject ambiguous matches.
- List conversations in deterministic order with title, ID, status, and update time.
- Write metadata atomically using temporary files and rename/replace semantics suitable for Windows.
- Detect malformed or unsupported metadata and report it without destroying data.
- Never silently overwrite an existing conversation.

## State Model

Define a minimal conversation status model that can support later milestones, such as:

```text
created
recording_added
audio_ready
transcribing
transcribed
failed
```

Do not fake transitions that are not yet implemented.

## Tests

Include:

- Creation and serialization
- Duplicate titles with distinct IDs
- Atomic update behavior
- Active selection
- Ambiguous prefix handling
- Missing and malformed metadata
- Deterministic listing
- Data-root override

## Constraints

- No WAV import.
- No worker invocation.
- No database.
- Filesystem JSON is the source of truth.

## Definition of Done

```powershell
ekko new "Product Strategy"
ekko list
ekko use <id-prefix>
ekko status
```

creates and selects a valid portable conversation workspace and passes all Go tests.

## Spark Planning Instructions

Use `/spark:plan` to convert this milestone into the issue contracts below. Create one GitHub issue per contract unless repository evidence proves a smaller split is necessary. Do not combine issues merely to reduce issue count.

For every issue:

- Keep one observable capability per issue.
- Use one focused branch and one pull request.
- Copy the acceptance criteria into the GitHub issue as checkboxes.
- Add implementation notes only when they are evidence-backed and necessary.
- Do not pull work forward from later milestones.
- Do not perform unrelated cleanup, renaming, or dependency upgrades.
- Update documentation only when behavior or a contract changed.

## Preconditions

- Milestone 1 is merged.
- `ekko version` and `ekko doctor` run.
- The working tree is clean.

## Spark Issue Contracts

### Issue 1: Define conversation schema and storage layout

**Acceptance criteria**

- [ ] A versioned conversation schema is documented and implemented.
- [ ] The filesystem layout matches the project overview.
- [ ] Unsupported schema versions fail safely.
- [ ] Serialization round-trip tests pass.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Implement conversation create, list, and get operations

**Acceptance criteria**

- [ ] `ekko new <title>` creates a unique workspace.
- [ ] `ekko list` is deterministic and displays required fields.
- [ ] Duplicate titles do not collide.
- [ ] Existing conversations are never silently overwritten.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Implement active conversation selection

**Acceptance criteria**

- [ ] `ekko use <id-or-prefix>` persists selection outside conversation folders.
- [ ] Unique prefixes work and ambiguous prefixes fail clearly.
- [ ] A global `--conversation` override works where required.
- [ ] Selection behavior is tested.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Implement status and corruption handling

**Acceptance criteria**

- [ ] `ekko status` reports the active conversation and true current state.
- [ ] Malformed metadata is reported without mutation or deletion.
- [ ] Atomic writes are used for metadata updates.
- [ ] Recovery guidance is actionable.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
go test ./...
go vet ./...
ekko new "Product Strategy"
ekko list
ekko use <id-prefix>
ekko status
```

## Ship Gate

Do not run `/spark:ship` until:

- Every acceptance criterion for the current issue is satisfied.
- Required automated checks pass.
- The diff contains no work from a later milestone.
- User-facing behavior and documentation agree.
- Generated files, secrets, recordings, transcripts, models, and local environments are excluded.
- The pull request links the GitHub issue and states how the behavior was verified.

## Stop Rule

After shipping the current issue, stop. Resume with the next approved GitHub issue through `/spark:codify`; do not continue implementing from the milestone prompt on the same branch.
