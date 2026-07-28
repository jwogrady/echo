# Milestone 2 Prompt — WAV Audio Pipeline

Continue the Echo repository from Milestone 1.

## Objective

Implement a safe WAV ingestion and optimization pipeline using FFmpeg.

## Required Behavior

Add support for:

```powershell
uv run echo audio inspect .\recording.wav
uv run echo audio optimize .\recording.wav
```

The inspect command should report:

- File path
- File size
- Duration
- Sample rate
- Channel count
- Bit depth or codec information
- Whether the file satisfies Echo's transcription-ready format

The optimize command should:

1. Validate that the source exists and is a WAV file
2. Preserve the original file without modification
3. Create a derived optimized WAV
4. Convert to mono
5. Resample to 16 kHz
6. Use PCM signed 16-bit encoding
7. Avoid overwriting existing output unless explicitly requested
8. Verify the generated file after conversion

Use FFmpeg and FFprobe through subprocess execution with safe argument handling.

## Engineering Requirements

Create or complete:

- `AudioService`
- WAV validation
- FFmpeg discovery
- FFprobe metadata extraction
- Typed `AudioMetadata`
- Safe destination naming
- Clear exception types
- Unit tests
- Integration tests that skip cleanly when FFmpeg is unavailable

Do not shell-concatenate untrusted file paths.

Keep source preservation and optimization separate.

## CLI UX

Show progress and final paths using Rich.

Examples:

```text
Source validated
Audio metadata read
Optimized WAV created
Output verified
```

## Definition of Done

A real WAV file can be inspected and converted into a transcription-ready WAV without modifying the source.

Do not add Whisper yet.
