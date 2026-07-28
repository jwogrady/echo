# ADR-0001: Go CLI with Python GPU Worker

- Status: Accepted
- Scope: Echo CLI v0.1

## Context

Echo needs a polished local Windows command-line interface and reliable NVIDIA GPU transcription. Go is strong for distributable CLIs, filesystem orchestration, concurrency, subprocess management, and long-term local-agent use. Python has the strongest practical ecosystem for Whisper inference, `faster-whisper`, CUDA, and future speech tooling.

A pure Python CLI would reduce initial language count but weaken the desired long-term executable and local-agent boundary. A pure Go implementation would increase native binding and CUDA integration risk during the MVP.

## Decision

Build the user-facing CLI and application orchestration in Go. Build the GPU inference worker in Python using uv and `faster-whisper`.

Connect them initially through a versioned subprocess protocol:

- Explicit command-line arguments identify input, output, model, language, and compute settings.
- The worker emits newline-delimited JSON progress events on stdout.
- Human-readable logs and diagnostics go to stderr.
- The worker writes a versioned transcript JSON result atomically.
- The Go CLI validates the result before committing job completion.

## Consequences

### Positive

- Single Go executable for the primary user experience
- Clear separation between orchestration and machine learning
- Strong Windows process and filesystem behavior
- Easy unit testing of the Go layer with a fake worker
- Access to the mature Python Whisper ecosystem
- Future worker transport can change without replacing the CLI

### Negative

- Two language toolchains
- Python and uv remain runtime prerequisites for v0.1
- Contract compatibility must be maintained
- Packaging is more involved than a single-language project

## Guardrails

- No conversation or export logic in Python.
- No direct Whisper-library dependency in Go.
- No unstructured parsing of human-readable worker output.
- Contract changes require a schema-version update and compatibility tests.
- Do not add HTTP, gRPC, or a queue until the subprocess contract proves insufficient.
