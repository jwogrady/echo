# Milestone 3 Prompt — GPU Whisper Transcription

Continue the Echo repository from Milestones 1 and 2.

## Objective

Implement local transcription with `faster-whisper` using the NVIDIA GPU when available.

## CLI Target

```powershell
uv run echo transcribe .\recording.wav
```

## Required Pipeline

1. Validate the input WAV
2. Inspect the audio
3. Create or reuse a transcription-ready optimized WAV
4. Detect GPU availability
5. Load the configured Whisper model
6. Transcribe the file
7. Save timestamped segments as JSON
8. Save the complete transcript as JSON
9. Save a readable Markdown transcript
10. Print a completion summary

## Configuration

Support configuration for:

- Model name
- Device: auto, cuda, cpu
- Compute type
- Language
- Beam size
- Output directory

Use sensible defaults.

Recommended development model:

```text
small
```

Recommended quality model after validation:

```text
large-v3
```

## Transcript Schema

Each segment should contain at least:

- Sequence number
- Start time in seconds
- End time in seconds
- Text
- Optional confidence-related metadata when available

The transcript should contain:

- Source audio metadata
- Optimized audio path
- Model
- Device
- Compute type
- Detected language
- Duration
- Processing time
- Segments
- Full text

## Reliability

- Fail clearly when CUDA is requested but unavailable
- Allow explicit CPU fallback
- Do not silently switch devices
- Preserve partial diagnostic information on failure
- Do not write a completed transcript until processing succeeds

## Testing

Mock model loading for unit tests.

Add an optional integration test marker for real local transcription.

## Definition of Done

A WAV file can be transcribed locally on the RTX-class GPU and produces valid `transcript.json`, `segments.json`, and `transcript.md` files.
