# Milestone 5 Prompt — Go and Worker Integration

Read the project overview, ADR, and completed worker contract before changing code.

## Objective

Implement `ekko transcribe` in Go using the Python worker subprocess contract. Add durable job state, observable progress, compatibility checks, cancellation, and safe recovery.

## Command

Implement:

```text
ekko transcribe [--model <name>] [--language <code|auto>] [--device cuda] [--compute-type <type>]
```

Operate on the active or explicitly selected conversation.

## Preconditions

Before starting:

- Conversation exists and is supported.
- A validated optimized WAV exists.
- No incompatible active job exists.
- uv and worker are discoverable.
- Worker contract version is compatible.
- Output paths are writable.

## Job State Machine

Implement explicit states such as:

```text
queued
starting
running
completed
failed
cancelled
```

Persist job state atomically before and after important transitions. Store worker settings, timestamps, process result, and structured error information.

## Subprocess Integration

- Build argument arrays safely.
- Capture stdout and stderr separately.
- Parse stdout as strict newline-delimited JSON events.
- Display useful terminal progress without making terminal output canonical state.
- Treat malformed events as protocol errors.
- Preserve useful stderr diagnostics in job records with reasonable size limits.
- On Ctrl+C, terminate the worker cleanly where possible, mark the job cancelled, and preserve partial diagnostics.
- Validate transcript JSON before moving the job to completed.

## Retry Behavior

A failed or cancelled job can be retried without deleting the source or optimized audio. Never overwrite a completed transcript unless an explicit overwrite/retranscribe option is used.

## Testing

Use a fake worker executable or test process to cover:

- Successful event stream
- Worker not found
- Incompatible contract
- Malformed event
- Nonzero worker exit
- Valid output missing
- Invalid transcript output
- Ctrl+C cancellation
- Retry after failure
- Existing completed transcript

GPU is not required for Go integration tests.

## Definition of Done

```powershell
ekko transcribe --model large-v3
ekko status
```

runs the worker, shows progress, writes durable job state, validates the transcript, and leaves the conversation in a correct state after success, failure, or interruption.

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

- Milestone 4 is merged.
- Worker contract fixtures are stable.
- A conversation with optimized audio exists.

## Spark Issue Contracts

### Issue 1: Discover worker and verify compatibility

**Acceptance criteria**

- [ ] The Go CLI locates uv and the worker deterministically.
- [ ] CLI and worker contract versions are checked before execution.
- [ ] Incompatibility produces actionable remediation.
- [ ] Discovery tests cover missing and mismatched workers.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Implement durable transcription job state

**Acceptance criteria**

- [ ] Job states and allowed transitions are explicit.
- [ ] State is persisted atomically around important transitions.
- [ ] Settings, timestamps, exit details, and errors are recorded.
- [ ] Invalid transitions are rejected.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Execute worker and parse progress events

**Acceptance criteria**

- [ ] Arguments are passed without shell interpolation.
- [ ] Stdout and stderr are captured separately.
- [ ] NDJSON events are strictly parsed.
- [ ] Terminal progress is informative but not canonical state.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Validate and commit transcript results

**Acceptance criteria**

- [ ] The final transcript is schema-validated before completion.
- [ ] Missing or malformed output fails the job.
- [ ] Completed transcripts are protected from accidental overwrite.
- [ ] Success integration tests pass with a fake worker.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 5: Handle cancellation, retry, and recovery

**Acceptance criteria**

- [ ] Ctrl+C attempts graceful termination and marks the job cancelled.
- [ ] Retries preserve source and optimized audio.
- [ ] Worker failures preserve bounded diagnostics.
- [ ] Cancellation and retry tests pass.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
go test ./...
go vet ./...
ekko transcribe --model large-v3
ekko status
Run fake-worker success, failure, malformed-event, and cancellation tests
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
