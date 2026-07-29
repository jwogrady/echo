package conversation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jwogrady/echo/internal/config"
)

// Repository is every conversation under one data root.
type Repository struct {
	paths config.Paths
	now   func() time.Time
	newID func() (string, error)
}

// NewRepository opens the repository rooted at paths.
func NewRepository(paths config.Paths) *Repository {
	return &Repository{
		paths: paths,
		now:   func() time.Time { return time.Now().UTC() },
		newID: NewID,
	}
}

// Entry is one conversation as `list` sees it.
//
// A directory that cannot be read is still an entry, carrying Err instead of a
// document: one damaged conversation must not hide the rest.
type Entry struct {
	// ID is the directory name, which is the conversation's identity.
	ID string
	// Conversation is the loaded document. Zero when Err is set.
	Conversation Conversation
	// Err explains why this entry could not be read, or nil.
	Err error
}

// Readable reports whether this entry's document loaded.
func (e Entry) Readable() bool { return e.Err == nil }

// Workspace locates a conversation's directory by ID.
func (r *Repository) Workspace(id string) Workspace {
	return NewWorkspace(r.paths.ConversationDir(id))
}

// Create makes a new conversation workspace for title and returns its document.
//
// Titles are labels, not keys: creating two conversations with the same title
// yields two conversations with distinct IDs and distinct directories.
func (r *Repository) Create(title string) (Conversation, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return Conversation{}, errors.New("a conversation needs a title")
	}

	id, err := r.newID()
	if err != nil {
		return Conversation{}, err
	}

	now := r.now()
	created := Conversation{
		SchemaVersion: SchemaVersion,
		ID:            id,
		Title:         trimmed,
		Slug:          Slugify(trimmed),
		CreatedAt:     now,
		UpdatedAt:     now,
		Status:        StatusCreated,
	}

	workspace := r.Workspace(id)
	if err := CreateWorkspace(workspace, created); err != nil {
		// A half-built workspace would be reported by list as a real
		// conversation, so remove it rather than leaving it to be discovered.
		_ = os.RemoveAll(workspace.Dir)

		return Conversation{}, err
	}

	return created, nil
}

// Get loads one conversation by its exact ID.
func (r *Repository) Get(id string) (Conversation, error) {
	if !ValidID(id) {
		return Conversation{}, fmt.Errorf("%w: %q is not a conversation id", ErrInvalid, id)
	}

	return Load(r.Workspace(id))
}

// List returns every conversation, newest update first, with ties broken by ID so
// the order is total and the same data always renders identically.
func (r *Repository) List() ([]Entry, error) {
	dir := r.paths.ConversationsDir()

	found, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No conversations have been created yet. That is not an error.
			return nil, nil
		}

		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	entries := make([]Entry, 0, len(found))
	for _, candidate := range found {
		if !candidate.IsDir() || !ValidID(candidate.Name()) {
			// Anything that is not a conversation directory is not ours to
			// report on.
			continue
		}

		entry := Entry{ID: candidate.Name()}
		entry.Conversation, entry.Err = Load(r.Workspace(candidate.Name()))
		entries = append(entries, entry)
	}

	sortEntries(entries)

	return entries, nil
}

// sortEntries orders entries deterministically: most recently updated first,
// then by ID. Unreadable entries sort last, since they have no timestamp to
// compare and burying them under readable ones keeps normal output stable.
func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]

		if left.Readable() != right.Readable() {
			return left.Readable()
		}
		if !left.Readable() {
			return left.ID < right.ID
		}
		if !left.Conversation.UpdatedAt.Equal(right.Conversation.UpdatedAt) {
			return left.Conversation.UpdatedAt.After(right.Conversation.UpdatedAt)
		}

		return left.ID < right.ID
	})
}
