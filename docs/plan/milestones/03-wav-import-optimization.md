# Milestone 3 Prompt — WAV Import and Optimization

Read the project overview, ADR, and completed conversation implementation before changing code.

## Objective

Implement a safe WAV ingestion pipeline in Go. Preserve the source exactly, inspect it, and create a transcription-ready derivative through FFmpeg.

## Command

Implement:

```text
ekko add <wav-path>
```

The command operates on the active conversation or an explicitly selected conversation.

## Required Pipeline

1. Resolve and validate the source path.
2. Require WAV input for v0.1.
3. Inspect audio metadata through ffprobe using machine-readable JSON output.
4. Compute SHA-256 before import.
5. Copy the source into the conversation without modifying it.
6. Recompute SHA-256 and verify the copy.
7. Create `optimized.wav` with FFmpeg as mono, 16 kHz, signed 16-bit PCM.
8. Inspect and validate the optimized output.
9. Persist recording metadata atomically.
10. Advance conversation state only after the complete pipeline succeeds.

## Safety and Idempotency

- Re-running with the identical source should produce an informative no-op or explicit replace workflow, not duplicate unknown state.
- A different source must not silently replace an existing recording.
- Partial files must be cleaned up or clearly quarantined.
- The original filename and external source path may be recorded, but the imported copy is authoritative.
- Never run shell-concatenated commands. Pass FFmpeg arguments safely through process APIs.

## Progress and Errors

Display meaningful stages:

```text
Inspecting source
Hashing source
Copying source
Verifying copy
Optimizing audio
Validating optimized audio
Saving metadata
```

Errors should include the failed stage and practical remediation.

## Tests

Use small committed fixtures or generated test WAVs. Mock process execution where appropriate and include integration tests when FFmpeg is available.

Test:

- Valid WAV
- Invalid extension
- Missing file
- Corrupt WAV
- Stereo/high-rate conversion
- Checksum mismatch simulation
- FFmpeg failure
- Retry after partial failure
- Existing recording collision

## Definition of Done

```powershell
ekko add .\testdata\sample.wav
ekko status
```

preserves a checksum-verified source, creates a validated optimized WAV, persists metadata, and never changes the source file.

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

- Milestone 2 is merged.
- A conversation can be created and selected.
- FFmpeg and ffprobe are available or diagnosed by `ekko doctor`.

## Spark Issue Contracts

### Issue 1: Validate WAV inputs

**Acceptance criteria**

- [ ] Nonexistent and non-WAV inputs are rejected before mutation.
- [ ] The container and codec are inspected rather than trusting extensions.
- [ ] Validation failures return stable error codes.
- [ ] Valid, malformed, and mislabeled fixtures are tested.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Import and preserve source audio

**Acceptance criteria**

- [ ] The source WAV is copied without modification.
- [ ] SHA-256 and source metadata are recorded.
- [ ] Filename collisions follow an explicit policy.
- [ ] Failed imports leave no committed partial state.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Inspect audio metadata with ffprobe

**Acceptance criteria**

- [ ] Duration, channels, sample rate, codec, and format are captured.
- [ ] ffprobe output parsing is strict and tested.
- [ ] Malformed tool output is reported as a structured error.
- [ ] Metadata writes are atomic.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Create optimized derivative with FFmpeg

**Acceptance criteria**

- [ ] A mono 16 kHz PCM WAV derivative is produced.
- [ ] The original remains byte-identical.
- [ ] FFmpeg arguments are safely constructed.
- [ ] Conversation state advances only after validated output exists.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 5: Guarantee idempotency and cleanup

**Acceptance criteria**

- [ ] Repeated add operations do not corrupt data.
- [ ] Temporary files are removed after failure.
- [ ] Explicit overwrite/replacement behavior is documented.
- [ ] Failure-path integration tests pass.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
go test ./...
go vet ./...
ekko new "Audio Test"
ekko add .\fixtures\speech.wav
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
