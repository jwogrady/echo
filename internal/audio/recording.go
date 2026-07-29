package audio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jwogrady/echo/internal/conversation"
)

// RecordingSchemaVersion is the version this build writes and can read.
const RecordingSchemaVersion = 1

// RecordingIDPrefix marks a recording identifier, so a bare id is never
// ambiguous about what it names.
const RecordingIDPrefix = "rec_"

// File names inside a conversation's audio directory.
const (
	// SourceName is the imported original, preserved byte-for-byte.
	SourceName = "source.wav"
	// OptimizedName is the transcription-ready derivative.
	OptimizedName = "optimized.wav"
	// RecordingFile holds the Recording document.
	//
	// It lives beside the audio it describes rather than at the conversation
	// root, so a workspace's audio directory is self-describing: someone
	// recovering files by hand can see which WAV came from where.
	RecordingFile = "recording.json"
)

// Properties are the audio characteristics ffprobe reports.
//
// This is a separate, optional struct so "not yet inspected" is representable.
// A zero duration or sample rate would be indistinguishable from a real value
// and would later look like an empty recording.
type Properties struct {
	// DurationSeconds is the audio length.
	DurationSeconds float64 `json:"duration_seconds"`
	// SampleRate is in hertz.
	SampleRate int `json:"sample_rate"`
	// Channels is the channel count.
	Channels int `json:"channels"`
	// SampleFormat is ffprobe's sample_fmt, such as "s16".
	SampleFormat string `json:"sample_format"`
	// Codec is ffprobe's codec_name, such as "pcm_s16le".
	Codec string `json:"codec"`
}

// Recording describes one imported audio file.
//
// Field names are the on-disk contract; renaming one is a schema change.
type Recording struct {
	// SchemaVersion is the document's format version.
	SchemaVersion int `json:"schema_version"`
	// ID identifies this recording.
	ID string `json:"id"`
	// ConversationID is the conversation it belongs to.
	ConversationID string `json:"conversation_id"`
	// OriginalFilename is what the user called the file.
	OriginalFilename string `json:"original_filename"`
	// OriginalPath is where it came from, kept as provenance only. The imported
	// copy is authoritative; this path may since have moved or been deleted.
	OriginalPath string `json:"original_path"`
	// SHA256 is the checksum of the imported copy, verified after the copy.
	SHA256 string `json:"sha256"`
	// SizeBytes is the imported copy's size.
	SizeBytes int64 `json:"size_bytes"`
	// ImportedAt is when the import completed, in UTC.
	ImportedAt time.Time `json:"imported_at"`
	// SourceProperties describes the imported original. Nil until inspected.
	SourceProperties *Properties `json:"source_properties,omitempty"`
	// OptimizedProperties describes the derivative. Nil until one is built.
	OptimizedProperties *Properties `json:"optimized_properties,omitempty"`
	// OptimizedSHA256 is the derivative's checksum. Empty until one is built.
	OptimizedSHA256 string `json:"optimized_sha256,omitempty"`
}

// Errors describing why a recording document cannot be used.
var (
	// ErrNoRecording means the conversation holds no recording.
	ErrNoRecording = errors.New("conversation has no recording")
	// ErrRecordingMalformed means the document is not usable JSON.
	ErrRecordingMalformed = errors.New("recording metadata is not valid JSON")
	// ErrRecordingInvalid means a required field is missing or impossible.
	ErrRecordingInvalid = errors.New("invalid recording metadata")
	// ErrRecordingUnsupportedSchema means it was written by a newer Echo.
	ErrRecordingUnsupportedSchema = errors.New("unsupported recording schema version")
)

// NewRecordingID returns a fresh recording identifier.
func NewRecordingID() (string, error) {
	return conversation.NewIDWithPrefix(RecordingIDPrefix)
}

// ValidRecordingID reports whether id is well formed.
func ValidRecordingID(id string) bool {
	return conversation.ValidIDWithPrefix(RecordingIDPrefix, id)
}

// Validate reports whether r is a document this build can use.
func (r Recording) Validate() error {
	if r.SchemaVersion > RecordingSchemaVersion {
		return fmt.Errorf("%w: document is version %d, this build supports %d",
			ErrRecordingUnsupportedSchema, r.SchemaVersion, RecordingSchemaVersion)
	}
	if r.SchemaVersion < 1 {
		return fmt.Errorf("%w: schema_version is %d", ErrRecordingInvalid, r.SchemaVersion)
	}
	if !ValidRecordingID(r.ID) {
		return fmt.Errorf("%w: id %q is not a recording id", ErrRecordingInvalid, r.ID)
	}
	if r.OriginalFilename == "" {
		return fmt.Errorf("%w: original_filename is empty", ErrRecordingInvalid)
	}
	if len(r.SHA256) != 64 {
		return fmt.Errorf("%w: sha256 is %d characters, want 64", ErrRecordingInvalid, len(r.SHA256))
	}
	if r.SizeBytes <= 0 {
		return fmt.Errorf("%w: size_bytes is %d", ErrRecordingInvalid, r.SizeBytes)
	}
	if r.ImportedAt.IsZero() {
		return fmt.Errorf("%w: imported_at is missing", ErrRecordingInvalid)
	}

	return nil
}

// recordingPath is where a workspace keeps its recording document.
func recordingPath(workspace conversation.Workspace) string {
	return filepath.Join(workspace.AudioPath(), RecordingFile)
}

// SourcePath is the imported original's path.
func SourcePath(workspace conversation.Workspace) string {
	return filepath.Join(workspace.AudioPath(), SourceName)
}

// OptimizedPath is the derivative's path.
func OptimizedPath(workspace conversation.Workspace) string {
	return filepath.Join(workspace.AudioPath(), OptimizedName)
}

// LoadRecording reads and validates a conversation's recording document.
//
// Like conversation metadata, a document Echo cannot understand is reported
// rather than repaired: it describes audio the user cannot regenerate.
func LoadRecording(workspace conversation.Workspace) (Recording, error) {
	path := recordingPath(workspace)

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Recording{}, fmt.Errorf("%w: no %s in %s", ErrNoRecording, RecordingFile, workspace.AudioPath())
		}

		return Recording{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var loaded Recording
	if err := json.Unmarshal(contents, &loaded); err != nil {
		return Recording{}, fmt.Errorf("%w: %s: %w", ErrRecordingMalformed, path, err)
	}

	if err := loaded.Validate(); err != nil {
		return Recording{}, fmt.Errorf("%s: %w", path, err)
	}

	return loaded, nil
}

// SaveRecording writes the recording document atomically.
func SaveRecording(workspace conversation.Workspace, r Recording) error {
	if err := r.Validate(); err != nil {
		return fmt.Errorf("refusing to write invalid recording metadata: %w", err)
	}

	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding recording %s: %w", r.ID, err)
	}
	encoded = append(encoded, '\n')

	return conversation.WriteFileAtomic(recordingPath(workspace), encoded)
}
