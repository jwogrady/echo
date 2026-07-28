# Echo CLI — Project Overview

## Product

Echo CLI is a local Windows transcription tool for people who think by talking. It accepts WAV recordings, preserves the original audio, creates a transcription-ready derivative, runs Whisper on the user's NVIDIA GPU, and produces structured JSON plus readable Markdown.

This repository delivers only the CLI product. A graphical application and hosted SaaS may be built later, but they are explicitly outside this plan.

## First Complete User Journey

```powershell
echo doctor
echo new "Product Strategy"
echo add .\recordings\idea.wav
echo transcribe
echo show
echo export markdown
```

The workflow must create a durable conversation workspace containing the original WAV, optimized WAV, job metadata, timestamped transcript, and export.

## Technology Stack

### Go CLI

Use a current stable Go release and conventional Go module layout.

Go owns:

- Command parsing and terminal UX
- Conversation creation and selection
- Configuration and path resolution
- WAV validation and metadata inspection
- Original-file preservation
- FFmpeg process orchestration
- Worker installation checks and invocation
- Job state and recovery
- Transcript loading and validation
- Markdown and JSON exports
- Logging, progress, and error presentation

Suggested libraries may include Cobra for commands and a lightweight structured logging package, but prefer the standard library where it keeps the system simpler.

### Python GPU Worker

Use:

- Python 3.12+
- uv
- faster-whisper
- CUDA
- cuDNN

Python owns only machine-learning inference:

- Load the requested Whisper model
- Select CUDA or report an actionable failure
- Transcribe an already optimized WAV
- Emit timestamped transcript segments
- Emit machine-readable progress events
- Write a versioned transcript result

The worker must not manage conversations, exports, or application configuration beyond its own inference options.

### External Runtime

FFmpeg is an explicit prerequisite for the first release. Echo detects it, reports its version, and invokes it without modifying the source WAV.

## Process Boundary

The first integration is a subprocess boundary:

```text
Go CLI
  -> invokes uv run echo-worker transcribe
  -> passes explicit input/output arguments
  -> reads newline-delimited JSON progress events from stdout
  -> reads the completed transcript JSON from disk
```

Human-readable diagnostics belong on stderr. Machine-readable status events belong on stdout. The contract must be versioned so a future HTTP or gRPC transport can replace the subprocess without changing domain models.

## Suggested Repository Structure

```text
echo/
├── cmd/
│   └── echo/
│       └── main.go
├── internal/
│   ├── app/
│   ├── audio/
│   ├── config/
│   ├── conversation/
│   ├── export/
│   ├── jobs/
│   ├── storage/
│   ├── transcript/
│   └── worker/
├── worker/
│   ├── pyproject.toml
│   ├── uv.lock
│   ├── src/
│   │   └── echo_worker/
│   └── tests/
├── docs/
│   ├── architecture/
│   └── usage/
├── testdata/
├── go.mod
├── go.sum
└── README.md
```

## Local Data Structure

```text
<ECHO_DATA_DIR>/
└── conversations/
    └── <conversation-id>/
        ├── conversation.json
        ├── outline.md
        ├── resources.json
        ├── audio/
        │   ├── source.wav
        │   └── optimized.wav
        ├── jobs/
        │   └── <job-id>.json
        ├── transcript/
        │   ├── transcript.json
        │   └── segments.json
        └── exports/
            └── transcript.md
```

The default Windows data root should follow Windows conventions, such as `%LOCALAPPDATA%\Echo`, and support an `ECHO_DATA_DIR` override.

## Canonical Domain Objects

### Conversation

- ID
- Title
- Slug
- Created and updated timestamps
- Status
- Active recording ID
- Active transcript ID

### Recording

- ID
- Original filename
- Source path
- Optimized path
- SHA-256 checksum
- Duration
- Sample rate
- Channels
- Sample format
- Import timestamp

### Transcription Job

- ID
- Conversation ID
- Recording ID
- State
- Worker contract version
- Model
- Language mode
- Device
- Compute type
- Created, started, and completed timestamps
- Error code and message

### Transcript

- Schema version
- Transcript ID
- Conversation ID
- Recording ID
- Model and runtime metadata
- Detected language
- Duration
- Ordered segments

### Transcript Segment

- Sequence
- Start seconds
- End seconds
- Text
- Optional confidence metadata

Use explicit JSON schemas or strongly validated structures on both sides of the Go/Python boundary.

## CLI Commands in Scope

```text
echo version
echo doctor
echo new <title>
echo list
echo use <conversation-id>
echo status
echo add <wav-path>
echo transcribe [flags]
echo show [flags]
echo export markdown [flags]
```

A global `--conversation` flag may allow commands to avoid relying on active-conversation state.

## Engineering Principles

1. The CLI is a real product, not disposable scaffolding.
2. Go owns orchestration; Python owns inference.
3. Preserve original audio exactly and verify it with a checksum.
4. Structured data is canonical; Markdown is an export.
5. Every operation should be safe to retry.
6. Writes that define state should be atomic.
7. Errors should be actionable for a Windows user.
8. Subprocess contracts must be versioned and testable without a GPU.
9. GPU-specific tests must be clearly separated from ordinary CI.
10. Scope is enforced more aggressively than architecture is optimized.

## Explicit Non-Goals

Do not add:

- GUI or desktop shell
- Qwik
- FastAPI
- Supabase
- Authentication
- Browser recording
- Subscriptions or billing
- Cloud storage
- Redis or distributed queues
- Docker as a user requirement
- MP3 or video input
- Speaker diarization
- Live transcription
- Multi-user support
- Remote APIs

## Release Definition of Done

The CLI MVP is complete when a Windows user with FFmpeg, uv, Python, CUDA, cuDNN, and an NVIDIA GPU can:

```powershell
echo doctor
echo new "My Idea"
echo add .\my-idea.wav
echo transcribe --model large-v3
echo show
echo export markdown
```

and receive:

- The unchanged source WAV
- A normalized mono 16 kHz PCM WAV
- Durable job state
- Timestamped transcript JSON
- Readable Markdown export
- Clear progress and actionable errors

The Go CLI should be distributed as a Windows executable. The Python worker may remain an explicitly managed uv environment for v0.1.
