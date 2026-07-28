# Echo CLI v2.1 Spark Edition — Build Sequence

Each milestone is independently shippable and should be converted into small GitHub issues. The milestone prompt defines scope; issues define implementation units.

## Milestone 1 — Go Repository and CLI Foundation

Deliver a runnable Go CLI with `version` and `doctor`, configuration, path resolution, logging, and test infrastructure. Create the Python worker package as a stub only.

Suggested issues:

1. Initialize Go module and command tree.
2. Add configuration and Windows data paths.
3. Implement environment diagnostics.
4. Scaffold Python worker and contract fixtures.
5. Add CI and developer documentation.

## Milestone 2 — Conversation Workspace

Deliver local conversation creation, listing, selection, metadata persistence, and atomic filesystem operations.

Suggested issues:

1. Define conversation schema and storage layout.
2. Implement create/list/get operations.
3. Implement active conversation selection.
4. Add status command and corruption handling.

## Milestone 3 — WAV Import and Optimization

Deliver safe WAV validation, source preservation, metadata extraction, checksum verification, and FFmpeg optimization.

Suggested issues:

1. WAV extension and format validation.
2. Source import with collision policy and SHA-256.
3. FFprobe metadata inspection.
4. FFmpeg optimized derivative.
5. Idempotency and failure cleanup tests.

## Milestone 4 — Python Whisper Worker

Deliver an independently testable worker that transcribes an optimized WAV and writes the canonical transcript schema.

Suggested issues:

1. Worker CLI and settings model.
2. Contract schemas and progress events.
3. `faster-whisper` CUDA implementation.
4. Atomic output and structured errors.
5. CPU/fake-model test path for CI.

## Milestone 5 — Go/Worker Integration

Deliver `echo transcribe`, subprocess management, progress display, job state, result validation, cancellation, and safe retries.

Suggested issues:

1. Worker discovery and compatibility check.
2. Job state machine.
3. Subprocess execution and event parsing.
4. Result validation and transcript commit.
5. Interrupt, retry, and failure recovery.

## Milestone 6 — Transcript Review and Export

Deliver terminal review and deterministic Markdown export from canonical JSON.

Suggested issues:

1. Transcript repository and segment formatting.
2. `echo show` with timestamps and pagination behavior.
3. Markdown exporter.
4. Export collision and overwrite policy.
5. Golden-file tests.

## Milestone 7 — Reliability, Packaging, and v0.1 Release

Deliver a Windows-ready release candidate with integration tests, diagnostics, documentation, and reproducible builds.

Suggested issues:

1. End-to-end fixture tests with fake worker.
2. Optional GPU smoke-test procedure.
3. Windows build and release artifacts.
4. Installation and troubleshooting docs.
5. Versioning, changelog, and release checklist.

## Stop Condition

After Milestone 7, stop. Do not add an API, GUI, recorder, Supabase, or SaaS concerns to this repository phase. Collect real usage feedback before planning the next product surface.


## Spark Execution Rule

The suggested issues above are no longer loose suggestions. Each milestone file contains the canonical issue contracts and checkbox acceptance criteria. Use `/spark:plan` to create those issues, then ship them one at a time. A milestone is complete only after all issue PRs are merged and its definition of done passes on the default branch.
