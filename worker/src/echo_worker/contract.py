"""The versioned subprocess contract between the Go CLI and this worker.

ADR-0001 makes the boundary a subprocess protocol so a later HTTP or gRPC
transport can replace it without touching domain models. This module holds the
version numbers both sides agree on; the fixtures under ``contract/`` at the
repository root are the shared examples each language tests against.

Changing an event shape or a transcript field requires bumping the matching
version here and updating the fixtures.
"""

from __future__ import annotations

#: Version of the stdout event protocol. Every event carries it as "version".
CONTRACT_VERSION = 1

#: Version of the transcript document written to disk.
TRANSCRIPT_SCHEMA_VERSION = 1

#: Event types the worker may emit on stdout. Nothing else is valid.
EVENT_TYPES = frozenset(
    {
        "started",
        "progress",
        "segment",
        "completed",
        "failed",
    }
)


def supports_contract_version(requested: int) -> bool:
    """Report whether this worker can speak the requested protocol version.

    The worker refuses rather than guesses: a caller expecting a version it does
    not implement would silently misread the event stream.
    """
    return requested == CONTRACT_VERSION
