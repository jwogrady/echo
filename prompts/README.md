# Echo CLI Prompt Kit v2.1 — Spark Edition

This kit is the authoritative, Spark-optimized build plan for Echo CLI v0.1: a local Windows command-line tool that converts WAV recordings into structured, timestamped transcripts using a local NVIDIA GPU.

## Architecture

- Go owns CLI orchestration, conversations, files, FFmpeg, configuration, progress, jobs, and exports.
- Python owns GPU transcription through `faster-whisper`, CUDA, and cuDNN.
- Go and Python communicate through a versioned subprocess contract: NDJSON events on stdout, diagnostics on stderr, and canonical JSON artifacts on disk.

## Contents

- `project-overview.md` — product scope and definition of done.
- `build-sequence.md` — milestone order and issue map.
- `SPARK-WORKFLOW.md` — exact Spark operating contract.
- `decisions/ADR-0001-go-cli-python-worker.md` — architecture decision.
- `milestones/` — standalone milestone prompts with preconditions, ready-to-create issue contracts, validation commands, ship gates, and stop rules.

## Start Here

1. Place this kit under `docs/planning/cli-v2.1-spark/`.
2. Run `/spark:onboard` once, then `spark doctor`.
3. Commit the planning kit.
4. Read `SPARK-WORKFLOW.md`.
5. Start Milestone 1 with `/spark:plan`.
6. Create the supplied issues and execute each through `/spark:codify`, `/spark:validate`, and `/spark:ship`.

The milestone documents are authoritative. Do not expand Echo into a GUI, API, browser recorder, Supabase project, or SaaS during this build.
