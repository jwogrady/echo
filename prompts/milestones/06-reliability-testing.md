# Milestone 6 Prompt — Reliability and Testing

Continue the Echo repository from Milestones 1–5.

## Objective

Harden the CLI into a dependable local product before adding an API or user interface.

## Required Improvements

### Job Tracking

Persist transcription jobs with:

- ID
- Conversation ID
- Recording ID
- Status
- Queued time
- Started time
- Completed time
- Model
- Device
- Error details

### Resumability

Support safe retry after:

- FFmpeg failure
- Interrupted transcription
- Model loading failure
- Output write failure

Do not redo completed work unnecessarily.

### Atomicity

Use temporary files and atomic replacement for:

- Conversation metadata
- Transcript JSON
- Segment JSON
- Exports

### Validation

Detect and report:

- Missing source files
- Corrupt WAV files
- Invalid metadata
- Unsupported state transitions
- Partial outputs
- Missing FFmpeg
- Missing GPU runtime

### Logging

Add:

- Human-readable console logs
- Rotating local log file
- Debug mode
- Correlation IDs for jobs

### Tests

Create:

- Unit tests for services and models
- Integration tests for FFmpeg
- Optional real-model transcription tests
- Failure-path tests
- Retry tests
- Atomic-write tests

### Documentation

Document:

- Installation on Windows
- FFmpeg setup
- NVIDIA and CUDA expectations
- Common errors
- Data directory layout
- Backup and recovery

## Definition of Done

The CLI can process multiple conversations repeatedly, survive common failures, retry safely, and pass a complete automated test suite.

Do not begin FastAPI work until this milestone is complete.
