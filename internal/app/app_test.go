package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/buildinfo"
)

// run executes args against a fresh command tree and returns the exit code
// alongside whatever each stream received.
func run(t *testing.T, args ...string) (ExitCode, string, string) {
	t.Helper()

	var out, errOut bytes.Buffer
	code := Run(args, Streams{Out: &out, Err: &errOut})

	return code, out.String(), errOut.String()
}

// pendingCommandNames lists the placeholder command names declared in pending.
func pendingCommandNames() []string {
	names := make([]string, 0, len(pending))
	for _, cmd := range pending {
		name, _, _ := strings.Cut(cmd.use, " ")
		names = append(names, name)
	}

	return names
}

// validArgs supplies a well-formed invocation for each placeholder command, so
// a not-implemented test is not accidentally passing on an argument error.
var validArgs = map[string][]string{
	"doctor":     {},
	"new":        {"Product Strategy"},
	"list":       {},
	"use":        {"conv-1"},
	"status":     {},
	"add":        {"idea.wav"},
	"transcribe": {},
	"show":       {},
	"export":     {"markdown"},
}

func TestVersionSucceeds(t *testing.T) {
	code, out, errOut := run(t, "version")

	if code != ExitOK {
		t.Errorf("exit code = %d, want %d (stderr: %q)", code, ExitOK, errOut)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
	if !strings.HasPrefix(out, buildinfo.Name+" ") {
		t.Errorf("stdout = %q, want it to start with %q", out, buildinfo.Name+" ")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("stdout = %q, want a trailing newline", out)
	}
}

func TestVersionRejectsArguments(t *testing.T) {
	code, _, errOut := run(t, "version", "extra")

	if code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, buildinfo.Name+":") {
		t.Errorf("stderr = %q, want it to name the program", errOut)
	}
}

func TestPendingCommandsReportNotImplemented(t *testing.T) {
	for _, name := range pendingCommandNames() {
		t.Run(name, func(t *testing.T) {
			args, ok := validArgs[name]
			if !ok {
				t.Fatalf("no valid invocation recorded for %q; add one to validArgs", name)
			}

			code, out, errOut := run(t, append([]string{name}, args...)...)

			if code != ExitNotImplemented {
				t.Errorf("exit code = %d, want %d", code, ExitNotImplemented)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty: an unbuilt command must not fake output", out)
			}
			if !strings.Contains(errOut, buildinfo.Name+" "+name+" is not available in this build") {
				t.Errorf("stderr = %q, want it to name %q as unavailable", errOut, name)
			}
			if !strings.Contains(errOut, `Run "`+buildinfo.Name+` help"`) {
				t.Errorf("stderr = %q, want actionable remediation", errOut)
			}
		})
	}
}

// The criterion is a *consistent* message, so assert the placeholders differ
// only by command name.
func TestPendingCommandsShareOneMessage(t *testing.T) {
	var canonical string

	for _, name := range pendingCommandNames() {
		_, _, errOut := run(t, append([]string{name}, validArgs[name]...)...)

		normalized := strings.ReplaceAll(errOut, buildinfo.Name+" "+name, buildinfo.Name+" <command>")
		if canonical == "" {
			canonical = normalized
			continue
		}

		if normalized != canonical {
			t.Errorf("%q reports:\n%q\nwant the shared message:\n%q", name, normalized, canonical)
		}
	}
}

// Argument validation runs before the not-implemented report, so a malformed
// invocation of an unbuilt command is still a usage error.
func TestPendingCommandsValidateArgumentsFirst(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"missing required argument", []string{"new"}},
		{"unexpected argument", []string{"list", "surplus"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, _, errOut := run(t, test.args...)

			if code != ExitUsage {
				t.Errorf("exit code = %d, want %d (stderr: %q)", code, ExitUsage, errOut)
			}
			if strings.Contains(errOut, "not available in this build") {
				t.Errorf("stderr = %q, want an argument error rather than the not-implemented report", errOut)
			}
		})
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	code, _, errOut := run(t, "transcribbe")

	if code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, `unknown command "transcribbe"`) {
		t.Errorf("stderr = %q, want it to quote the unknown command", errOut)
	}
	if !strings.Contains(errOut, `Run "`+buildinfo.Name+` help"`) {
		t.Errorf("stderr = %q, want it to point at help", errOut)
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	code, _, errOut := run(t, "version", "--nonsense")

	if code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "nonsense") {
		t.Errorf("stderr = %q, want it to name the offending flag", errOut)
	}
}

func TestNoArgumentsShowsHelp(t *testing.T) {
	code, out, errOut := run(t)

	if code != ExitOK {
		t.Errorf("exit code = %d, want %d (stderr: %q)", code, ExitOK, errOut)
	}
	if !strings.Contains(out, "Available Commands:") {
		t.Errorf("stdout = %q, want the command listing", out)
	}
}

// Every command in the v0.1 surface must be reachable, so no user meets
// "unknown command" for something Echo intends to support.
func TestEveryV01CommandIsRegistered(t *testing.T) {
	surface := []string{
		"version", "doctor", "new", "list", "use",
		"status", "add", "transcribe", "show", "export",
	}

	registered := map[string]bool{}
	for _, cmd := range NewRootCommand(Streams{}).Commands() {
		registered[cmd.Name()] = true
	}

	for _, name := range surface {
		if !registered[name] {
			t.Errorf("command %q is not registered", name)
		}
	}
}

// The command surface is the plan's, not Cobra's defaults.
func TestCompletionCommandIsNotOffered(t *testing.T) {
	for _, cmd := range NewRootCommand(Streams{}).Commands() {
		if cmd.Name() == "completion" {
			t.Error("completion is registered; it is not part of the v0.1 surface")
		}
	}
}

// The prefix would otherwise read "ekko: ekko transcribe is not available".
func TestNotImplementedMessageNamesTheCommandOnce(t *testing.T) {
	_, _, errOut := run(t, "transcribe")

	if strings.HasPrefix(errOut, buildinfo.Name+": "+buildinfo.Name+" ") {
		t.Errorf("stderr = %q, want the command named once", errOut)
	}
	if !strings.HasPrefix(errOut, buildinfo.Name+" transcribe is not available") {
		t.Errorf("stderr = %q, want it to lead with the command path", errOut)
	}
}

// A subcommand defining its own PersistentPreRun used to suppress a root-level
// one, which would misreport every runtime error as a usage error. Dispatch is
// now marked at each RunE, so a child hook cannot break the classification.
func TestChildHookDoesNotBreakErrorClassification(t *testing.T) {
	root, dispatched := newRootCommand(Streams{})

	var target *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "transcribe" {
			target = cmd
			break
		}
	}
	if target == nil {
		t.Fatal("transcribe command not found")
	}
	target.PersistentPreRun = func(*cobra.Command, []string) {}

	root.SetArgs([]string{"transcribe"})
	err := root.Execute()

	if err == nil {
		t.Fatal("expected the not-implemented error")
	}
	if !dispatched.ran {
		t.Error("dispatch was not marked; a child hook suppressed it")
	}
	if got := classify(err); got != ExitNotImplemented {
		t.Errorf("classify = %d, want %d: a runtime error was misread as usage", got, ExitNotImplemented)
	}
}

// Run must not panic when a caller supplies an incomplete Streams.
func TestRunToleratesMissingWriters(t *testing.T) {
	if got := Run([]string{"version"}, Streams{}); got != ExitOK {
		t.Errorf("exit code = %d, want %d", got, ExitOK)
	}
	if got := Run([]string{"transcribe"}, Streams{}); got != ExitNotImplemented {
		t.Errorf("exit code = %d, want %d", got, ExitNotImplemented)
	}
}

// doctor is implemented now, so it must not be reachable as a placeholder.
func TestDoctorIsNoLongerPending(t *testing.T) {
	for _, name := range pendingCommandNames() {
		if name == "doctor" {
			t.Error("doctor is still registered as a placeholder")
		}
	}
}

// A diagnostic tool that exits nonzero reads as though the tool itself broke, so
// doctor reports and succeeds by default.
func TestDoctorSucceedsEvenWhenTheMachineIsNotReady(t *testing.T) {
	code, out, _ := run(t, "doctor")

	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}
	for _, section := range []string{"Environment", "Required"} {
		if !strings.Contains(out, section) {
			t.Errorf("output missing the %q section:\n%s", section, out)
		}
	}
}

// --strict is the scriptable gate. This asserts the wiring, not a verdict: the
// exit code depends on what the host machine actually has installed.
func TestDoctorStrictReportsAVerdict(t *testing.T) {
	code, out, errOut := run(t, "doctor", "--strict")

	switch code {
	case ExitOK:
		if !strings.Contains(out, "Ready") {
			t.Errorf("strict succeeded but the report does not say Ready:\n%s", out)
		}
	case ExitError:
		if !strings.Contains(errOut, "required") {
			t.Errorf("strict failed without naming the required dependencies: %q", errOut)
		}
	default:
		t.Errorf("exit code = %d, want %d or %d", code, ExitOK, ExitError)
	}
}

func TestDoctorRejectsArguments(t *testing.T) {
	if code, _, _ := run(t, "doctor", "surplus"); code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
}
