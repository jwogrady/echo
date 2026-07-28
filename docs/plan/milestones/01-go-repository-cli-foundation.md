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

## Spark Planning Instructions

Use `/spark:plan` to convert this milestone into the issue contracts below. Create one GitHub issue per contract unless repository evidence proves a smaller split is necessary. Do not combine issues merely to reduce issue count.

For every issue:

- Keep one observable capability per issue.
- Use one focused branch and one pull request.
- Copy the acceptance criteria into the GitHub issue as checkboxes.
- Add implementation notes only when they are evidence-backed and necessary.
- Do not pull work forward from later milestones.
- Do not perform unrelated cleanup, renaming, or dependency upgrades.
- Update documentation only when behavior or a contract changed.

## Preconditions

- Repository exists and is checked out on a clean default branch.
- Go and uv are installed or their absence can be diagnosed.
- No prior Echo implementation must be preserved unless explicitly documented.

## Spark Issue Contracts

### Issue 1: Initialize Go CLI and command tree

**Acceptance criteria**

- [ ] A Go module and `cmd/echo` entrypoint exist.
- [ ] `echo version` returns version information and exits zero.
- [ ] Unimplemented commands fail with a consistent, user-facing message.
- [ ] `go test ./...` passes.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Implement configuration and Windows path resolution

**Acceptance criteria**

- [ ] Echo resolves a deterministic Windows data directory.
- [ ] A data-root override is supported for tests and automation.
- [ ] No user-specific path is hardcoded.
- [ ] Path behavior has unit tests.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Implement environment diagnostics

**Acceptance criteria**

- [ ] `echo doctor` reports required and optional dependencies separately.
- [ ] FFmpeg, ffprobe, uv, Python, NVIDIA tooling, and worker location are checked.
- [ ] Diagnostics include actionable remediation.
- [ ] Strict mode returns nonzero when required conditions fail.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Scaffold Python worker and contract fixtures

**Acceptance criteria**

- [ ] `worker/` is a valid uv project.
- [ ] `echo-worker version` works without Whisper dependencies.
- [ ] Versioned contract fixtures exist for later integration tests.
- [ ] `uv run pytest` passes.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 5: Add CI and developer documentation

**Acceptance criteria**

- [ ] CI runs Go tests and vet plus Python tests without a GPU.
- [ ] README documents local setup and validation commands.
- [ ] Generated files and local data are ignored.
- [ ] No product features beyond this milestone are added.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
go test ./...
go vet ./...
go run ./cmd/echo version
go run ./cmd/echo doctor
cd worker && uv sync && uv run pytest
```

## Ship Gate

Do not run `/spark:ship` until:

- Every acceptance criterion for the current issue is satisfied.
- Required automated checks pass.
- The diff contains no work from a later milestone.
- User-facing behavior and documentation agree.
- Generated files, secrets, recordings, transcripts, models, and local environments are excluded.
- The pull request links the GitHub issue and states how the behavior was verified.

## Stop Rule

After shipping the current issue, stop. Resume with the next approved GitHub issue through `/spark:codify`; do not continue implementing from the milestone prompt on the same branch.
