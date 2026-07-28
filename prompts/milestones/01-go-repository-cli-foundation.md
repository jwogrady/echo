# Milestone 1 Prompt — Go Repository and CLI Foundation

Read `project-overview.md` and `decisions/ADR-0001-go-cli-python-worker.md` before changing code.

## Objective

Create a production-quality repository foundation for Echo CLI. Implement a real Go command-line application with environment diagnostics. Scaffold, but do not implement, the Python GPU worker.

## Required CLI Behavior

Implement:

```text
echo version
echo doctor
```

Create placeholders that return a clear "not implemented in this milestone" result for:

```text
echo new
echo list
echo use
echo status
echo add
echo transcribe
echo show
echo export
```

`echo doctor` must report:

- Echo version
- Operating system and architecture
- Resolved data directory and whether it is writable
- FFmpeg and ffprobe availability and versions
- `uv` availability and version
- Python availability and version
- NVIDIA tooling visibility, including `nvidia-smi` where available
- Worker package location and current compatibility status

Diagnostics must distinguish required, optional, unavailable, and misconfigured dependencies. Exit nonzero only when a requested strict mode is used or a required condition prevents core operation.

## Repository Requirements

Create a conventional Go module with thin command handlers and testable internal services. Establish packages for configuration, diagnostics, storage paths, terminal presentation, and worker contracts.

Scaffold `worker/` as a uv Python project with a placeholder `echo-worker version` command and no Whisper dependency yet.

Add:

- Go formatting, vetting, and tests
- Python formatting and tests for the stub worker
- Cross-platform path abstractions with Windows as the primary target
- Structured errors with user-facing remediation
- README development setup
- `.gitignore`
- CI that does not require a GPU

## Constraints

- No conversation implementation yet.
- No FFmpeg conversion yet.
- No Whisper model download.
- No Cobra business logic inside command definitions.
- No hardcoded user paths.
- No GUI, API, Docker requirement, or SaaS infrastructure.

## Definition of Done

The following work in PowerShell:

```powershell
go test ./...
go vet ./...
go run ./cmd/echo version
go run ./cmd/echo doctor
cd worker
uv sync
uv run echo-worker version
uv run pytest
```

The output is clear enough that a Windows user knows exactly what prerequisite is missing and how Echo resolved its data and worker paths.
