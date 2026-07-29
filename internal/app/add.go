package app

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jwogrady/echo/internal/audio"
	"github.com/jwogrady/echo/internal/conversation"
)

// newAddCommand builds `ekko add <wav-path>`.
func newAddCommand(streams Streams, selected *conversationFlag, dispatched *dispatch) *cobra.Command {
	var replace bool

	command := &cobra.Command{
		Use:   "add <wav-path>",
		Short: "Import a WAV recording",
		Long: "Import a WAV recording into the active conversation.\n\n" +
			"Your file is copied, never moved or modified, and the copy is checksum-\n" +
			"verified against the original. Re-importing the same file is a no-op.\n" +
			"Importing a different file over an existing recording requires --replace.",
		Args: cobra.ExactArgs(1),
		RunE: dispatched.mark(func(cmd *cobra.Command, args []string) error {
			repo, err := repository()
			if err != nil {
				return err
			}

			id, err := selected.target(repo)
			if err != nil {
				return err
			}

			current, err := repo.Get(id)
			if err != nil {
				return err
			}

			source, err := audio.Validate(args[0])
			if err != nil {
				return err
			}

			workspace := repo.Workspace(id)

			importer := audio.NewImporter()
			importer.Progress = func(stage audio.Stage) {
				fmt.Fprintf(streams.Err, "  %s...\n", stage)
			}

			recording, err := importer.Import(cmd.Context(), workspace, id, source, replace)
			if err != nil {
				if errors.Is(err, audio.ErrAlreadyImported) {
					fmt.Fprintf(streams.Out, "Already imported: %s (%s)\n",
						recording.OriginalFilename, recording.ID)
					fmt.Fprintf(streams.Out, "Nothing to do. Pass --replace to import a different file.\n")

					return nil
				}

				return err
			}

			// The recording is committed, so record it on the conversation. The
			// status advances only this far: audio_ready waits for a validated
			// derivative, which this build does not produce yet.
			current.ActiveRecordingID = recording.ID
			current.Status = conversation.StatusRecordingAdded
			current.UpdatedAt = recording.ImportedAt
			if err := conversation.Save(workspace, current); err != nil {
				return err
			}

			fmt.Fprintf(streams.Out, "Imported %s\n", recording.OriginalFilename)
			fmt.Fprintf(streams.Out, "  recording  %s\n", recording.ID)
			fmt.Fprintf(streams.Out, "  bytes      %d\n", recording.SizeBytes)
			fmt.Fprintf(streams.Out, "  sha256     %s\n", recording.SHA256)
			fmt.Fprintf(streams.Out, "  stored     %s\n", audio.SourcePath(workspace))

			if properties := recording.SourceProperties; properties != nil {
				fmt.Fprintf(streams.Out, "  audio      %s, %d Hz, %d channel(s), %.2fs\n",
					properties.Codec, properties.SampleRate, properties.Channels, properties.DurationSeconds)
			}

			return nil
		}),
	}

	command.Flags().BoolVar(&replace, "replace", false,
		"replace an existing recording, discarding the current audio")

	return command
}
