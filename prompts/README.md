# Echo CLI Prompt Kit v2

This kit contains the authoritative build plan for the first shippable version of Echo: a local Windows command-line tool that converts WAV recordings into structured, timestamped transcripts using a local NVIDIA GPU.

## Architecture

- Go owns the CLI, conversation model, filesystem orchestration, FFmpeg execution, configuration, progress, and exports.
- Python owns GPU inference through `faster-whisper`, CUDA, and cuDNN.
- The two components communicate through a versioned subprocess contract using JSON on disk and JSON status events on standard output.

## Contents

- `project-overview.md` — product scope, architecture, domain model, and definition of done.
- `build-sequence.md` — recommended order and issue boundaries.
- `decisions/ADR-0001-go-cli-python-worker.md` — architectural decision record.
- `milestones/` — one standalone implementation prompt per milestone.

## Recommended Claude + Spark Workflow

1. Place this kit in the Echo repository under `docs/planning/cli-v2/`.
2. Run `/spark:onboard` once if the repository has not been onboarded.
3. Give Claude `project-overview.md`, the ADR, and the current milestone prompt.
4. Use `/spark:plan` to convert the milestone into the smallest independently shippable issues.
5. For each issue, run `/spark:codify`, `/spark:validate`, and `/spark:ship`.
6. Do not begin the next milestone until the current milestone definition of done passes.

The milestone documents are authoritative. Do not let implementation sessions expand the product into a GUI, web service, or SaaS.
