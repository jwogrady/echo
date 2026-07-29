package app

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/buildinfo"
)

// newVersionCommand builds `ekko version`, which reports the running build.
func newVersionCommand(streams Streams, dispatched *dispatch) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Report the Echo version and build details",
		Args:  cobra.NoArgs,
		RunE: dispatched.mark(func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(streams.Out, buildinfo.Current())
			return err
		}),
	}
}
