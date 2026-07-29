# Echo

Echo turns WAV recordings into structured, timestamped transcripts using a local
NVIDIA GPU. It is for people who think by talking: record, import, transcribe,
read, export. Audio and transcripts never leave the machine.

The command is **`ekko`**, not `echo` — `echo` is a shell builtin in zsh and bash
and an alias for `Write-Output` in PowerShell, so a program installed under that
name can never be reached. See
[ADR-0002](docs/plan/decisions/ADR-0002-command-name.md).

## Status

**Early. Not usable for its purpose yet.**

Milestone 1 of 7 is complete. This build implements two commands:

| Command | State |
| --- | --- |
| `ekko version` | works |
| `ekko doctor` | works |
| `ekko new`, `list`, `use`, `status` | not implemented — Milestone 2 |
| `ekko add` | not implemented — Milestone 3 |
| `ekko transcribe` | not implemented — Milestone 5 |
| `ekko show`, `export` | not implemented — Milestone 6 |

Unimplemented commands are registered and exit 3 with an explanation rather than
looking like a typo. Nothing transcribes audio yet: the Python worker is a
scaffold with no `faster-whisper` dependency.

The plan lives in [`docs/plan/`](docs/plan/) — see
[`project-overview.md`](docs/plan/project-overview.md) for product scope and
[`build-sequence.md`](docs/plan/build-sequence.md) for the milestone order.
[`ROADMAP.md`](ROADMAP.md) tracks progress toward v0.1.0.

## Architecture

Go owns orchestration; Python owns inference
([ADR-0001](docs/plan/decisions/ADR-0001-go-cli-python-worker.md)). They speak a
versioned subprocess protocol: NDJSON events on stdout, human diagnostics on
stderr, and a canonical transcript JSON written atomically to disk.

```text
cmd/ekko/             CLI entrypoint
internal/app/         command tree, exit codes, error reporting
internal/buildinfo/   command name and build identity
internal/config/      data-directory resolution
internal/diagnostics/ environment inspection behind ekko doctor
internal/worker/      worker discovery and the Go side of the contract
worker/               uv project: the Python GPU worker
contract/v1/          shared protocol fixtures both languages test against
docs/plan/            the build plan, ADRs, and milestone contracts
```

## Prerequisites

Building and testing needs only Go and uv. The full transcription pipeline also
needs FFmpeg and an NVIDIA GPU — run `ekko doctor` to see what is missing.

| Tool | Needed for | Install |
| --- | --- | --- |
| Go 1.26+ | building the CLI | [go.dev/dl](https://go.dev/dl/) |
| uv | the Python worker | [docs.astral.sh/uv](https://docs.astral.sh/uv/getting-started/installation/) |
| FFmpeg + ffprobe | audio import and conversion | `winget install Gyan.FFmpeg`, `apt install ffmpeg`, or `brew install ffmpeg` |
| NVIDIA driver + CUDA | GPU transcription | vendor installer |

Python is managed by uv; you do not need a system interpreter.

## Setup

```bash
git clone git@github.com:jwogrady/echo.git
cd echo

# Go side
go build ./cmd/ekko

# Python side
cd worker && uv sync && cd ..

# See what your machine is missing
go run ./cmd/ekko doctor
```

`ekko doctor` groups findings into required and optional, distinguishes a missing
tool from a broken one, and prints a remediation line for anything that fails.
It exits 0 even when something is wrong, so the report is always readable; add
`--strict` to make it a gate that exits nonzero.

## Validation

Everything CI runs, runnable locally. No GPU, no model download.

```bash
# Go
gofmt -l .            # must print nothing
go vet ./...
go test -race ./...
GOOS=windows GOARCH=amd64 go build ./...   # the primary target

# Python
cd worker
uv sync --locked
uv run ruff check .
uv run ruff format --check .
uv run pytest
```

CI gates both languages in separate jobs on every push and pull request, so a
failure names which side broke. It never requires a GPU; GPU transcription is
verified by a manual smoke test documented in Milestone 7.

## Where Echo stores data

| Platform | Location |
| --- | --- |
| Windows (primary target) | `%LOCALAPPDATA%\Echo` |
| macOS | `~/Library/Application Support/Echo` |
| Linux | `$XDG_DATA_HOME/echo`, else `~/.local/share/echo` |

`LOCALAPPDATA` rather than `APPDATA` is deliberate: it is machine-local and
excluded from roaming profiles, which is correct for recordings and transcripts.

Two environment variables override discovery:

- `ECHO_DATA_DIR` — the data root. Set it in tests and automation so they never
  touch real conversations.
- `ECHO_WORKER_DIR` — the Python worker project, when it is not beside the
  executable or in the working directory.

## Contributing

Read [`CONTRIBUTING.md`](CONTRIBUTING.md) for the branch and commit conventions,
and [`CONVENTIONS.md`](CONVENTIONS.md) plus
[`ENGINEERING-STANDARDS.md`](ENGINEERING-STANDARDS.md) for the project contract.

Two rules worth knowing before your first pull request:

- Commits are conventional, and the `commit-msg` hook enforces it: the subject
  must start with `feat`, `fix`, `docs`, `chore`, `refactor`, or `test` and stay
  under 72 characters.
- Never commit recordings, transcripts, model weights, or virtual environments.
  `.gitignore` covers them; the WAV fixture exceptions are deliberate and
  narrow.

Changing the Go/Python protocol means bumping a version in
`worker/src/echo_worker/contract.py`, adding a new `contract/vN/` directory, and
updating both languages' tests — see [`contract/README.md`](contract/README.md).

## Privacy

Echo runs entirely on your machine. It makes no network requests: audio,
transcripts, and exports stay in the data directory above. There is no
telemetry, no account, and no cloud processing.

## License

Proprietary. All rights reserved — see [`LICENSE`](LICENSE).
