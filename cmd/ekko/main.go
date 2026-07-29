// Command ekko turns WAV recordings into timestamped transcripts locally.
//
// The command is "ekko" rather than "echo" because that name is a shell builtin
// in zsh and bash and an alias for Write-Output in PowerShell; see
// docs/plan/decisions/ADR-0002-command-name.md.
//
// This entrypoint stays deliberately thin: everything testable lives in
// internal/app.
package main

import (
	"os"

	"github.com/jwogrady/echo/internal/app"
)

func main() {
	code := app.Run(os.Args[1:], app.Streams{
		Out: os.Stdout,
		Err: os.Stderr,
	})

	os.Exit(int(code))
}
