# Milestone 6 Prompt — Transcript Review and Export

Read the project overview and completed transcription integration before changing code.

## Objective

Make completed transcripts useful from the terminal. Implement deterministic review and Markdown export while keeping transcript JSON canonical.

## Commands

Implement:

```text
echo show
echo export markdown
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
echo show
echo export markdown
```

provides a readable terminal transcript and creates a deterministic Markdown document without changing canonical transcript data.
