# Milestone 7 Prompt — Local FastAPI Service

Continue the Echo repository from Milestones 1–6.

## Objective

Expose the proven Echo core through a local HTTP API without duplicating business logic.

## Architecture

```text
FastAPI routes
    ↓
Application services
    ↓
Echo core domain services
    ↓
Filesystem and Whisper
```

The CLI and API must use the same domain services.

## Required Endpoints

```text
GET    /health
GET    /conversations
POST   /conversations
GET    /conversations/{id}
POST   /conversations/{id}/recordings
POST   /conversations/{id}/transcriptions
GET    /conversations/{id}/transcript
GET    /jobs/{id}
```

## Requirements

- Run locally on `127.0.0.1`
- Use typed request and response models
- Return stable error objects
- Support multipart WAV upload
- Start transcription as a background job
- Expose job status
- Prevent duplicate active jobs
- Keep local filesystem storage
- Keep JSON persistence initially unless migration is justified

## Security

This is a local single-user service.

Bind to localhost by default.

Do not add external authentication yet.

## Testing

Use FastAPI test clients and mock transcription services.

## Definition of Done

The entire CLI workflow can be completed through the local API while the CLI remains functional.
