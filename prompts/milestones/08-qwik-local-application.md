# Milestone 8 Prompt — Qwik Local Application

Continue the Echo repository from Milestones 1–7.

## Objective

Build a local Qwik web application on top of the Echo FastAPI service.

## Required Screens

### Conversation Dashboard

- List conversations
- Show status
- Create conversation
- Open conversation

### Conversation Workspace

- Edit title and description
- Add an outline
- Upload a WAV file
- Start transcription
- Show processing progress
- Show errors

### Transcript Viewer

- Display timestamped transcript
- Play the source audio
- Seek audio by clicking timestamps
- Export Markdown, JSON, or text

## Technical Requirements

- Use Qwik and Qwik City
- Keep API access in a dedicated client layer
- Use typed API contracts where practical
- Handle loading, empty, processing, success, and failure states
- Keep visual design simple and functional
- Make desktop Windows usage the primary target

## Non-Goals

Do not add:

- Supabase
- User accounts
- Billing
- Teams
- Cloud hosting
- Browser recording

## Definition of Done

A user can create a conversation, upload a WAV file, transcribe it, review the timestamped transcript, play audio, and export results entirely through the local web interface.
