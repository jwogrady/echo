package audio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jwogrady/echo/internal/conversation"
)

// Import errors distinguishable by callers.
var (
	// ErrChecksumMismatch means the copy does not match the original. The copy is
	// removed: a recording Echo cannot vouch for is worse than none.
	ErrChecksumMismatch = errors.New("imported copy does not match the source")
	// ErrAlreadyImported means the identical file is already imported. It is
	// informational, not a failure.
	ErrAlreadyImported = errors.New("this recording is already imported")
	// ErrDifferentRecording means a different file is already imported and
	// replacing it was not requested.
	ErrDifferentRecording = errors.New("a different recording is already imported")
)

// Stage names a step of the import pipeline, so a failure says where it happened.
type Stage string

const (
	StageInspecting Stage = "inspecting source"
	StageHashing    Stage = "hashing source"
	StageCopying    Stage = "copying source"
	StageVerifying  Stage = "verifying copy"
	StageSaving     Stage = "saving metadata"
)

// StageError wraps a failure with the stage it happened in. "ffmpeg failed" is
// not actionable; "optimizing audio: ffmpeg failed" is a place to start.
type StageError struct {
	Stage Stage
	Cause error
}

func (e *StageError) Error() string { return fmt.Sprintf("%s: %v", e.Stage, e.Cause) }

func (e *StageError) Unwrap() error { return e.Cause }

// Importer copies validated audio into a conversation.
type Importer struct {
	// Now supplies the import timestamp.
	Now func() time.Time
	// NewID generates the recording identifier.
	NewID func() (string, error)
	// Inspect reports a file's audio properties. Optional: when nil the
	// recording is stored without properties, which a later inspection fills in.
	Inspect func(ctx context.Context, path string) (Properties, error)
	// Progress receives stage announcements. Optional.
	Progress func(Stage)
}

// NewImporter returns an Importer using the real clock and id generator.
func NewImporter() *Importer {
	prober := NewProber()

	return &Importer{
		Now:     func() time.Time { return time.Now().UTC() },
		NewID:   NewRecordingID,
		Inspect: prober.Inspect,
	}
}

// announce reports a stage if anyone is listening.
func (i *Importer) announce(stage Stage) {
	if i.Progress != nil {
		i.Progress(stage)
	}
}

// Import copies source into the conversation and records its checksum.
//
// The external file is opened read-only and never written to, moved, or deleted.
// The checksum is computed from the original and then recomputed from the copy;
// a mismatch fails the import and removes the copy.
//
// Replacing an existing recording requires replace, so a second `add` cannot
// silently discard audio the user may not have elsewhere. Re-importing the
// identical file returns ErrAlreadyImported, which callers may treat as a no-op.
func (i *Importer) Import(ctx context.Context, workspace conversation.Workspace, conversationID string, source Source, replace bool) (Recording, error) {
	existing, hasExisting, err := i.existingRecording(workspace)
	if err != nil {
		return Recording{}, err
	}

	// The documented pipeline inspects before hashing, and inspecting the
	// original rather than the copy means a file Echo cannot read is rejected
	// before anything is written.
	var properties *Properties
	if i.Inspect != nil {
		i.announce(StageInspecting)
		inspected, err := i.Inspect(ctx, source.Path)
		if err != nil {
			return Recording{}, &StageError{Stage: StageInspecting, Cause: err}
		}
		properties = &inspected
	}

	i.announce(StageHashing)
	sourceSum, err := checksumFile(source.Path)
	if err != nil {
		return Recording{}, &StageError{Stage: StageHashing, Cause: err}
	}

	if hasExisting && !replace {
		if existing.SHA256 == sourceSum {
			return existing, ErrAlreadyImported
		}

		return Recording{}, fmt.Errorf("%w (%s, imported %s); pass --replace to overwrite it",
			ErrDifferentRecording, existing.OriginalFilename, existing.ImportedAt.Format(time.RFC3339))
	}

	id, err := i.NewID()
	if err != nil {
		return Recording{}, err
	}

	destination := SourcePath(workspace)

	i.announce(StageCopying)
	written, err := copyFile(source.Path, destination)
	if err != nil {
		// A partial copy would be reported by status as a real recording.
		_ = os.Remove(destination)

		return Recording{}, &StageError{Stage: StageCopying, Cause: err}
	}

	i.announce(StageVerifying)
	copySum, err := checksumFile(destination)
	if err != nil {
		_ = os.Remove(destination)

		return Recording{}, &StageError{Stage: StageVerifying, Cause: err}
	}
	if copySum != sourceSum {
		_ = os.Remove(destination)

		return Recording{}, &StageError{
			Stage: StageVerifying,
			Cause: fmt.Errorf("%w: source %s, copy %s", ErrChecksumMismatch, sourceSum, copySum),
		}
	}

	recording := Recording{
		SchemaVersion:    RecordingSchemaVersion,
		ID:               id,
		ConversationID:   conversationID,
		OriginalFilename: source.Name,
		OriginalPath:     source.Path,
		SHA256:           sourceSum,
		SizeBytes:        written,
		ImportedAt:       i.Now(),
		SourceProperties: properties,
	}

	i.announce(StageSaving)
	if err := SaveRecording(workspace, recording); err != nil {
		// Without its document the copy is an orphan that nothing references.
		_ = os.Remove(destination)

		return Recording{}, &StageError{Stage: StageSaving, Cause: err}
	}

	return recording, nil
}

// existingRecording loads the current recording, if the conversation has one.
//
// A document that exists but cannot be read is an error rather than "no
// recording": treating damage as absence would overwrite audio on the next add.
func (i *Importer) existingRecording(workspace conversation.Workspace) (Recording, bool, error) {
	existing, err := LoadRecording(workspace)
	if err == nil {
		return existing, true, nil
	}
	if errors.Is(err, ErrNoRecording) {
		return Recording{}, false, nil
	}

	return Recording{}, false, err
}

// checksumFile returns the SHA-256 of a file as lowercase hex.
func checksumFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

// copyFile copies source to destination, returning the bytes written.
//
// The source is opened read-only so a bug here cannot damage the user's file.
// The destination is written through a temporary sibling and renamed, so an
// interrupted copy never leaves a truncated source.wav in place.
func copyFile(source, destination string) (int64, error) {
	in, err := os.Open(source)
	if err != nil {
		return 0, fmt.Errorf("opening %s: %w", source, err)
	}
	defer func() { _ = in.Close() }()

	temp, err := os.CreateTemp(filepath.Dir(destination), "."+SourceName+".tmp-*")
	if err != nil {
		return 0, fmt.Errorf("creating a temporary file: %w", err)
	}
	tempName := temp.Name()

	written, copyErr := io.Copy(temp, in)
	if copyErr != nil {
		_ = temp.Close()
		_ = os.Remove(tempName)

		return 0, fmt.Errorf("copying to %s: %w", tempName, copyErr)
	}

	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempName)

		return 0, fmt.Errorf("flushing %s: %w", tempName, err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)

		return 0, fmt.Errorf("closing %s: %w", tempName, err)
	}

	if err := os.Rename(tempName, destination); err != nil {
		_ = os.Remove(tempName)

		return 0, fmt.Errorf("moving into place as %s: %w", destination, err)
	}

	return written, nil
}

// RemoveRecording deletes a conversation's recording document and audio files.
// It is used by replacement, never automatically.
func RemoveRecording(workspace conversation.Workspace) error {
	var errs []error

	for _, path := range []string{
		SourcePath(workspace),
		OptimizedPath(workspace),
		recordingPath(workspace),
	} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("removing %s: %w", path, err))
		}
	}

	return errors.Join(errs...)
}
