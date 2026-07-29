// Package conversation defines Echo's central domain object and how it is
// stored.
//
// A conversation is a directory on disk holding one JSON metadata document plus
// the audio, jobs, transcript, and exports belonging to it. The filesystem is the
// source of truth — there is no database (ADR-0001 keeps orchestration in Go and
// state on disk).
package conversation

import (
	"errors"
	"fmt"
	"time"
)

// SchemaVersion is the version this build writes and can read. A document
// declaring a higher version is refused rather than misread.
const SchemaVersion = 1

// Status is where a conversation has got to. It is a closed set: an unrecognized
// value is a detectable error, never a silently tolerated string.
type Status string

const (
	// StatusCreated means the workspace exists and holds no audio yet.
	StatusCreated Status = "created"
	// StatusRecordingAdded means a source recording has been imported.
	StatusRecordingAdded Status = "recording_added"
	// StatusAudioReady means a transcription-ready derivative exists.
	StatusAudioReady Status = "audio_ready"
	// StatusTranscribing means a transcription job is running.
	StatusTranscribing Status = "transcribing"
	// StatusTranscribed means a validated transcript exists.
	StatusTranscribed Status = "transcribed"
	// StatusFailed means the last operation failed and was recorded as such.
	StatusFailed Status = "failed"
)

// statuses is every valid Status, in lifecycle order.
var statuses = []Status{
	StatusCreated,
	StatusRecordingAdded,
	StatusAudioReady,
	StatusTranscribing,
	StatusTranscribed,
	StatusFailed,
}

// Valid reports whether s is a status this build understands.
func (s Status) Valid() bool {
	for _, known := range statuses {
		if s == known {
			return true
		}
	}

	return false
}

// String makes Status printable.
func (s Status) String() string { return string(s) }

// Statuses returns every valid status, for error messages and tests.
func Statuses() []Status {
	return append([]Status(nil), statuses...)
}

// Conversation is the metadata document stored as conversation.json.
//
// Field names are the on-disk contract: renaming one is a schema change and
// requires a SchemaVersion bump.
type Conversation struct {
	// SchemaVersion is the document's format version.
	SchemaVersion int `json:"schema_version"`
	// ID is the conversation's identity. It never changes.
	ID string `json:"id"`
	// Title is what the user called it. Not an identifier: titles may repeat.
	Title string `json:"title"`
	// Slug is a filesystem- and URL-safe rendering of Title, for humans only.
	Slug string `json:"slug"`
	// CreatedAt is when the workspace was created, in UTC.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the metadata last changed, in UTC.
	UpdatedAt time.Time `json:"updated_at"`
	// Status is the lifecycle position.
	Status Status `json:"status"`
	// ActiveRecordingID is the recording later commands operate on, empty until
	// one is imported.
	ActiveRecordingID string `json:"active_recording_id,omitempty"`
	// ActiveTranscriptID is the transcript later commands read, empty until one
	// is produced.
	ActiveTranscriptID string `json:"active_transcript_id,omitempty"`
}

// Errors describing why a document cannot be used. Callers distinguish these so
// they can tell a user something actionable instead of "invalid".
var (
	// ErrUnsupportedSchema means the document was written by a newer Echo.
	ErrUnsupportedSchema = errors.New("unsupported conversation schema version")
	// ErrInvalid means a required field is missing or malformed.
	ErrInvalid = errors.New("invalid conversation metadata")
)

// Validate reports whether c is a document this build can use.
//
// It deliberately rejects rather than repairs. A conversation directory may hold
// the user's only copy of their thinking, and silently rewriting a field Echo
// did not understand is how that gets lost.
func (c Conversation) Validate() error {
	if c.SchemaVersion > SchemaVersion {
		return fmt.Errorf("%w: document is version %d, this build supports %d",
			ErrUnsupportedSchema, c.SchemaVersion, SchemaVersion)
	}
	if c.SchemaVersion < 1 {
		return fmt.Errorf("%w: schema_version is %d", ErrInvalid, c.SchemaVersion)
	}
	if c.ID == "" {
		return fmt.Errorf("%w: id is empty", ErrInvalid)
	}
	if c.Title == "" {
		return fmt.Errorf("%w: title is empty", ErrInvalid)
	}
	if !c.Status.Valid() {
		return fmt.Errorf("%w: status %q is not one of %v", ErrInvalid, c.Status, statuses)
	}
	if c.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_at is missing", ErrInvalid)
	}
	if c.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: updated_at is missing", ErrInvalid)
	}

	return nil
}
