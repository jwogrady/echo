# Milestone 7 Prompt — Reliability, Packaging, and v0.1 Release

Read all project documents and treat the current CLI scope as frozen.

## Objective

Turn the completed feature set into a trustworthy Windows v0.1 release. Focus on verification, packaging, diagnostics, documentation, and honest limitations. Do not add product features.

## End-to-End Verification

Create a hermetic integration test using:

- Temporary data directory
- Generated WAV fixture
- Fake worker with valid event stream and transcript output

Verify the full flow:

```text
doctor -> new -> use -> add -> transcribe -> show -> export
```

Add failure-path integration tests for FFmpeg failure, worker failure, corrupt state, interrupted transcription, and retry.

## Windows Packaging

Produce reproducible Windows builds for the Go CLI, including version metadata. The primary artifact should be clearly named and checksummed.

For v0.1, document the Python worker installation as a managed uv environment rather than pretending it is bundled into the executable.

Provide a bootstrap or setup command/script only if it remains simple, inspectable, and reversible. Do not introduce a heavy installer framework merely for polish.

## Documentation

Deliver:

- Installation prerequisites
- Windows setup
- CUDA/cuDNN and `faster-whisper` verification
- FFmpeg installation verification
- Worker setup with uv
- Quick start
- Command reference
- Data directory and backup guidance
- Troubleshooting matrix based on actual error codes
- Uninstall and data-removal instructions
- Privacy statement explaining that audio and transcripts remain local
- Known limitations

## Release Quality

- `go test ./...` passes.
- `go vet ./...` passes.
- Python tests and linting pass.
- CI does not require a GPU.
- Manual GPU smoke test is documented and performed where hardware is available.
- `echo doctor` catches the most common installation problems.
- Version output identifies CLI version, worker version, and contract compatibility.
- Changelog and release notes describe only shipped behavior.

## Scope Audit

Search the code and docs for accidental promises or dependencies involving:

- Web UI
- API server
- Supabase
- Authentication
- Browser recording
- SaaS
- Billing
- Cloud processing

Remove or clearly label them as future ideas outside v0.1.

## Definition of Done

A new Windows user can follow the documentation, install prerequisites, obtain the Go executable, configure the uv worker, run the complete WAV-to-Markdown workflow, understand failures, and back up or remove all local Echo data.

Stop after this release. Gather usage evidence before expanding the product.

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

- Milestones 1 through 6 are merged.
- The default branch is green.
- CLI scope is frozen for v0.1.

## Spark Issue Contracts

### Issue 1: Add hermetic end-to-end tests

**Acceptance criteria**

- [ ] The full doctor-to-export flow runs in a temporary data root.
- [ ] A generated WAV and fake worker make the suite GPU-independent.
- [ ] Key failure paths and retry behavior are covered.
- [ ] Tests are reliable in CI.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Document and perform optional GPU smoke test

**Acceptance criteria**

- [ ] A Windows RTX/CUDA procedure is documented.
- [ ] Expected outputs and common failures are described.
- [ ] The result is recorded without making GPU hardware a CI requirement.
- [ ] No unsupported performance claims are added.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Produce reproducible Windows release artifacts

**Acceptance criteria**

- [ ] A versioned Windows executable is produced.
- [ ] Artifact checksums are generated.
- [ ] Version metadata is embedded and shown by `echo version`.
- [ ] Build steps are documented and repeatable.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Complete installation and troubleshooting documentation

**Acceptance criteria**

- [ ] Prerequisites, uv worker setup, FFmpeg, CUDA, quick start, backup, uninstall, privacy, and limitations are covered.
- [ ] Troubleshooting maps actual error codes to remediation.
- [ ] Documentation matches observed behavior.
- [ ] No future feature is presented as shipped.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 5: Prepare and verify v0.1 release

**Acceptance criteria**

- [ ] Go and Python checks pass.
- [ ] Changelog and release notes describe only shipped behavior.
- [ ] A scope audit finds no accidental SaaS dependencies or promises.
- [ ] Release checklist and stop condition are satisfied.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
go test ./...
go vet ./...
cd worker && uv run pytest
Run hermetic end-to-end suite
Build Windows executable and verify checksum
Run scope-audit searches listed in the milestone
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
