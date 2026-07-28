# Milestone 5 Prompt — Transcript Export and Review

Continue the Echo repository from Milestones 1–4.

## Objective

Make transcripts easy to inspect, navigate, and export.

## Required Commands

```powershell
uv run echo show
uv run echo show --timestamps
uv run echo export markdown
uv run echo export json
uv run echo export text
```

## Terminal Review

The `show` command should display:

- Conversation title
- Recording name
- Model and language
- Audio duration
- Processing duration
- Timestamped transcript segments

Use readable timestamp formatting such as:

```text
[00:03:14] The first product decision is...
```

## Markdown Export

Create a polished Markdown export containing:

- Conversation title
- Metadata summary
- Source audio reference
- Transcript with timestamps
- Generated date

## JSON Export

Export a stable machine-readable schema.

Version the schema explicitly.

## Text Export

Provide plain text with optional timestamps.

## Export Rules

- Never overwrite without an explicit flag
- Use deterministic filenames
- Keep canonical transcript data separate from exported files
- Preserve UTF-8 text

## Testing

Add snapshot or golden-file tests for exports.

## Definition of Done

A transcript can be reviewed comfortably in the terminal and exported consistently to Markdown, JSON, and plain text.
