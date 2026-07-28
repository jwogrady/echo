# Milestone 9 Prompt — Browser Recording

Continue the Echo repository from Milestones 1–8.

## Objective

Add reliable browser-based audio capture to the local Qwik application.

## Required Recorder Controls

- Record
- Pause
- Resume
- Stop
- Discard
- Save

## Recording Modes

### Manual Mode

The user directly controls pause and resume.

### Automatic Mode

The recorder automatically pauses after a configurable period without detected speech.

Configuration should include:

- Silence threshold
- Inactivity duration
- Optional countdown warning
- Manual resume
- Optional speech-triggered resume only if reliable

## Architecture

Record in short sequential chunks rather than one unbounded browser blob.

Each chunk should:

- Have a sequence number
- Have start and end timestamps
- Upload or persist independently
- Be recoverable after partial failure

The backend should assemble chunks into a logical recording session.

## Audio Requirements

- Preserve the raw browser recording chunks
- Create a server-side optimized WAV for transcription
- Keep browser-specific encoding concerns isolated from Whisper

## UI Requirements

Clearly display:

- Microphone permission state
- Recording state
- Manual pause state
- Automatic pause state
- Current duration
- Audio level
- Upload progress
- Processing state

## Failure Handling

Handle:

- Permission denial
- Device disconnection
- Browser refresh
- Partial upload
- Network interruption
- Empty recording

## Definition of Done

A user can record an idea in the browser, pause manually or automatically after inactivity, save the recording, and receive a transcript using the existing Echo transcription pipeline.
