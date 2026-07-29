package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jwogrady/echo/internal/buildinfo"
	"github.com/jwogrady/echo/internal/diagnostics"
)

// ExitCode is the process status Echo returns to the shell. Windows callers
// branch on these, so the meaning of each value is part of Echo's contract.
type ExitCode int

const (
	// ExitOK means the command did what was asked.
	ExitOK ExitCode = 0
	// ExitError means the command failed for a reason of its own.
	ExitError ExitCode = 1
	// ExitUsage means the invocation was wrong: an unknown command, a bad
	// flag, or missing arguments. Nothing was attempted.
	ExitUsage ExitCode = 2
	// ExitNotImplemented means the command is part of Echo v0.1 but is not
	// available in this build.
	ExitNotImplemented ExitCode = 3
)

// usageError marks a failure caused by how Echo was invoked rather than by
// anything it tried to do.
type usageError struct {
	msg string
}

func (e *usageError) Error() string { return e.msg }

func usageErrorf(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// notImplementedError marks a command that is planned for v0.1 but not built
// in this binary. Every placeholder fails through this one type so the message
// and exit code cannot drift between commands.
type notImplementedError struct {
	// command is the full invocation path, such as "echo export".
	command string
}

func (e *notImplementedError) Error() string {
	return fmt.Sprintf("%s is not available in this build", e.command)
}

// remediation is the actionable half of the message: what the user can do now.
func (e *notImplementedError) remediation() string {
	return "Echo v0.1 is being built one capability at a time and this command is\n" +
		"not ready yet. Run \"" + buildinfo.Name + " help\" to see what this build supports."
}

// classify maps an error returned by the command tree onto an exit code.
func classify(err error) ExitCode {
	var notImplemented *notImplementedError
	if errors.As(err, &notImplemented) {
		return ExitNotImplemented
	}

	var usage *usageError
	if errors.As(err, &usage) {
		return ExitUsage
	}

	return ExitError
}

// environmentError means required dependencies are missing or misconfigured. It
// exists so `doctor --strict` fails as a runtime condition rather than a usage
// error: the invocation was correct, the machine is not ready.
type environmentError struct {
	blocking []diagnostics.Check
}

func (e *environmentError) Error() string {
	names := make([]string, 0, len(e.blocking))
	for _, check := range e.blocking {
		names = append(names, check.Name)
	}

	noun := "dependencies are"
	if len(e.blocking) == 1 {
		noun = "dependency is"
	}

	return fmt.Sprintf("%d required %s unavailable: %s",
		len(e.blocking), noun, strings.Join(names, ", "))
}

// reportedError wraps a failure whose full explanation has already been written
// to the user. It still sets a nonzero exit code, but the central reporter stays
// quiet so the same cause is not printed twice.
type reportedError struct {
	cause error
}

func (e *reportedError) Error() string { return e.cause.Error() }

func (e *reportedError) Unwrap() error { return e.cause }
