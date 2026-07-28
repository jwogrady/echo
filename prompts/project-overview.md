# Echo Project Overview

## Product

Echo is a voice-capture and transcription workspace for users who think best by talking.

A user creates a conversation, adds an outline and supporting resources, records or imports audio, transcribes it with Whisper, and receives a timestamped transcript connected to the original audio and supporting context.

## Product Evolution

Echo will be developed incrementally through three product stages.

### Stage 1: Windows CLI

The first version runs locally on Windows and accepts WAV files.

The user can:

- Create a conversation
- Add a WAV file
- Preserve the original audio
- Optimize audio for transcription
- Transcribe with Whisper using a local NVIDIA GPU
- View the transcript
- Export Markdown and JSON

### Stage 2: Local Application

The second version adds:

- Local API
- Qwik web interface
- Conversation dashboard
- File and link attachments
- Browser recording
- Manual pause and resume
- Automatic pause after inactivity
- Transcript editing

### Stage 3: SaaS

The final stage adds:

- Supabase authentication
- Multi-user data isolation
- Cloud storage
- Subscription billing
- Usage metering
- Hosted processing jobs
- GPU worker orchestration

## Initial Technical Stack

### CLI and Core Engine

- Python 3.12+
- uv
- Typer
- Rich
- Pydantic
- faster-whisper
- FFmpeg
- pytest

### Local Application

- FastAPI
- SQLite
- Qwik
- Local filesystem storage
- Background job runner

### SaaS

- Qwik
- Supabase Auth
- Supabase PostgreSQL
- Supabase Storage or S3-compatible object storage
- GPU transcription workers
- Durable job queue
- Stripe

## First Executable Target

```powershell
uv run echo transcribe .\recording.wav
```

The command must:

1. Validate the input WAV
2. Preserve the source file
3. Create a transcription-optimized WAV
4. Run Whisper on the local GPU
5. Save a timestamped JSON transcript
6. Save a readable Markdown transcript
7. Report useful progress and errors

## Core Domain Objects

### Conversation

A workspace containing recordings, resources, transcripts, and exports.

### Recording

An original audio source associated with a conversation.

### Optimized Audio

A derived transcription-ready copy of the original audio.

### Transcript

The complete machine-generated text for a recording or conversation.

### Transcript Segment

A timestamped section of speech.

### Resource

A note, file, image, media attachment, or website linked to a conversation.

### Transcription Job

A tracked processing operation with status, timing, model, and error information.

## Architectural Boundaries

The following modules should remain separate from the first milestone onward.

### Conversation Service

- Create conversations
- Load and update metadata
- Resolve conversation paths
- Track state

### Audio Service

- Validate audio
- Inspect metadata
- Preserve source files
- Run FFmpeg optimization

### Transcription Service

- Load Whisper
- Select device and compute type
- Transcribe audio
- Produce structured segments

### Export Service

- Export JSON
- Export Markdown
- Later export text and other formats

### Storage Service

- Resolve paths
- Copy files safely
- Write files atomically
- Prevent accidental overwrite

## Local Conversation Structure

```text
echo-data/
└── conversations/
    └── <conversation-id>/
        ├── conversation.json
        ├── outline.md
        ├── resources.json
        ├── audio/
        │   ├── source.wav
        │   └── optimized.wav
        ├── transcript/
        │   ├── transcript.json
        │   ├── transcript.md
        │   └── segments.json
        └── exports/
```

## Engineering Principles

1. Build the smallest complete vertical slice first.
2. Preserve original audio without modification.
3. Keep domain logic independent from interfaces.
4. Make every long-running operation observable.
5. Make failed jobs resumable.
6. Use structured data as the source of truth.
7. Treat Markdown as an export, not the canonical transcript.
8. Keep local-first workflows working after the SaaS exists.
9. Avoid premature distributed-system complexity.
10. Add infrastructure only when a product requirement demands it.

## Explicit Non-Goals for the First MVP

Do not add:

- Supabase
- Qwik
- FastAPI
- Docker
- Redis
- Stripe
- User accounts
- Speaker diarization
- Live transcription
- MP3 or video support
- Cloud storage
- Team collaboration

The first MVP is successful when one WAV file can reliably become a structured transcript on the local Windows machine.
