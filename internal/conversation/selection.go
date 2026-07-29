package conversation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SelectionFile records which conversation is active. It lives at the data root,
// deliberately outside every conversation directory: copying or deleting one
// conversation must not silently change which is selected, and a selection
// pointing at a deleted conversation must be detectable rather than vanish.
const SelectionFile = "active.json"

// SelectionSchemaVersion is the version of the selection document.
const SelectionSchemaVersion = 1

// Selection is the persisted active-conversation pointer.
type Selection struct {
	// SchemaVersion is the document's format version.
	SchemaVersion int `json:"schema_version"`
	// ConversationID is the active conversation.
	ConversationID string `json:"conversation_id"`
	// SelectedAt is when it was selected, in UTC.
	SelectedAt time.Time `json:"selected_at"`
}

// Resolution errors. Callers distinguish these because "you typed too few
// characters" and "that conversation does not exist" need different advice.
var (
	// ErrNoSelection means no conversation has been selected yet.
	ErrNoSelection = errors.New("no conversation selected")
	// ErrNoMatch means nothing matched the given id or prefix.
	ErrNoMatch = errors.New("no conversation matches")
	// ErrAmbiguous means a prefix matched more than one conversation.
	ErrAmbiguous = errors.New("ambiguous conversation prefix")
)

// selectionPath is where the pointer lives.
func (r *Repository) selectionPath() string {
	return filepath.Join(r.paths.Root, SelectionFile)
}

// ActiveID reports the selected conversation's ID.
//
// It returns the recorded ID without checking that the conversation still
// exists; callers that need a usable conversation load it and get a clear
// not-found error, which is more useful than conflating the two failures.
func (r *Repository) ActiveID() (string, error) {
	contents, err := os.ReadFile(r.selectionPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", ErrNoSelection
		}

		return "", fmt.Errorf("reading %s: %w", r.selectionPath(), err)
	}

	var selection Selection
	if err := json.Unmarshal(contents, &selection); err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrMalformed, r.selectionPath(), err)
	}

	if selection.SchemaVersion > SelectionSchemaVersion {
		return "", fmt.Errorf("%w: %s is version %d, this build supports %d",
			ErrUnsupportedSchema, SelectionFile, selection.SchemaVersion, SelectionSchemaVersion)
	}
	if !ValidID(selection.ConversationID) {
		return "", fmt.Errorf("%w: %s names %q, which is not a conversation id",
			ErrInvalid, SelectionFile, selection.ConversationID)
	}

	return selection.ConversationID, nil
}

// SetActive records id as the active conversation.
//
// The conversation must exist: selecting something absent would leave the user
// with a selection that fails on every later command. A failure here leaves any
// previous selection untouched, because the write is atomic.
func (r *Repository) SetActive(id string) error {
	if _, err := r.Get(id); err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(Selection{
		SchemaVersion:  SelectionSchemaVersion,
		ConversationID: id,
		SelectedAt:     r.now(),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the selection: %w", err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(r.paths.Root, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", r.paths.Root, err)
	}

	return WriteFileAtomic(r.selectionPath(), encoded)
}

// Resolve turns a full ID or a unique prefix into a conversation ID.
//
// An ambiguous prefix is refused rather than resolved to the first match:
// silently acting on the wrong conversation is worse than making the user type
// two more characters. Matching is case-insensitive and the cnv_ prefix is
// optional, since both are things a person reasonably types.
func (r *Repository) Resolve(input string) (string, error) {
	query := strings.ToUpper(strings.TrimSpace(input))
	if query == "" {
		return "", fmt.Errorf("%w: no conversation id or prefix given", ErrNoMatch)
	}

	// Accept the id with or without its prefix.
	query = strings.TrimPrefix(query, strings.ToUpper(IDPrefix))

	entries, err := r.List()
	if err != nil {
		return "", err
	}

	var matches []string
	for _, entry := range entries {
		body := strings.TrimPrefix(entry.ID, IDPrefix)
		if strings.HasPrefix(body, query) {
			matches = append(matches, entry.ID)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("%w %q", ErrNoMatch, input)
	default:
		return "", fmt.Errorf("%w %q: it matches %d conversations (%s)",
			ErrAmbiguous, input, len(matches), strings.Join(matches, ", "))
	}
}
