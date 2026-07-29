package app

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/config"
	"github.com/jwogrady/echo/internal/conversation"
)

// repository opens the conversation repository for the resolved data root.
func repository() (*conversation.Repository, error) {
	paths, err := config.Resolve()
	if err != nil {
		return nil, err
	}

	return conversation.NewRepository(paths), nil
}

// newNewCommand builds `ekko new <title>`.
func newNewCommand(streams Streams, dispatched *dispatch) *cobra.Command {
	return &cobra.Command{
		Use:   "new <title>",
		Short: "Create a conversation",
		Long: "Create a conversation workspace.\n\n" +
			"Titles are labels, not identifiers: creating two conversations with the\n" +
			"same title gives you two conversations with different ids.",
		Args: cobra.ExactArgs(1),
		RunE: dispatched.mark(func(_ *cobra.Command, args []string) error {
			repo, err := repository()
			if err != nil {
				return err
			}

			created, err := repo.Create(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(streams.Out, "Created %s\n", created.ID)
			fmt.Fprintf(streams.Out, "  title   %s\n", created.Title)
			fmt.Fprintf(streams.Out, "  status  %s\n", created.Status)
			fmt.Fprintf(streams.Out, "  path    %s\n", repo.Workspace(created.ID).Dir)

			return nil
		}),
	}
}

// newListCommand builds `ekko list`.
func newListCommand(streams Streams, dispatched *dispatch) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List conversations",
		Long: "List every conversation, most recently updated first.\n\n" +
			"A conversation whose metadata cannot be read is reported as unreadable\n" +
			"rather than omitted, so a damaged workspace is visible instead of silently\n" +
			"disappearing.",
		Args: cobra.NoArgs,
		RunE: dispatched.mark(func(_ *cobra.Command, _ []string) error {
			repo, err := repository()
			if err != nil {
				return err
			}

			entries, err := repo.List()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Fprintf(streams.Out, "No conversations yet. Create one with \"%s new <title>\".\n", commandName())
				return nil
			}

			writeEntries(streams.Out, entries)

			return nil
		}),
	}
}

// writeEntries renders the list as aligned columns.
func writeEntries(out io.Writer, entries []conversation.Entry) {
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	fmt.Fprintln(table, "ID\tTITLE\tSTATUS\tUPDATED")

	for _, entry := range entries {
		if !entry.Readable() {
			// Show the id so the user can find the directory, and say plainly
			// that Echo could not read it.
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", entry.ID, "(unreadable)", "-", "-")
			continue
		}

		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n",
			entry.Conversation.ID,
			entry.Conversation.Title,
			entry.Conversation.Status,
			entry.Conversation.UpdatedAt.Format(time.RFC3339),
		)
	}

	_ = table.Flush()

	for _, entry := range entries {
		if !entry.Readable() {
			fmt.Fprintf(out, "\n%s: %v\n", entry.ID, entry.Err)
		}
	}
}
