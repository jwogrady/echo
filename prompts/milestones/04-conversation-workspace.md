# Milestone 4 Prompt — Conversation Workspace

Continue the Echo repository from Milestones 1–3.

## Objective

Turn single-file transcription into a persistent conversation-based workflow.

## Required Commands

```powershell
uv run echo new "Product Strategy"
uv run echo list
uv run echo open <conversation-id>
uv run echo add .\idea.wav
uv run echo transcribe
uv run echo status
```

## Conversation Behavior

Each conversation must have:

- Stable unique ID
- Title
- Slug
- Created and updated timestamps
- Status
- Optional description
- Optional language
- Recordings
- Transcript references
- Error state where applicable

Use a portable folder structure under the Echo data root.

## Active Conversation

Implement a simple active-conversation mechanism so commands can operate without repeatedly passing the ID.

The active selection should be explicit, inspectable, and easy to clear.

## Audio Import

`echo add` should:

- Validate the input
- Copy the source into the conversation
- Preserve its original filename in metadata
- Avoid collisions
- Record file size and audio metadata
- Never modify the source

## State Transitions

Use explicit states such as:

- draft
- ready
- processing
- transcribed
- failed

Prevent invalid transitions.

## Persistence

Use JSON files for this milestone.

All writes must be atomic.

## Definition of Done

A user can create a conversation, import a WAV file, transcribe it, close the terminal, reopen Echo, and still inspect the complete conversation state.
