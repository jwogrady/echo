# Worker contract fixtures

The Go CLI and the Python worker talk over a versioned subprocess protocol
([ADR-0001](../docs/plan/decisions/ADR-0001-go-cli-python-worker.md)). These
fixtures are the shared, committed examples of that protocol. Both languages
test against these same files, so neither side can drift from the contract
without a test failing.

They live at the repository root rather than under `worker/` because both
languages consume them: the worker validates that what it emits matches, and the
Go integration tests replay them through a fake worker with no GPU present.

## Layout

```text
contract/
└── v1/
    ├── events.ndjson         a successful run's stdout stream
    ├── failed-events.ndjson  a run that fails during model load
    └── transcript.json       the canonical transcript document
```

One directory per contract version. A new version is a new directory, never an
edit to an existing one — old fixtures stay so compatibility can be tested
across versions.

## The rules these fixtures encode

- Stdout carries newline-delimited JSON events and nothing else. Human-readable
  diagnostics go to stderr.
- Every event carries `version`. Every transcript carries `schema_version`.
- Event types are `started`, `progress`, `segment`, `completed`, and `failed`.
- A run ends with exactly one terminal event: `completed` or `failed`.
- Segments are ordered and `sequence` starts at 1.
- `confidence` is optional; the third segment in `transcript.json` carries one
  and the others do not, so parsers are forced to handle both.

## Changing the contract

Bump `CONTRACT_VERSION` or `TRANSCRIPT_SCHEMA_VERSION` in
`worker/src/echo_worker/contract.py`, add a new `contract/vN/` directory, and
update both languages' compatibility tests. ADR-0001 requires a schema-version
change for any contract change.

## Status

The fixtures are hand-written from the protocol the ADR and the milestone
documents specify. No worker has produced them yet — transcription is not
implemented. Their purpose today is to let both sides be built against a fixed
target; the worker's own output is asserted against them once inference lands.
