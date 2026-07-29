"""Tests that the committed contract fixtures obey the contract they document.

These fixtures are the shared target both languages build against, so a fixture
that drifts from the documented rules is a defect in its own right — it would
teach the Go side the wrong shape.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from echo_worker.contract import (
    CONTRACT_VERSION,
    EVENT_TYPES,
    TRANSCRIPT_SCHEMA_VERSION,
    supports_contract_version,
)

FIXTURES = Path(__file__).resolve().parents[2] / "contract" / f"v{CONTRACT_VERSION}"

TERMINAL_EVENT_TYPES = {"completed", "failed"}


def load_events(name: str) -> list[dict]:
    """Parse an NDJSON fixture, rejecting blank or malformed lines."""
    path = FIXTURES / name
    events = []
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        assert line.strip(), f"{name}:{number} is blank; NDJSON carries one event per line"
        events.append(json.loads(line))

    return events


def test_fixture_directory_exists() -> None:
    assert FIXTURES.is_dir(), f"missing contract fixtures at {FIXTURES}"


@pytest.mark.parametrize("name", ["events.ndjson", "failed-events.ndjson"])
def test_every_event_declares_the_contract_version(name: str) -> None:
    for event in load_events(name):
        assert event["version"] == CONTRACT_VERSION


@pytest.mark.parametrize("name", ["events.ndjson", "failed-events.ndjson"])
def test_every_event_type_is_known(name: str) -> None:
    for event in load_events(name):
        assert event["type"] in EVENT_TYPES, f"{event['type']} is not a contract event type"


@pytest.mark.parametrize(
    ("name", "terminal"),
    [("events.ndjson", "completed"), ("failed-events.ndjson", "failed")],
)
def test_a_run_ends_with_exactly_one_terminal_event(name: str, terminal: str) -> None:
    events = load_events(name)

    terminals = [event for event in events if event["type"] in TERMINAL_EVENT_TYPES]
    assert len(terminals) == 1, "a run must have exactly one terminal event"
    assert terminals[0]["type"] == terminal
    assert events[-1]["type"] == terminal, "the terminal event must be last"


def test_segment_sequences_start_at_one_and_are_contiguous() -> None:
    segments = [event for event in load_events("events.ndjson") if event["type"] == "segment"]

    assert segments, "the success fixture must contain segments"
    assert [segment["sequence"] for segment in segments] == list(range(1, len(segments) + 1))


def test_segment_times_do_not_move_backwards() -> None:
    segments = [event for event in load_events("events.ndjson") if event["type"] == "segment"]

    previous_end = 0.0
    for segment in segments:
        assert segment["start"] <= segment["end"], "a segment cannot end before it starts"
        assert segment["start"] >= previous_end, "segments must not overlap or reverse"
        previous_end = segment["end"]


def test_completed_event_agrees_with_the_segments_it_summarizes() -> None:
    events = load_events("events.ndjson")

    segments = [event for event in events if event["type"] == "segment"]
    completed = next(event for event in events if event["type"] == "completed")

    assert completed["segments"] == len(segments)
    assert completed["duration_seconds"] == pytest.approx(segments[-1]["end"])


def test_failure_event_carries_a_code_and_message() -> None:
    failed = next(
        event for event in load_events("failed-events.ndjson") if event["type"] == "failed"
    )

    assert failed["code"], "a failure must carry a machine-readable code"
    assert failed["message"], "a failure must carry a human-readable message"


def load_transcript() -> dict:
    return json.loads((FIXTURES / "transcript.json").read_text(encoding="utf-8"))


def test_transcript_declares_its_schema_version() -> None:
    assert load_transcript()["schema_version"] == TRANSCRIPT_SCHEMA_VERSION


def test_transcript_carries_the_documented_metadata() -> None:
    """The project overview names each of these as canonical."""
    transcript = load_transcript()

    for field in (
        "transcript_id",
        "conversation_id",
        "recording_id",
        "worker",
        "model",
        "language",
        "duration_seconds",
        "segments",
    ):
        assert field in transcript, f"transcript is missing {field}"

    assert transcript["worker"]["contract_version"] == CONTRACT_VERSION
    assert transcript["model"]["name"]
    assert transcript["language"]["detected"]


def test_transcript_segments_are_ordered_and_complete() -> None:
    segments = load_transcript()["segments"]

    assert [segment["sequence"] for segment in segments] == list(range(1, len(segments) + 1))

    previous_end = 0.0
    for segment in segments:
        assert segment["text"].strip(), "a segment must carry text"
        assert segment["start"] <= segment["end"]
        assert segment["start"] >= previous_end
        previous_end = segment["end"]


def test_transcript_duration_matches_its_last_segment() -> None:
    transcript = load_transcript()

    assert transcript["duration_seconds"] == pytest.approx(transcript["segments"][-1]["end"])


def test_confidence_is_optional() -> None:
    """Both shapes appear in the fixture so parsers must handle each."""
    segments = load_transcript()["segments"]

    assert any("confidence" in segment for segment in segments)
    assert any("confidence" not in segment for segment in segments)


def test_worker_refuses_unknown_contract_versions() -> None:
    assert supports_contract_version(CONTRACT_VERSION)
    assert not supports_contract_version(CONTRACT_VERSION + 1)
    assert not supports_contract_version(0)
