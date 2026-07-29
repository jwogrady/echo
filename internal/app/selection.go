package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/buildinfo"
	"github.com/jwogrady/echo/internal/conversation"
)

// conversationFlag is the global --conversation override. It is package state
// because Cobra binds persistent flags at construction; newRootCommand resets it
// per tree so tests stay independent.
type conversationFlag struct {
	value string
}

// target reports which conversation a command should act on.
//
// The flag wins over the persisted selection and does not change it, so a script
// passing --conversation never depends on, or disturbs, what a human selected.
func (f *conversationFlag) target(repo *conversation.Repository) (string, error) {
	if f.value != "" {
		return repo.Resolve(f.value)
	}

	id, err := repo.ActiveID()
	if err != nil {
		if errors.Is(err, conversation.ErrNoSelection) {
			return "", fmt.Errorf("%w\n\nSelect one with \"%s use <id-or-prefix>\", or pass --conversation",
				err, buildinfo.Name)
		}

		return "", err
	}

	return id, nil
}

// newUseCommand builds `ekko use <conversation-id-or-prefix>`.
func newUseCommand(streams Streams, dispatched *dispatch) *cobra.Command {
	return &cobra.Command{
		Use:   "use <conversation-id-or-prefix>",
		Short: "Select the active conversation",
		Long: "Select the conversation later commands act on.\n\n" +
			"A unique id prefix is enough. An ambiguous prefix is refused rather than\n" +
			"resolved to the first match, so you are never acting on a conversation you\n" +
			"did not mean. The selection is stored at the data root, not inside any\n" +
			"conversation directory.",
		Args: cobra.ExactArgs(1),
		RunE: dispatched.mark(func(_ *cobra.Command, args []string) error {
			// A blank argument is a malformed invocation, not a failed lookup.
			if strings.TrimSpace(args[0]) == "" {
				return usageErrorf("a conversation id or prefix is required")
			}

			repo, err := repository()
			if err != nil {
				return err
			}

			id, err := repo.Resolve(args[0])
			if err != nil {
				return err
			}

			if err := repo.SetActive(id); err != nil {
				return err
			}

			selected, err := repo.Get(id)
			if err != nil {
				return err
			}

			fmt.Fprintf(streams.Out, "Using %s\n", selected.ID)
			fmt.Fprintf(streams.Out, "  title   %s\n", selected.Title)
			fmt.Fprintf(streams.Out, "  status  %s\n", selected.Status)

			return nil
		}),
	}
}
