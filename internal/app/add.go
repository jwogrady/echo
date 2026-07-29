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
			"verified against the original.\n\n" +
			"Safe to run twice. Re-importing the same file does nothing, unless the\n" +
			"optimized derivative is missing or unusable — then it is rebuilt, so an\n" +
			"interrupted import is repaired by simply running add again.\n\n" +
			"Importing a different file over an existing recording requires --replace,\n" +
			"which discards the current audio and its derivative.",
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

			var alreadyImported bool

			importer := audio.NewImporter()
			importer.Progress = func(stage audio.Stage) {
				fmt.Fprintf(streams.Err, "  %s...\n", stage)
			}

			recording, err := importer.Import(cmd.Context(), workspace, id, source, replace)
			if err != nil {
				if !errors.Is(err, audio.ErrAlreadyImported) {
					return err
				}

				// The same file is already imported, but the derivative may be
				// missing or unusable if an earlier run was interrupted. Repairing
				// it here is what makes add safe to retry.
				alreadyImported = true
			}

			// Guarantee a validated derivative. The status advances to audio_ready
			// only once one exists, so an interrupted add leaves a conversation
			// that truthfully says it has no usable audio.
			converter := audio.NewConverter()
			converter.Progress = func(stage audio.Stage) {
				fmt.Fprintf(streams.Err, "  %s...\n", stage)
			}

			optimized, rebuilt, err := converter.EnsureOptimized(cmd.Context(), workspace)
			if err != nil {
				return err
			}

			if rebuilt || recording.OptimizedProperties == nil {
				recording.OptimizedProperties = &optimized
				if err := audio.SaveRecording(workspace, recording); err != nil {
					return err
				}
			}

			current.ActiveRecordingID = recording.ID
			current.Status = conversation.StatusAudioReady
			current.UpdatedAt = recording.ImportedAt
			if err := conversation.Save(workspace, current); err != nil {
				return err
			}

			if alreadyImported && !rebuilt {
				fmt.Fprintf(streams.Out, "Already imported: %s (%s)\n",
					recording.OriginalFilename, recording.ID)
				fmt.Fprintf(streams.Out, "Nothing to do. Pass --replace to import a different file.\n")

				return nil
			}

			if alreadyImported {
				fmt.Fprintf(streams.Out, "Repaired %s\n", recording.OriginalFilename)
			} else {
				fmt.Fprintf(streams.Out, "Imported %s\n", recording.OriginalFilename)
			}
			fmt.Fprintf(streams.Out, "  recording  %s\n", recording.ID)
			fmt.Fprintf(streams.Out, "  bytes      %d\n", recording.SizeBytes)
			fmt.Fprintf(streams.Out, "  sha256     %s\n", recording.SHA256)
			fmt.Fprintf(streams.Out, "  stored     %s\n", audio.SourcePath(workspace))

			if properties := recording.SourceProperties; properties != nil {
				fmt.Fprintf(streams.Out, "  source     %s, %d Hz, %d channel(s), %.2fs\n",
					properties.Codec, properties.SampleRate, properties.Channels, properties.DurationSeconds)
			}
			fmt.Fprintf(streams.Out, "  optimized  %s, %d Hz, %d channel(s), %.2fs\n",
				optimized.Codec, optimized.SampleRate, optimized.Channels, optimized.DurationSeconds)
			fmt.Fprintf(streams.Out, "             %s\n", audio.OptimizedPath(workspace))

			return nil
		}),
	}

	command.Flags().BoolVar(&replace, "replace", false,
		"replace an existing recording, discarding the current audio")

	return command
}
