// Package app assembles Echo's command tree and turns its outcome into a
// process exit code.
//
// Commands defined here stay thin: they parse input, call a service, and
// present the result. Behavior belongs in the internal packages so it can be
// tested without running a binary.
package app

import (
	"errors"
	"fmt"
	"io"

	"github.com/jwogrady/echo/internal/buildinfo"
)

// Streams are the output destinations a command writes to. Tests pass buffers;
// main passes the real process streams. Machine-readable output goes to Out,
// human-readable diagnostics to Err.
type Streams struct {
	Out io.Writer
	Err io.Writer
}

// withDefaults substitutes io.Discard for any writer the caller omitted, so a
// partially populated Streams cannot panic mid-command.
func (s Streams) withDefaults() Streams {
	if s.Out == nil {
		s.Out = io.Discard
	}
	if s.Err == nil {
		s.Err = io.Discard
	}

	return s
}

// Run executes args (excluding the program name) against the command tree and
// reports the exit code the process should return. It never panics on a
// user-supplied argument, and it prints every failure exactly once.
func Run(args []string, streams Streams) ExitCode {
	streams = streams.withDefaults()

	root, dispatched := newRootCommand(streams)
	root.SetArgs(args)

	err := root.Execute()
	if err == nil {
		return ExitOK
	}

	// Failing before any command ran means the invocation itself was wrong.
	if !dispatched.ran {
		err = usageErrorf("%s", err)
	}

	report(err, streams.Err)

	return classify(err)
}

// report writes a single user-facing description of err.
func report(err error, w io.Writer) {
	// A not-implemented error already names its own command path, so it needs
	// no program-name prefix.
	var notImplemented *notImplementedError
	if errors.As(err, &notImplemented) {
		fmt.Fprintf(w, "%s\n\n%s\n", notImplemented.Error(), notImplemented.remediation())
		return
	}

	fmt.Fprintf(w, "%s: %s\n", buildinfo.Name, err)

	var usage *usageError
	if errors.As(err, &usage) {
		fmt.Fprintf(w, "\nRun %q to see the available commands.\n", buildinfo.Name+" help")
	}
}
