# Echo MVP Prompt Kit

Echo is a voice-capture and transcription product that begins as a local Windows CLI for WAV files, grows into a local application, and eventually becomes a multi-user SaaS.

This kit contains:

- `project-overview.md` — product vision, architecture, constraints, and build philosophy
- `build-sequence.md` — the recommended milestone order and definition of done
- `milestones/` — one standalone implementation prompt per milestone

## Recommended Use

Give your coding agent `project-overview.md` first. Then run the milestone prompts in numerical order.

Each milestone prompt is designed to be usable independently, but assumes the repository already includes the work from previous milestones.

## Core Principle

Do not build the SaaS first.

Build the product in this order:

1. Local transcription engine
2. Windows CLI
3. Local API
4. Qwik application
5. Browser recorder
6. Single-user server
7. Supabase-backed SaaS

The CLI remains a permanent interface for automation, debugging, batch processing, and advanced use.
