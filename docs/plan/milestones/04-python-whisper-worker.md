# Milestone 4 Prompt — Python Whisper GPU Worker

Read the project overview and ADR before changing the worker. Do not put conversation logic in Python.

## Objective

Implement an independently runnable Python worker that accepts an optimized WAV and creates a versioned, timestamped transcript using `faster-whisper` on CUDA.

## Worker Interface

Implement a command similar to:

```powershell
uv run echo-worker transcribe `
  --input C:\path\optimized.wav `
  --output C:\path\transcript.json `
  --model large-v3 `
  --device cuda `
  --compute-type float16 `
  --contract-version 1
```

Also implement:

```text
echo-worker version
echo-worker doctor
```

## Contract

- Emit newline-delimited JSON events to stdout.
- Emit human diagnostics to stderr.
- Never mix plain text into the stdout event stream.
- Write the final transcript to a temporary path and atomically replace the destination only after success.
- Use explicit error codes and nonzero exit statuses.
- Include a schema version in every result.

Example event categories:

```json
{"version":1,"type":"started","model":"large-v3"}
{"version":1,"type":"progress","stage":"loading_model"}
{"version":1,"type":"segment","sequence":1,"start":0.0,"end":4.2}
{"version":1,"type":"completed","segments":42,"duration_seconds":300.5}
```

The canonical transcript result must include worker/model metadata, detected language, source duration, and ordered segments with start, end, and normalized text.

## Inference Requirements

- Use `faster-whisper`.
- Default to CUDA and fail clearly when requested CUDA support is unavailable.
- Permit explicit model, language, device, and compute-type settings.
- Avoid hidden fallback from CUDA to CPU unless the user explicitly requests an auto mode.
- Record effective settings in the result.

## Testability

Normal CI must not download a large model or require a GPU. Introduce an inference adapter so tests can use a deterministic fake backend.

Test:

- Argument validation
- Event stream validity
- Transcript schema
- Atomic output behavior
- Backend failure mapping
- Invalid audio
- Unsupported contract version
- Segment ordering and normalization

Provide a separately documented manual GPU smoke test.

## Constraints

- No conversation lookup.
- No Markdown generation.
- No Go invocation code.
- No HTTP service.
- No diarization.

## Definition of Done

The fake backend passes in CI, and a documented Windows GPU command can transcribe a prepared WAV into contract-valid JSON with valid progress events.

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

- Milestone 3 is merged.
- An optimized WAV fixture exists.
- The worker contract version is documented.

## Spark Issue Contracts

### Issue 1: Implement worker CLI and settings model

**Acceptance criteria**

- [ ] `echo-worker version`, `doctor`, and `transcribe` exist.
- [ ] Arguments are validated with stable exit codes.
- [ ] Effective settings are explicit.
- [ ] CLI tests pass without GPU access.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Implement versioned event and transcript contracts

**Acceptance criteria**

- [ ] Stdout contains NDJSON events only.
- [ ] Human diagnostics go to stderr.
- [ ] Every event and transcript includes a schema version.
- [ ] Contract fixtures validate successfully.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Implement faster-whisper inference adapter

**Acceptance criteria**

- [ ] The production adapter uses `faster-whisper`.
- [ ] CUDA is the default requested device and unavailable CUDA fails clearly.
- [ ] No silent CPU fallback occurs unless explicitly selected.
- [ ] Model, language, device, and compute type are recorded.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Implement atomic output and structured errors

**Acceptance criteria**

- [ ] Transcript output is committed atomically only after success.
- [ ] Backend and input failures map to documented codes.
- [ ] Partial output never masquerades as complete.
- [ ] Failure tests pass.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 5: Add deterministic fake backend and GPU smoke test

**Acceptance criteria**

- [ ] CI uses a fake backend without model downloads.
- [ ] Segment ordering and text normalization are tested.
- [ ] A manual Windows GPU smoke-test procedure exists.
- [ ] `uv run pytest` passes.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
cd worker && uv sync
cd worker && uv run pytest
cd worker && uv run echo-worker version
cd worker && uv run echo-worker doctor
Run documented fake-backend transcription fixture
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
