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
