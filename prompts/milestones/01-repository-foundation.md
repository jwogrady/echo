# Milestone 1 Prompt — Repository Foundation

You are building Echo, a local-first voice transcription product that will begin as a Windows CLI and later grow into a Qwik application and SaaS.

Read `project-overview.md` before making changes.

## Objective

Create a clean Python repository foundation for a production-quality CLI without implementing Whisper transcription yet.

## Requirements

Use:

- Python 3.12+
- uv
- Typer
- Rich
- Pydantic v2
- pytest
- Ruff
- mypy or pyright-compatible typing

Create a `src` package layout.

Recommended structure:

```text
src/echo/
├── __init__.py
├── cli.py
├── config.py
├── models.py
├── logging.py
├── paths.py
├── conversations.py
├── audio.py
├── transcription.py
├── exporters.py
└── storage.py
```

Implement a CLI application named `echo` with placeholder commands:

```text
echo version
echo doctor
echo new
echo add
echo transcribe
echo show
echo export
```

Only `version` and `doctor` need meaningful behavior in this milestone.

`echo doctor` should inspect and report:

- Python version
- Operating system
- Echo data directory
- FFmpeg availability
- NVIDIA GPU visibility where detectable
- CUDA-related environment information where detectable

Create typed Pydantic models for:

- Conversation
- Recording
- AudioMetadata
- Transcript
- TranscriptSegment
- TranscriptionJob

Create a configurable data root with a sensible Windows default and an environment variable override.

Add structured logging with useful human-readable terminal output.

## Quality Requirements

- No business logic inside Typer command functions
- No hardcoded user-specific paths
- Clear exceptions with user-friendly messages
- Full type hints
- Unit tests for configuration, paths, and model serialization
- README setup instructions using uv
- `.gitignore`, `pyproject.toml`, and development commands

## Deliverables

- Runnable CLI
- Passing tests
- Formatting and linting configuration
- Documented project structure
- No Whisper dependency required yet

## Definition of Done

These commands work:

```powershell
uv sync
uv run echo version
uv run echo doctor
uv run pytest
uv run ruff check .
```

Do not add Qwik, Supabase, FastAPI, Docker, or cloud services.
