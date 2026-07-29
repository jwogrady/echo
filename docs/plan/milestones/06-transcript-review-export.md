# Milestone 6 Prompt — Transcript Review and Export

Read the project overview and completed transcription integration before changing code.

## Objective

Make completed transcripts useful from the terminal. Implement deterministic review and Markdown export while keeping transcript JSON canonical.

## Commands

Implement:

```text
ekko show
ekko export markdown
```

Useful flags may include:

```text
--conversation
--from
--to
--timestamps
--output
--overwrite
```

Do not invent a complex query language.

## Show Behavior

- Load and validate canonical transcript JSON.
- Present conversation title, recording, model, language, and duration.
- Render ordered transcript segments with human-readable timestamps.
- Handle long output sensibly using terminal-aware behavior, while remaining script-friendly when output is redirected.
- Provide a plain-output option if rich formatting is used.

## Markdown Export

Generate deterministic Markdown containing:

- Conversation title
- Export metadata
- Source recording information
- Whisper model and detected language
- Timestamped transcript sections

Use stable timestamp link or anchor conventions that can later support audio playback, but do not build a player.

The exporter must:

- Read canonical JSON only
- Produce identical output for identical input
- Refuse accidental overwrite unless explicitly allowed
- Write atomically
- Record the export in conversation metadata or an export manifest

## Tests

Include golden-file tests for:

- Short transcript
- Long transcript
- Unicode and punctuation
- Empty or whitespace-only segments
- Hour-plus timestamps
- Invalid transcript
- Deterministic export
- Collision and overwrite behavior
- Redirected/plain output

## Constraints

- No transcript editor.
- No summarization or generative AI.
- No HTML export.
- No audio player.

## Definition of Done

```powershell
ekko show
ekko export markdown
```

provides a readable terminal transcript and creates a deterministic Markdown document without changing canonical transcript data.

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

- Milestone 5 is merged.
- A canonical transcript fixture exists.
- Transcript schema validation is implemented.

## Spark Issue Contracts

### Issue 1: Implement transcript repository and formatting primitives

**Acceptance criteria**

- [ ] Canonical JSON is loaded and validated.
- [ ] Segments are ordered deterministically.
- [ ] Timestamp formatting supports hour-plus recordings.
- [ ] Formatting primitives have unit tests.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 2: Implement terminal transcript review

**Acceptance criteria**

- [ ] `ekko show` displays required metadata and timestamped text.
- [ ] Redirected output is script-friendly.
- [ ] Plain output is available when rich formatting is used.
- [ ] Invalid transcripts fail without mutation.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 3: Implement deterministic Markdown export

**Acceptance criteria**

- [ ] `ekko export markdown` reads canonical JSON only.
- [ ] Identical input produces identical output.
- [ ] Required recording and model metadata are included.
- [ ] Golden-file tests pass.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 4: Implement collision and overwrite policy

**Acceptance criteria**

- [ ] Existing exports are not overwritten by default.
- [ ] Explicit overwrite behavior is supported.
- [ ] Writes are atomic.
- [ ] Export records are persisted.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

### Issue 5: Complete edge-case and golden-file coverage

**Acceptance criteria**

- [ ] Unicode, punctuation, blank segments, long transcripts, and long timestamps are covered.
- [ ] Malformed input and redirected output are covered.
- [ ] No canonical transcript content is changed by show/export.
- [ ] All tests pass.

**Required artifacts**

- Focused implementation and tests.
- Updated help or documentation only where behavior changed.
- No generated user data, model files, local environments, or unrelated changes in the diff.

## Required Validation

Run and record the relevant results:

```text
go test ./...
go vet ./...
ekko show
ekko export markdown
Compare export against golden fixture
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
