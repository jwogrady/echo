package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/buildinfo"
	"github.com/jwogrady/echo/internal/conversation"
)

// newStatusCommand builds `ekko status`.
func newStatusCommand(streams Streams, selected *conversationFlag, dispatched *dispatch) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active conversation's state",
		Long: "Report what is actually true about the active conversation.\n\n" +
			"State comes from what is recorded on disk, never from a guess: if no\n" +
			"recording has been imported, status says so rather than implying Echo is\n" +
			"ready to transcribe.",
		Args: cobra.NoArgs,
		RunE: dispatched.mark(func(_ *cobra.Command, _ []string) error {
			repo, err := repository()
			if err != nil {
				return err
			}

			id, err := selected.target(repo)
			if err != nil {
				return err
			}

			workspace := repo.Workspace(id)

			current, err := repo.Get(id)
			if err != nil {
				// A conversation that cannot be read is still reported: the user
				// needs the path and the reason, not a bare failure.
				writeUnreadable(streams.Out, id, workspace, err)

				return &reportedError{cause: err}
			}

			writeStatus(streams.Out, current, workspace)

			return nil
		}),
	}
}

// writeStatus renders a readable conversation's state.
func writeStatus(out io.Writer, current conversation.Conversation, workspace conversation.Workspace) {
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	fmt.Fprintf(table, "conversation\t%s\n", current.ID)
	fmt.Fprintf(table, "title\t%s\n", current.Title)
	fmt.Fprintf(table, "status\t%s\n", current.Status)
	fmt.Fprintf(table, "created\t%s\n", current.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(table, "updated\t%s\n", current.UpdatedAt.Format(time.RFC3339))

	// Say plainly what does not exist yet rather than leaving a blank field, so
	// nobody reads an empty value as "ready".
	fmt.Fprintf(table, "recording\t%s\n", presentOr(current.ActiveRecordingID, "none imported yet"))
	fmt.Fprintf(table, "transcript\t%s\n", presentOr(current.ActiveTranscriptID, "none yet"))
	fmt.Fprintf(table, "path\t%s\n", workspace.Dir)

	_ = table.Flush()

	writeNextStep(out, current)
}

// presentOr renders value, or a plain statement of its absence.
func presentOr(value, absent string) string {
	if value == "" {
		return absent
	}

	return value
}

// writeNextStep names the one command that moves this conversation forward.
// Commands not yet implemented are described as not available rather than
// suggested, so status never advertises something that would fail.
func writeNextStep(out io.Writer, current conversation.Conversation) {
	switch current.Status {
	case conversation.StatusCreated:
		fmt.Fprintf(out, "\nNext: import a recording with \"%s add <wav-path>\" (not available in this build).\n",
			buildinfo.Name)
	case conversation.StatusRecordingAdded, conversation.StatusAudioReady:
		fmt.Fprintf(out, "\nNext: transcribe with \"%s transcribe\" (not available in this build).\n",
			buildinfo.Name)
	case conversation.StatusTranscribed:
		fmt.Fprintf(out, "\nNext: read it with \"%s show\" (not available in this build).\n",
			buildinfo.Name)
	case conversation.StatusFailed:
		fmt.Fprintln(out, "\nThe last operation on this conversation failed.")
	}
}

// writeUnreadable reports a conversation whose metadata could not be loaded.
//
// It never modifies the file. The guidance names the path and the concrete things
// a person can do, because Echo will not repair the document for them.
func writeUnreadable(out io.Writer, id string, workspace conversation.Workspace, cause error) {
	fmt.Fprintf(out, "conversation  %s\n", id)
	fmt.Fprintf(out, "status        unreadable\n")
	fmt.Fprintf(out, "path          %s\n", workspace.Dir)
	fmt.Fprintf(out, "metadata      %s\n", workspace.MetadataPath())
	fmt.Fprintf(out, "reason        %v\n", cause)

	fmt.Fprintf(out, "\n%s\n", recoveryGuidance(workspace, cause))

	// The rest of the workspace may be intact, and audio is the part a user
	// cannot regenerate, so say what survived.
	if surviving := survivingContents(workspace); len(surviving) > 0 {
		fmt.Fprintln(out, "\nStill present in the workspace:")
		for _, entry := range surviving {
			fmt.Fprintf(out, "  %s\n", entry)
		}
	}
}

// recoveryGuidance explains what to do about a specific kind of damage.
func recoveryGuidance(workspace conversation.Workspace, cause error) string {
	switch {
	case errors.Is(cause, conversation.ErrUnsupportedSchema):
		return "This conversation was written by a newer version of Echo. Upgrade Echo\n" +
			"rather than editing the file; this build would lose whatever it does not\n" +
			"understand."

	case errors.Is(cause, conversation.ErrNotFound):
		return fmt.Sprintf("There is no metadata file at %s. If the directory holds audio you want\n"+
			"to keep, copy it somewhere safe, then create a new conversation with\n"+
			"\"%s new <title>\".", workspace.MetadataPath(), buildinfo.Name)

	default:
		return fmt.Sprintf("Echo will not repair this file for you. Inspect %s, restore it from a\n"+
			"backup, or copy any audio you want to keep and start a new conversation\n"+
			"with \"%s new <title>\".", workspace.MetadataPath(), buildinfo.Name)
	}
}

// survivingContents lists what the workspace still holds, ignoring the metadata
// file itself.
func survivingContents(workspace conversation.Workspace) []string {
	entries, err := os.ReadDir(workspace.Dir)
	if err != nil {
		return nil
	}

	var surviving []string
	for _, entry := range entries {
		if entry.Name() == conversation.MetadataFile {
			continue
		}

		if entry.IsDir() {
			count := countFiles(filepath.Join(workspace.Dir, entry.Name()))
			if count == 0 {
				continue
			}
			surviving = append(surviving, fmt.Sprintf("%s/ (%d file(s))", entry.Name(), count))

			continue
		}

		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			continue
		}
		surviving = append(surviving, entry.Name())
	}

	return surviving
}

// countFiles reports how many entries a directory holds.
func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	return len(entries)
}
