# Milestone 5 Prompt — Go and Worker Integration

Read the project overview, ADR, and completed worker contract before changing code.

## Objective

Implement `echo transcribe` in Go using the Python worker subprocess contract. Add durable job state, observable progress, compatibility checks, cancellation, and safe recovery.

## Command

Implement:

```text
echo transcribe [--model <name>] [--language <code|auto>] [--device cuda] [--compute-type <type>]
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
echo transcribe --model large-v3
echo status
```

runs the worker, shows progress, writes durable job state, validates the transcript, and leaves the conversation in a correct state after success, failure, or interruption.
