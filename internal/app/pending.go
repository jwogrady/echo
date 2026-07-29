package app

import (
	"github.com/spf13/cobra"
)

// newPendingCommand builds a placeholder for a v0.1 command this build does not
// implement. It validates its arguments like the real command will, then fails
// through notImplementedError so every placeholder reports identically.
//
// use is the Cobra use string, so "new <title>" yields the command "new".
func newPendingCommand(use, short string, args cobra.PositionalArgs, dispatched *dispatch) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  args,
		RunE: dispatched.mark(func(cmd *cobra.Command, _ []string) error {
			return &notImplementedError{command: cmd.CommandPath()}
		}),
	}
}
