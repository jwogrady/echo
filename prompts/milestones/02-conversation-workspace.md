# Milestone 2 Prompt — Conversation Workspace

Read the project overview, ADR, and completed Milestone 1 implementation before changing code.

## Objective

Implement durable local conversation workspaces managed entirely by the Go CLI.

## Commands

Implement:

```text
echo new <title>
echo list
echo use <conversation-id-or-prefix>
echo status
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
echo new "Product Strategy"
echo list
echo use <id-prefix>
echo status
```

creates and selects a valid portable conversation workspace and passes all Go tests.
