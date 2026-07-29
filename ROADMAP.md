# Roadmap

What is planned, in priority order. Keep it honest — remove what has shipped
or been abandoned.

## v0.1.0 — Echo CLI MVP

**Status:** Planned

A local Windows command-line tool that turns a WAV recording into a
timestamped transcript on the user's own NVIDIA GPU. Go owns orchestration,
Python owns inference, connected by a versioned subprocess contract
([ADR-0001](docs/plan/decisions/ADR-0001-go-cli-python-worker.md)).

Done when a Windows user with FFmpeg, uv, Python, CUDA, cuDNN, and an NVIDIA
GPU can run `ekko doctor`, `ekko new`, `ekko add`, `ekko transcribe`,
`ekko show`, and `ekko export markdown`, and receive an unchanged source WAV, a
normalized mono 16 kHz PCM derivative, durable job state, timestamped
transcript JSON, and a readable Markdown export.

Scope, in build order — each item is independently shippable, and the release
lands when the seventh is green
([build sequence](docs/plan/build-sequence.md)):

1. **Go repository and CLI foundation** — runnable CLI with `version` and
   `doctor`, configuration, path resolution, and a stubbed Python worker.
   Planned as #2, #3, #4, #5, #6.
2. **Conversation workspace** — conversation creation, listing, selection, and
   atomic metadata persistence.
3. **WAV import and optimization** — validation, source preservation with
   SHA-256, metadata inspection, and the FFmpeg derivative.
4. **Python Whisper worker** — independently testable `faster-whisper`
   transcription writing the canonical transcript schema.
5. **Go/worker integration** — `ekko transcribe`, subprocess management, job
   state, progress, cancellation, and safe retries.
6. **Transcript review and export** — `ekko show` and deterministic Markdown
   export from canonical JSON.
7. **Reliability, packaging, and release** — end-to-end fixture tests, Windows
   build artifacts, and installation docs.

Not in this release: GUI or desktop shell, HTTP API, browser recording,
authentication, hosted storage, billing, MP3 or video input, speaker
diarization, live transcription, and multi-user support. The full list is in
the [project overview](docs/plan/project-overview.md).

## After v0.1.0

Nothing planned. Collect real usage feedback before scoping the next product
surface.
