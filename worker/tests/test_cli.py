"""Tests for the worker command surface.

Nothing here imports faster-whisper or touches a GPU: the scaffold must be
verifiable on a machine that has neither.
"""

from __future__ import annotations

import pytest

from echo_worker import __version__
from echo_worker.cli import (
    EXIT_NOT_IMPLEMENTED,
    EXIT_OK,
    EXIT_USAGE,
    PENDING_COMMANDS,
    PROGRAM,
    main,
)
from echo_worker.contract import CONTRACT_VERSION, TRANSCRIPT_SCHEMA_VERSION


def test_version_succeeds(capsys: pytest.CaptureFixture[str]) -> None:
    assert main(["version"]) == EXIT_OK

    captured = capsys.readouterr()
    assert captured.err == ""
    assert f"{PROGRAM} {__version__}" in captured.out


def test_version_reports_both_contract_versions(capsys: pytest.CaptureFixture[str]) -> None:
    """The Go CLI checks compatibility before invoking, so both must be visible."""
    main(["version"])

    out = capsys.readouterr().out
    assert f"contract version {CONTRACT_VERSION}" in out
    assert f"transcript schema version {TRANSCRIPT_SCHEMA_VERSION}" in out


def test_bare_invocation_shows_help(capsys: pytest.CaptureFixture[str]) -> None:
    assert main([]) == EXIT_OK
    assert "usage" in capsys.readouterr().out.lower()


@pytest.mark.parametrize("command", PENDING_COMMANDS)
def test_pending_commands_report_not_implemented(
    command: str, capsys: pytest.CaptureFixture[str]
) -> None:
    assert main([command]) == EXIT_NOT_IMPLEMENTED

    captured = capsys.readouterr()
    assert captured.out == "", "an unbuilt command must not write to the event stream"
    assert f"{PROGRAM} {command} is not available in this build" in captured.err


def test_pending_commands_share_one_message(capsys: pytest.CaptureFixture[str]) -> None:
    """The message must differ only by command name."""
    messages = set()
    for command in PENDING_COMMANDS:
        main([command])
        err = capsys.readouterr().err
        messages.add(err.replace(command, "<command>"))

    assert len(messages) == 1, f"placeholders disagree: {messages}"


def test_unknown_command_is_a_usage_error() -> None:
    with pytest.raises(SystemExit) as exit_info:
        main(["frobnicate"])

    assert exit_info.value.code == EXIT_USAGE


def test_transcribe_is_not_silently_available() -> None:
    """Guard against a future edit exposing transcription before it works."""
    assert "transcribe" in PENDING_COMMANDS
