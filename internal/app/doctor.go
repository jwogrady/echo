package app

import (
	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/diagnostics"
)

// newDoctorCommand builds `ekko doctor`, which reports what Echo can see.
//
// Exit status is deliberately forgiving by default: a user runs doctor to find
// out what is wrong, and a nonzero status from a diagnostic tool reads like the
// tool itself failed. --strict makes it a gate for scripts and CI, where a
// missing prerequisite should stop the run.
func newDoctorCommand(streams Streams, dispatched *dispatch) *cobra.Command {
	var strict bool

	command := &cobra.Command{
		Use:   "doctor",
		Short: "Check the environment Echo needs",
		Long: "Inspect everything Echo depends on and report what is present, what is\n" +
			"missing, and what to do about it.\n\n" +
			"Exits zero even when something is missing, so the report is always\n" +
			"readable. Use --strict to exit nonzero when a required dependency fails.",
		Args: cobra.NoArgs,
		RunE: dispatched.mark(func(_ *cobra.Command, _ []string) error {
			report := diagnostics.Inspect(diagnostics.HostEnvironment())
			report.Render(streams.Out)

			if strict {
				if blocking := report.Blocking(); len(blocking) > 0 {
					return &environmentError{blocking: blocking}
				}
			}

			return nil
		}),
	}

	command.Flags().BoolVar(&strict, "strict", false,
		"exit nonzero when a required dependency is missing or misconfigured")

	return command
}
