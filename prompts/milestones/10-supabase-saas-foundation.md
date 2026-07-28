# Milestone 10 Prompt — Supabase SaaS Foundation

Continue the Echo repository from Milestones 1–9.

## Objective

Convert the proven single-user local application into the foundation of a secure multi-user SaaS.

## Required Platform Components

### Qwik Frontend

- Sign up
- Login
- Logout
- Dashboard
- Conversation workspace
- Recorder
- Transcript viewer
- Usage page
- Subscription page

### Supabase

Use Supabase for:

- Authentication
- PostgreSQL
- Row Level Security
- User profiles
- Conversation metadata
- Recording metadata
- Transcript metadata
- Job status
- Subscription records

### Object Storage

Store:

- Raw audio
- Optimized audio
- Attachments
- Exports

Use private buckets and signed access.

### GPU Worker

The worker should:

1. Claim an authorized transcription job
2. Download audio
3. Optimize it
4. Run Whisper
5. Upload transcript artifacts
6. Update job status
7. Record usage
8. Remove temporary worker files

## Tenancy and Security

- Every user-owned row must have an owner ID
- Enforce access with Row Level Security
- Never rely only on frontend filtering
- Use short-lived signed URLs
- Authenticate GPU workers separately
- Validate all job ownership server-side

## Subscription Foundation

Support plans with limits for:

- Monthly transcription minutes
- Storage
- Maximum recording duration
- Retention
- Processing priority

Do not overbuild billing logic before the usage model is proven.

## Migration Requirement

Preserve the local CLI and local application.

The core transcript schema and processing engine should remain reusable across local and SaaS deployments.

## Definition of Done

Two separate users can sign up, create conversations, upload or record audio, receive transcripts, and remain fully isolated from each other's data.
