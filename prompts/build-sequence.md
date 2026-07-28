# Echo Build Sequence

## Milestone 1 — Repository Foundation

Create the Python project, package layout, configuration, typed models, logging, and CLI shell.

## Milestone 2 — WAV Inspection and Optimization

Validate WAV files, inspect audio metadata, preserve originals, and create FFmpeg-optimized copies.

## Milestone 3 — GPU Whisper Transcription

Run faster-whisper locally on the NVIDIA GPU and produce timestamped transcript segments.

## Milestone 4 — Conversation Workspace

Add persistent conversation folders, metadata, audio imports, and active-conversation commands.

## Milestone 5 — Transcript Export and Review

Add Markdown and JSON export, terminal transcript viewing, timestamps, and stable schemas.

## Milestone 6 — Reliability and Testing

Add retries, atomic writes, resumable jobs, validation, test fixtures, and clear operational logging.

## Milestone 7 — Local FastAPI Service

Expose the existing core through a local HTTP API without duplicating domain logic.

## Milestone 8 — Qwik Local Application

Build the conversation dashboard, WAV upload, job status, transcript viewer, and audio playback.

## Milestone 9 — Browser Recording

Add recording, pause, resume, segmented capture, upload, and inactivity-based automatic pause.

## Milestone 10 — Supabase SaaS Foundation

Add authentication, tenant-aware data, storage, subscriptions, usage metering, and remote GPU jobs.

## Product Gates

Do not begin Milestone 7 until Milestones 1–6 work reliably from the CLI.

Do not begin Milestone 10 until the local Qwik application has a proven user workflow.
