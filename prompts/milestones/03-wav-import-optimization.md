# Milestone 3 Prompt — WAV Import and Optimization

Read the project overview, ADR, and completed conversation implementation before changing code.

## Objective

Implement a safe WAV ingestion pipeline in Go. Preserve the source exactly, inspect it, and create a transcription-ready derivative through FFmpeg.

## Command

Implement:

```text
echo add <wav-path>
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
echo add .\testdata\sample.wav
echo status
```

preserves a checksum-verified source, creates a validated optimized WAV, persists metadata, and never changes the source file.
