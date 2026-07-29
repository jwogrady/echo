package app

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/buildinfo"
	"github.com/jwogrady/echo/internal/config"
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

// withTempDataDir points Echo at a throwaway data root so command tests never
// touch a developer's real conversations.
func withTempDataDir(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	t.Setenv(config.EnvDataDir, root)

	return root
}

func TestNewAndListRoundTrip(t *testing.T) {
	withTempDataDir(t)

	code, out, errOut := run(t, "new", "Product Strategy")
	if code != ExitOK {
		t.Fatalf("new exit = %d, want %d (stderr: %q)", code, ExitOK, errOut)
	}
	if !strings.Contains(out, "Created cnv_") {
		t.Errorf("new output = %q, want it to report the created id", out)
	}

	code, out, errOut = run(t, "list")
	if code != ExitOK {
		t.Fatalf("list exit = %d, want %d (stderr: %q)", code, ExitOK, errOut)
	}
	for _, want := range []string{"ID", "TITLE", "STATUS", "UPDATED", "Product Strategy", "created"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q:\n%s", want, out)
		}
	}
}

// An empty list is a normal state, not an error.
func TestListOnAnEmptyDataRootSucceeds(t *testing.T) {
	withTempDataDir(t)

	code, out, _ := run(t, "list")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out, "No conversations yet") {
		t.Errorf("output = %q, want a friendly empty state", out)
	}
	if !strings.Contains(out, buildinfo.Name+" new") {
		t.Errorf("output = %q, want it to suggest the real command name", out)
	}
}

func TestNewRequiresATitle(t *testing.T) {
	withTempDataDir(t)

	if code, _, _ := run(t, "new"); code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if code, _, _ := run(t, "new", "   "); code != ExitError {
		t.Errorf("blank title exit = %d, want %d", code, ExitError)
	}
}

func TestNewAndListAreNoLongerPending(t *testing.T) {
	for _, name := range pendingCommandNames() {
		if name == "new" || name == "list" {
			t.Errorf("%s is still registered as a placeholder", name)
		}
	}
}

// Deterministic output is what makes list safe to script against.
func TestListOutputIsStableAcrossRuns(t *testing.T) {
	withTempDataDir(t)

	for _, title := range []string{"One", "Two", "Three"} {
		if code, _, _ := run(t, "new", title); code != ExitOK {
			t.Fatalf("new %q failed", title)
		}
	}

	_, first, _ := run(t, "list")
	_, second, _ := run(t, "list")

	if first != second {
		t.Errorf("list output varies between runs:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// createConversation makes one via the CLI and returns its id.
func createConversation(t *testing.T, title string) string {
	t.Helper()

	code, out, errOut := run(t, "new", title)
	if code != ExitOK {
		t.Fatalf("new %q exit = %d (stderr: %q)", title, code, errOut)
	}

	fields := strings.Fields(strings.SplitN(out, "\n", 2)[0])
	if len(fields) < 2 {
		t.Fatalf("cannot read the id from %q", out)
	}

	return fields[1]
}

func TestUseSelectsByFullID(t *testing.T) {
	withTempDataDir(t)
	id := createConversation(t, "Alpha")

	code, out, errOut := run(t, "use", id)
	if code != ExitOK {
		t.Fatalf("use exit = %d (stderr: %q)", code, errOut)
	}
	if !strings.Contains(out, "Using "+id) {
		t.Errorf("output = %q, want it to confirm %q", out, id)
	}
	if !strings.Contains(out, "Alpha") {
		t.Errorf("output = %q, want the title", out)
	}
}

func TestUseSelectsByUniquePrefix(t *testing.T) {
	withTempDataDir(t)
	id := createConversation(t, "Alpha")

	// Six characters of the body, lowercased and without the cnv_ prefix.
	prefix := strings.ToLower(strings.TrimPrefix(id, "cnv_")[:6])

	code, out, errOut := run(t, "use", prefix)
	if code != ExitOK {
		t.Fatalf("use %q exit = %d (stderr: %q)", prefix, code, errOut)
	}
	if !strings.Contains(out, "Using "+id) {
		t.Errorf("output = %q, want it to resolve to %q", out, id)
	}
}

func TestUseReportsNoMatchAsARuntimeError(t *testing.T) {
	withTempDataDir(t)
	createConversation(t, "Alpha")

	code, _, errOut := run(t, "use", "ZZZZZZ")
	if code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut, "no conversation matches") {
		t.Errorf("stderr = %q", errOut)
	}
}

// A blank argument is a malformed invocation, not a failed lookup.
func TestUseTreatsABlankArgumentAsUsage(t *testing.T) {
	withTempDataDir(t)

	code, _, errOut := run(t, "use", "   ")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if strings.Contains(errOut, "no conversation matches") {
		t.Errorf("stderr = %q, want a usage message rather than a lookup failure", errOut)
	}
}

func TestUseRequiresAnArgument(t *testing.T) {
	withTempDataDir(t)

	if code, _, _ := run(t, "use"); code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestUseIsNoLongerPending(t *testing.T) {
	for _, name := range pendingCommandNames() {
		if name == "use" {
			t.Error("use is still registered as a placeholder")
		}
	}
}

// The flag must exist on every command, so automation never depends on what a
// human selected.
func TestConversationFlagIsGlobal(t *testing.T) {
	root := NewRootCommand(Streams{})

	if root.PersistentFlags().Lookup("conversation") == nil {
		t.Fatal("--conversation is not registered as a persistent flag")
	}

	for _, command := range root.Commands() {
		if command.Flags().Lookup("conversation") == nil && command.InheritedFlags().Lookup("conversation") == nil {
			t.Errorf("%s does not see --conversation", command.Name())
		}
	}
}

// The flag overrides the persisted selection without changing it.
func TestConversationFlagOverridesWithoutPersisting(t *testing.T) {
	withTempDataDir(t)

	selected := createConversation(t, "Selected")
	other := createConversation(t, "Other")

	if code, _, errOut := run(t, "use", selected); code != ExitOK {
		t.Fatalf("use exit = %d (stderr: %q)", code, errOut)
	}

	repo, err := repository()
	if err != nil {
		t.Fatalf("repository() error = %v", err)
	}

	flag := &conversationFlag{value: other}
	target, err := flag.target(repo)
	if err != nil {
		t.Fatalf("target() error = %v", err)
	}
	if target != other {
		t.Errorf("target() = %q, want the flag value %q", target, other)
	}

	active, err := repo.ActiveID()
	if err != nil {
		t.Fatalf("ActiveID() error = %v", err)
	}
	if active != selected {
		t.Errorf("the flag changed the persisted selection to %q, want %q", active, selected)
	}
}

func TestTargetFallsBackToTheSelection(t *testing.T) {
	withTempDataDir(t)
	id := createConversation(t, "Alpha")

	if code, _, _ := run(t, "use", id); code != ExitOK {
		t.Fatal("use failed")
	}

	repo, err := repository()
	if err != nil {
		t.Fatalf("repository() error = %v", err)
	}

	target, err := (&conversationFlag{}).target(repo)
	if err != nil {
		t.Fatalf("target() error = %v", err)
	}
	if target != id {
		t.Errorf("target() = %q, want %q", target, id)
	}
}

// With nothing selected and no flag, the error must say how to proceed.
func TestTargetWithoutASelectionIsActionable(t *testing.T) {
	withTempDataDir(t)

	repo, err := repository()
	if err != nil {
		t.Fatalf("repository() error = %v", err)
	}

	_, err = (&conversationFlag{}).target(repo)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), buildinfo.Name+" use") {
		t.Errorf("error = %q, want it to suggest the use command", err)
	}
	if !strings.Contains(err.Error(), "--conversation") {
		t.Errorf("error = %q, want it to mention the flag", err)
	}
}

// damageMetadata overwrites a conversation's metadata and returns what it wrote,
// so a test can prove Echo left it alone.
func damageMetadata(t *testing.T, root, id, contents string) string {
	t.Helper()

	path := filepath.Join(root, "conversations", id, "conversation.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing the damaged fixture: %v", err)
	}

	return path
}

func TestStatusReportsTheActiveConversation(t *testing.T) {
	withTempDataDir(t)
	id := createConversation(t, "Product Strategy")

	if code, _, _ := run(t, "use", id); code != ExitOK {
		t.Fatal("use failed")
	}

	code, out, errOut := run(t, "status")
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr: %q)", code, errOut)
	}

	for _, want := range []string{id, "Product Strategy", "created", "path"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// A blank field would read as "ready"; absence must be stated.
func TestStatusStatesWhatDoesNotExistYet(t *testing.T) {
	withTempDataDir(t)
	id := createConversation(t, "Alpha")
	if code, _, _ := run(t, "use", id); code != ExitOK {
		t.Fatal("use failed")
	}

	_, out, _ := run(t, "status")

	for _, want := range []string{"none imported yet", "none yet"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// status must not advertise a command that would fail.
func TestStatusMarksUnbuiltNextStepsAsUnavailable(t *testing.T) {
	withTempDataDir(t)
	id := createConversation(t, "Alpha")
	if code, _, _ := run(t, "use", id); code != ExitOK {
		t.Fatal("use failed")
	}

	_, out, _ := run(t, "status")

	if !strings.Contains(out, "not available in this build") {
		t.Errorf("output should mark the next step unavailable:\n%s", out)
	}
}

func TestStatusHonorsTheConversationFlag(t *testing.T) {
	withTempDataDir(t)
	selected := createConversation(t, "Selected")
	other := createConversation(t, "Other")

	if code, _, _ := run(t, "use", selected); code != ExitOK {
		t.Fatal("use failed")
	}

	code, out, errOut := run(t, "status", "--conversation", other)
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr: %q)", code, errOut)
	}
	if !strings.Contains(out, other) {
		t.Errorf("output should describe the flagged conversation:\n%s", out)
	}
	if strings.Contains(out, selected) {
		t.Errorf("output describes the selected conversation instead:\n%s", out)
	}
}

func TestStatusWithoutASelectionIsActionable(t *testing.T) {
	withTempDataDir(t)

	code, _, errOut := run(t, "status")
	if code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut, buildinfo.Name+" use") {
		t.Errorf("stderr = %q, want it to suggest use", errOut)
	}
}

// Damage is reported in full, the file is untouched, and the exit code is nonzero.
func TestStatusReportsDamageWithoutTouchingTheFile(t *testing.T) {
	tests := []struct {
		name         string
		contents     string
		wantGuidance string
	}{
		{
			name:         "truncated json",
			contents:     `{"schema_version":1,"id":"trunc`,
			wantGuidance: "will not repair",
		},
		{
			name:         "newer schema",
			contents:     `{"schema_version":99,"id":"cnv_AAAAAAAAAAAAAAAAAAAAAAAAAA","title":"x","status":"created","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`,
			wantGuidance: "newer version of Echo",
		},
		{
			name:         "impossible status",
			contents:     `{"schema_version":1,"id":"cnv_AAAAAAAAAAAAAAAAAAAAAAAAAA","title":"x","status":"halfway","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`,
			wantGuidance: "will not repair",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := withTempDataDir(t)
			id := createConversation(t, "Doomed")
			if code, _, _ := run(t, "use", id); code != ExitOK {
				t.Fatal("use failed")
			}

			path := damageMetadata(t, root, id, test.contents)

			code, out, errOut := run(t, "status")

			if code != ExitError {
				t.Errorf("exit = %d, want %d", code, ExitError)
			}
			if !strings.Contains(out, "unreadable") {
				t.Errorf("output should say unreadable:\n%s", out)
			}
			if !strings.Contains(out, path) {
				t.Errorf("output should name the metadata path:\n%s", out)
			}
			if !strings.Contains(out, test.wantGuidance) {
				t.Errorf("output missing guidance %q:\n%s", test.wantGuidance, out)
			}
			// The cause is explained once, in the report, not twice.
			if errOut != "" {
				t.Errorf("stderr should be empty when the report already explained it, got %q", errOut)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading back: %v", err)
			}
			if string(after) != test.contents {
				t.Errorf("status modified the damaged file:\n got %q\nwant %q", after, test.contents)
			}
		})
	}
}

// Audio is the part a user cannot regenerate, so say it survived.
func TestStatusReportsSurvivingWorkspaceContents(t *testing.T) {
	root := withTempDataDir(t)
	id := createConversation(t, "Doomed")
	if code, _, _ := run(t, "use", id); code != ExitOK {
		t.Fatal("use failed")
	}

	audio := filepath.Join(root, "conversations", id, "audio", "source.wav")
	if err := os.WriteFile(audio, []byte("not really audio"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	damageMetadata(t, root, id, "{broken")

	_, out, _ := run(t, "status")

	if !strings.Contains(out, "Still present") || !strings.Contains(out, "audio/") {
		t.Errorf("output should report the surviving audio:\n%s", out)
	}
}

// A missing metadata file needs different advice than a corrupt one.
func TestStatusDistinguishesMissingMetadata(t *testing.T) {
	root := withTempDataDir(t)
	id := createConversation(t, "Doomed")
	if code, _, _ := run(t, "use", id); code != ExitOK {
		t.Fatal("use failed")
	}

	if err := os.Remove(filepath.Join(root, "conversations", id, "conversation.json")); err != nil {
		t.Fatalf("removing the metadata: %v", err)
	}

	code, out, _ := run(t, "status")
	if code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
	if !strings.Contains(out, "no metadata file") {
		t.Errorf("output should distinguish a missing file:\n%s", out)
	}
}

// list must survive damage too, showing the healthy rows.
func TestListSurvivesADamagedConversation(t *testing.T) {
	root := withTempDataDir(t)
	healthy := createConversation(t, "Healthy")
	damaged := createConversation(t, "Damaged")

	damageMetadata(t, root, damaged, "{broken")

	code, out, errOut := run(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr: %q)", code, errOut)
	}
	if !strings.Contains(out, healthy) || !strings.Contains(out, "Healthy") {
		t.Errorf("the healthy conversation is missing:\n%s", out)
	}
	if !strings.Contains(out, damaged) || !strings.Contains(out, "(unreadable)") {
		t.Errorf("the damaged conversation should be flagged, not hidden:\n%s", out)
	}
}

func TestStatusIsNoLongerPending(t *testing.T) {
	for _, name := range pendingCommandNames() {
		if name == "status" {
			t.Error("status is still registered as a placeholder")
		}
	}
}

func TestStatusRejectsArguments(t *testing.T) {
	withTempDataDir(t)

	if code, _, _ := run(t, "status", "surplus"); code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}

// wavFixture writes a real RIFF/WAVE file for command-level tests.
func wavFixture(t *testing.T, name string, samples int) string {
	t.Helper()

	payload := make([]byte, samples*2)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+len(payload)))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], 1)
	binary.LittleEndian.PutUint32(header[24:28], 16000)
	binary.LittleEndian.PutUint32(header[28:32], 32000)
	binary.LittleEndian.PutUint16(header[32:34], 2)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(len(payload)))

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, append(header, payload...), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	return path
}

// selectedConversation creates a conversation and makes it active.
func selectedConversation(t *testing.T, title string) string {
	t.Helper()

	id := createConversation(t, title)
	if code, _, errOut := run(t, "use", id); code != ExitOK {
		t.Fatalf("use failed: %s", errOut)
	}

	return id
}

func TestAddImportsARecording(t *testing.T) {
	withTempDataDir(t)
	selectedConversation(t, "Audio Test")
	fixture := wavFixture(t, "recording.wav", 800)

	code, out, errOut := run(t, "add", fixture)
	if code != ExitOK {
		t.Fatalf("exit = %d (stderr: %q)", code, errOut)
	}
	for _, want := range []string{"Imported recording.wav", "rec_", "sha256"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// Progress belongs on stderr so stdout stays parseable.
func TestAddReportsProgressOnStderr(t *testing.T) {
	withTempDataDir(t)
	selectedConversation(t, "Audio Test")
	fixture := wavFixture(t, "recording.wav", 800)

	_, out, errOut := run(t, "add", fixture)

	if !strings.Contains(errOut, "hashing source") {
		t.Errorf("stderr should carry stage progress:\n%s", errOut)
	}
	if strings.Contains(out, "hashing source") {
		t.Errorf("stdout should not carry progress:\n%s", out)
	}
}

func TestAddAdvancesTheConversationStatus(t *testing.T) {
	withTempDataDir(t)
	selectedConversation(t, "Audio Test")
	fixture := wavFixture(t, "recording.wav", 800)

	if code, _, errOut := run(t, "add", fixture); code != ExitOK {
		t.Fatalf("add failed: %s", errOut)
	}

	_, out, _ := run(t, "status")
	if !strings.Contains(out, "audio_ready") {
		t.Errorf("status should advance to audio_ready:\n%s", out)
	}
	if !strings.Contains(out, "rec_") {
		t.Errorf("status should name the recording:\n%s", out)
	}
}

func TestAddRejectsNonWAVInput(t *testing.T) {
	withTempDataDir(t)
	selectedConversation(t, "Audio Test")

	fake := filepath.Join(t.TempDir(), "fake.wav")
	if err := os.WriteFile(fake, []byte("ID3\x04\x00\x00\x00\x00\x00\x00junk"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	code, _, errOut := run(t, "add", fake)
	if code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut, "not a WAV file") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestAddIsIdempotent(t *testing.T) {
	withTempDataDir(t)
	selectedConversation(t, "Audio Test")
	fixture := wavFixture(t, "recording.wav", 800)

	if code, _, _ := run(t, "add", fixture); code != ExitOK {
		t.Fatal("first add failed")
	}

	code, out, _ := run(t, "add", fixture)
	if code != ExitOK {
		t.Errorf("re-adding the same file should succeed, got exit %d", code)
	}
	if !strings.Contains(out, "Already imported") {
		t.Errorf("output should say it was already imported:\n%s", out)
	}
}

func TestAddRefusesADifferentFileWithoutReplace(t *testing.T) {
	withTempDataDir(t)
	selectedConversation(t, "Audio Test")

	first := wavFixture(t, "first.wav", 800)
	second := wavFixture(t, "second.wav", 1600)

	if code, _, _ := run(t, "add", first); code != ExitOK {
		t.Fatal("first add failed")
	}

	code, _, errOut := run(t, "add", second)
	if code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut, "--replace") {
		t.Errorf("stderr should name --replace: %q", errOut)
	}

	code, out, _ := run(t, "add", second, "--replace")
	if code != ExitOK {
		t.Errorf("--replace exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out, "second.wav") {
		t.Errorf("output should confirm the replacement:\n%s", out)
	}
}

func TestAddRequiresASelectedConversation(t *testing.T) {
	withTempDataDir(t)
	fixture := wavFixture(t, "recording.wav", 800)

	code, _, errOut := run(t, "add", fixture)
	if code != ExitError {
		t.Errorf("exit = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut, buildinfo.Name+" use") {
		t.Errorf("stderr should suggest selecting a conversation: %q", errOut)
	}
}

func TestAddHonorsTheConversationFlag(t *testing.T) {
	withTempDataDir(t)
	selected := selectedConversation(t, "Selected")
	other := createConversation(t, "Other")
	fixture := wavFixture(t, "recording.wav", 800)

	if code, _, errOut := run(t, "add", fixture, "--conversation", other); code != ExitOK {
		t.Fatalf("add failed: %s", errOut)
	}

	_, out, _ := run(t, "status", "--conversation", other)
	if !strings.Contains(out, "audio_ready") {
		t.Errorf("the flagged conversation should hold the recording:\n%s", out)
	}

	_, out, _ = run(t, "status", "--conversation", selected)
	if strings.Contains(out, "audio_ready") {
		t.Errorf("the selected conversation should be untouched:\n%s", out)
	}
}

func TestAddIsNoLongerPending(t *testing.T) {
	for _, name := range pendingCommandNames() {
		if name == "add" {
			t.Error("add is still registered as a placeholder")
		}
	}
}
