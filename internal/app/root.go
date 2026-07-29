package app

import (
	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/buildinfo"
)

// pending names every command in Echo's v0.1 surface that this build does not
// implement yet, in the order it should appear in help. Registering them keeps
// the whole surface reachable: an unbuilt command explains itself instead of
// looking like a typo. Each is replaced by its real implementation as that
// capability lands.
var pending = []struct {
	use   string
	args  cobra.PositionalArgs
	short string
}{
	{"new <title>", cobra.ExactArgs(1), "Create a conversation"},
	{"list", cobra.NoArgs, "List conversations"},
	{"use <conversation-id>", cobra.ExactArgs(1), "Select the active conversation"},
	{"status", cobra.NoArgs, "Show the active conversation's state"},
	{"add <wav-path>", cobra.ExactArgs(1), "Import a WAV recording"},
	{"transcribe", cobra.NoArgs, "Transcribe the active recording"},
	{"show", cobra.NoArgs, "Display the transcript"},
	{"export <format>", cobra.ExactArgs(1), "Export the transcript"},
}

// dispatch records whether Cobra got as far as running a command. Anything that
// fails before that point — an unknown command, an unparseable flag, the wrong
// number of arguments — is a usage error, and tracking it here means Echo never
// has to match on Cobra's message text to say so.
//
// Every command marks this through mark() rather than a PersistentPreRun hook:
// Cobra runs only the closest such hook in the chain unless run-hook traversal
// is enabled, so the first subcommand to define its own would silently disable a
// root-level one and make runtime errors look like usage errors.
type dispatch struct {
	ran bool
}

// mark wraps run so that reaching it records a successful dispatch.
func (d *dispatch) mark(run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		d.ran = true
		return run(cmd, args)
	}
}

// NewRootCommand builds the `ekko` command tree writing to streams.
func NewRootCommand(streams Streams) *cobra.Command {
	root, _ := newRootCommand(streams)
	return root
}

func newRootCommand(streams Streams) (*cobra.Command, *dispatch) {
	dispatched := &dispatch{}

	root := &cobra.Command{
		Use:   buildinfo.Name,
		Short: "Turn WAV recordings into timestamped transcripts, locally",
		Long: "Echo turns WAV recordings into structured, timestamped transcripts using\n" +
			"a local NVIDIA GPU. Audio never leaves the machine.",

		// Run and Execute report failures; app.report prints them once, so
		// Cobra must not also print them or append usage to every error.
		SilenceErrors: true,
		SilenceUsage:  true,

		// A bare invocation is not an error; it introduces the tool.
		RunE: dispatched.mark(func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		}),
	}

	// Keep the surface to the commands the plan defines; shell completion is
	// not part of v0.1.
	root.CompletionOptions.DisableDefaultCmd = true

	root.SetOut(streams.Out)
	root.SetErr(streams.Err)

	// A malformed flag is a usage error, not a runtime one.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageErrorf("%s", err)
	})

	root.AddCommand(newVersionCommand(streams, dispatched))
	root.AddCommand(newDoctorCommand(streams, dispatched))

	for _, cmd := range pending {
		root.AddCommand(newPendingCommand(cmd.use, cmd.short, cmd.args, dispatched))
	}

	return root, dispatched
}
