# Milestone 7 Prompt — Reliability, Packaging, and v0.1 Release

Read all project documents and treat the current CLI scope as frozen.

## Objective

Turn the completed feature set into a trustworthy Windows v0.1 release. Focus on verification, packaging, diagnostics, documentation, and honest limitations. Do not add product features.

## End-to-End Verification

Create a hermetic integration test using:

- Temporary data directory
- Generated WAV fixture
- Fake worker with valid event stream and transcript output

Verify the full flow:

```text
doctor -> new -> use -> add -> transcribe -> show -> export
```

Add failure-path integration tests for FFmpeg failure, worker failure, corrupt state, interrupted transcription, and retry.

## Windows Packaging

Produce reproducible Windows builds for the Go CLI, including version metadata. The primary artifact should be clearly named and checksummed.

For v0.1, document the Python worker installation as a managed uv environment rather than pretending it is bundled into the executable.

Provide a bootstrap or setup command/script only if it remains simple, inspectable, and reversible. Do not introduce a heavy installer framework merely for polish.

## Documentation

Deliver:

- Installation prerequisites
- Windows setup
- CUDA/cuDNN and `faster-whisper` verification
- FFmpeg installation verification
- Worker setup with uv
- Quick start
- Command reference
- Data directory and backup guidance
- Troubleshooting matrix based on actual error codes
- Uninstall and data-removal instructions
- Privacy statement explaining that audio and transcripts remain local
- Known limitations

## Release Quality

- `go test ./...` passes.
- `go vet ./...` passes.
- Python tests and linting pass.
- CI does not require a GPU.
- Manual GPU smoke test is documented and performed where hardware is available.
- `echo doctor` catches the most common installation problems.
- Version output identifies CLI version, worker version, and contract compatibility.
- Changelog and release notes describe only shipped behavior.

## Scope Audit

Search the code and docs for accidental promises or dependencies involving:

- Web UI
- API server
- Supabase
- Authentication
- Browser recording
- SaaS
- Billing
- Cloud processing

Remove or clearly label them as future ideas outside v0.1.

## Definition of Done

A new Windows user can follow the documentation, install prerequisites, obtain the Go executable, configure the uv worker, run the complete WAV-to-Markdown workflow, understand failures, and back up or remove all local Echo data.

Stop after this release. Gather usage evidence before expanding the product.
