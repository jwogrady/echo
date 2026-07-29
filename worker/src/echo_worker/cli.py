"""Command-line entrypoint for the Echo GPU worker.

This build implements ``version`` only. Transcription and environment
diagnostics arrive with the inference adapter; until then the worker must
install and run with no faster-whisper, CUDA, or cuDNN present.

Exit codes mirror the Go CLI's contract so both sides of the subprocess
boundary mean the same thing by a status:

    0  success
    1  runtime failure
    2  usage error
    3  not implemented
"""

from __future__ import annotations

import argparse
import sys
from collections.abc import Sequence
from typing import TextIO

from echo_worker import __version__
from echo_worker.contract import CONTRACT_VERSION, TRANSCRIPT_SCHEMA_VERSION

EXIT_OK = 0
EXIT_ERROR = 1
EXIT_USAGE = 2
EXIT_NOT_IMPLEMENTED = 3

PROGRAM = "echo-worker"

#: Commands named by the worker interface but not built in this version.
PENDING_COMMANDS = ("doctor", "transcribe")


class _Parser(argparse.ArgumentParser):
    """An ArgumentParser that exits with the shared usage code."""

    def error(self, message: str) -> None:  # pragma: no cover - argparse hook
        self.print_usage(sys.stderr)
        print(f"{PROGRAM}: {message}", file=sys.stderr)
        raise SystemExit(EXIT_USAGE)


def build_parser() -> argparse.ArgumentParser:
    """Construct the argument parser for the whole command surface."""
    parser = _Parser(
        prog=PROGRAM,
        description="Transcribe an optimized WAV on a local NVIDIA GPU.",
    )
    subcommands = parser.add_subparsers(dest="command", metavar="<command>")

    subcommands.add_parser("version", help="Report worker and contract versions")

    for name in PENDING_COMMANDS:
        subcommands.add_parser(name, help=f"{name} (not available in this build)")

    return parser


def write_version(out: TextIO) -> None:
    """Report the worker version and the contract versions it speaks."""
    print(f"{PROGRAM} {__version__}", file=out)
    print(f"contract version {CONTRACT_VERSION}", file=out)
    print(f"transcript schema version {TRANSCRIPT_SCHEMA_VERSION}", file=out)


def main(argv: Sequence[str] | None = None) -> int:
    """Run the worker and return the process exit code."""
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.command is None:
        parser.print_help()
        return EXIT_OK

    if args.command == "version":
        write_version(sys.stdout)
        return EXIT_OK

    if args.command in PENDING_COMMANDS:
        print(
            f"{PROGRAM} {args.command} is not available in this build",
            file=sys.stderr,
        )
        print(
            "\nThe Echo worker is being built one capability at a time and this\n"
            f'command is not ready yet. Run "{PROGRAM} version" to see what this\n'
            "build supports.",
            file=sys.stderr,
        )
        return EXIT_NOT_IMPLEMENTED

    # Unreachable while every declared subcommand is handled above.
    print(f"{PROGRAM}: unknown command {args.command!r}", file=sys.stderr)
    return EXIT_USAGE


if __name__ == "__main__":
    raise SystemExit(main())
