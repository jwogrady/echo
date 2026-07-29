package conversation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createWithID makes a conversation with a chosen id, so prefix collisions can be
// constructed deliberately. Random ids practically never share a prefix, which
// would leave the ambiguity path untested.
func createWithID(t *testing.T, repo *Repository, id, title string) Conversation {
	t.Helper()

	previous := repo.newID
	repo.newID = func() (string, error) { return id, nil }
	defer func() { repo.newID = previous }()

	created, err := repo.Create(title)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", title, err)
	}

	return created
}

// id builds a valid identifier from a body prefix, padded to the right length.
func id(body string) string {
	return IDPrefix + body + strings.Repeat("0", idBodyLength-len(body))
}

func TestNoSelectionInitially(t *testing.T) {
	repo, _ := newTestRepository(t)

	if _, err := repo.ActiveID(); !errors.Is(err, ErrNoSelection) {
		t.Errorf("error = %v, want ErrNoSelection", err)
	}
}

func TestSetActiveThenActiveID(t *testing.T) {
	repo, _ := newTestRepository(t)
	created := createWithID(t, repo, id("AAHA"), "Aaha")

	if err := repo.SetActive(created.ID); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}

	active, err := repo.ActiveID()
	if err != nil {
		t.Fatalf("ActiveID() error = %v", err)
	}
	if active != created.ID {
		t.Errorf("ActiveID() = %q, want %q", active, created.ID)
	}
}

// The pointer must not live inside a conversation directory, or copying or
// deleting one conversation could change which is selected.
func TestSelectionIsStoredOutsideConversationDirectories(t *testing.T) {
	repo, paths := newTestRepository(t)
	created := createWithID(t, repo, id("AAHA"), "Aaha")

	if err := repo.SetActive(created.ID); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}

	expected := filepath.Join(paths.Root, SelectionFile)
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("selection not at %s: %v", expected, err)
	}

	if strings.HasPrefix(expected, paths.ConversationsDir()) {
		t.Errorf("selection at %q lives under the conversations directory", expected)
	}

	// Removing the conversation must not remove the selection.
	if err := os.RemoveAll(repo.Workspace(created.ID).Dir); err != nil {
		t.Fatalf("removing the conversation: %v", err)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("deleting a conversation removed the selection: %v", err)
	}

	// And the stale pointer is still readable, so it can be reported.
	if active, err := repo.ActiveID(); err != nil || active != created.ID {
		t.Errorf("ActiveID() = %q, %v; want the recorded id and no error", active, err)
	}
}

// Selecting something absent would leave every later command failing.
func TestSetActiveRejectsAMissingConversation(t *testing.T) {
	repo, _ := newTestRepository(t)

	if err := repo.SetActive(id("GHST")); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestSetActiveRejectsAMalformedID(t *testing.T) {
	repo, _ := newTestRepository(t)

	if err := repo.SetActive("../../etc"); !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

// A failed selection must leave the previous one intact.
func TestFailedSetActiveKeepsThePreviousSelection(t *testing.T) {
	repo, _ := newTestRepository(t)
	kept := createWithID(t, repo, id("KEEP"), "Keep")

	if err := repo.SetActive(kept.ID); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}
	if err := repo.SetActive(id("GHST")); err == nil {
		t.Fatal("expected the second SetActive to fail")
	}

	active, err := repo.ActiveID()
	if err != nil {
		t.Fatalf("ActiveID() error = %v", err)
	}
	if active != kept.ID {
		t.Errorf("ActiveID() = %q, want the earlier selection %q", active, kept.ID)
	}
}

func TestResolveAcceptsAFullID(t *testing.T) {
	repo, _ := newTestRepository(t)
	created := createWithID(t, repo, id("AAHA"), "Aaha")

	resolved, err := repo.Resolve(created.ID)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved != created.ID {
		t.Errorf("Resolve() = %q, want %q", resolved, created.ID)
	}
}

func TestResolveAcceptsAUniquePrefix(t *testing.T) {
	repo, _ := newTestRepository(t)
	aaha := createWithID(t, repo, id("AAHA"), "Aaha")
	createWithID(t, repo, id("BETA"), "Beta")

	// Forms a person plausibly types: with and without the cnv_ prefix, in
	// either case.
	for _, input := range []string{
		"AAHA", "aaha", "AaHa",
		"cnv_AAHA", "CNV_AAHA", "cnv_aaha",
		"AA", "a",
	} {
		t.Run(input, func(t *testing.T) {
			resolved, err := repo.Resolve(input)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v", input, err)
			}
			if resolved != aaha.ID {
				t.Errorf("Resolve(%q) = %q, want %q", input, resolved, aaha.ID)
			}
		})
	}
}

// Resolving to the first match would silently act on the wrong conversation.
func TestResolveRefusesAnAmbiguousPrefix(t *testing.T) {
	repo, _ := newTestRepository(t)
	first := createWithID(t, repo, id("SHARE1"), "First")
	second := createWithID(t, repo, id("SHARE2"), "Second")

	_, err := repo.Resolve("SHARE")
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("error = %v, want ErrAmbiguous", err)
	}

	// The message must name the candidates, or the user cannot act on it.
	for _, candidate := range []string{first.ID, second.ID} {
		if !strings.Contains(err.Error(), candidate) {
			t.Errorf("error = %q, want it to list %q", err, candidate)
		}
	}
	if !strings.Contains(err.Error(), "2 conversations") {
		t.Errorf("error = %q, want it to state how many matched", err)
	}
}

// One more character must disambiguate.
func TestALongerPrefixResolvesAnAmbiguity(t *testing.T) {
	repo, _ := newTestRepository(t)
	first := createWithID(t, repo, id("SHARE1"), "First")
	createWithID(t, repo, id("SHARE2"), "Second")

	resolved, err := repo.Resolve("SHARE1")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved != first.ID {
		t.Errorf("Resolve() = %q, want %q", resolved, first.ID)
	}
}

func TestResolveReportsNoMatch(t *testing.T) {
	repo, _ := newTestRepository(t)
	createWithID(t, repo, id("AAHA"), "Aaha")

	for _, input := range []string{"ZZZZ", "cnv_ZZZZ", "  ", ""} {
		if _, err := repo.Resolve(input); !errors.Is(err, ErrNoMatch) {
			t.Errorf("Resolve(%q) error = %v, want ErrNoMatch", input, err)
		}
	}
}

// An unreadable conversation is still selectable by prefix — the id is known even
// when the document is not, and refusing would strand the user.
func TestResolveMatchesADamagedConversation(t *testing.T) {
	repo, paths := newTestRepository(t)

	damagedID := id("BRKN")
	dir := filepath.Join(paths.ConversationsDir(), damagedID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("preparing the workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, MetadataFile), []byte("{trunc"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	resolved, err := repo.Resolve("BRKN")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved != damagedID {
		t.Errorf("Resolve() = %q, want %q", resolved, damagedID)
	}

	// Selecting it must still fail, because the document cannot be loaded.
	if err := repo.SetActive(damagedID); err == nil {
		t.Error("SetActive succeeded on a damaged conversation")
	}
}

func TestActiveIDRejectsADamagedSelectionFile(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		wantErr  error
	}{
		{"not json", "{nope", ErrMalformed},
		{"not an id", `{"schema_version":1,"conversation_id":"whatever"}`, ErrInvalid},
		{"newer schema", `{"schema_version":99,"conversation_id":"` + id("A") + `"}`, ErrUnsupportedSchema},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, paths := newTestRepository(t)
			if err := os.MkdirAll(paths.Root, 0o755); err != nil {
				t.Fatalf("preparing the data root: %v", err)
			}
			path := filepath.Join(paths.Root, SelectionFile)
			if err := os.WriteFile(path, []byte(test.contents), 0o644); err != nil {
				t.Fatalf("writing the fixture: %v", err)
			}

			_, err := repo.ActiveID()
			if !errors.Is(err, test.wantErr) {
				t.Errorf("error = %v, want %v", err, test.wantErr)
			}

			// Reporting damage must not modify it.
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("reading back: %v", readErr)
			}
			if string(after) != test.contents {
				t.Error("ActiveID modified the selection file")
			}
		})
	}
}

func TestSelectionDocumentShape(t *testing.T) {
	repo, paths := newTestRepository(t)
	created := createWithID(t, repo, id("AAHA"), "Aaha")

	if err := repo.SetActive(created.ID); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}

	contents, err := os.ReadFile(filepath.Join(paths.Root, SelectionFile))
	if err != nil {
		t.Fatalf("reading the selection: %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatalf("selection is not valid JSON: %v", err)
	}
	for _, field := range []string{"schema_version", "conversation_id", "selected_at"} {
		if _, ok := document[field]; !ok {
			t.Errorf("selection is missing %q", field)
		}
	}
}

// Reselecting must replace rather than accumulate.
func TestSetActiveReplacesThePreviousSelection(t *testing.T) {
	repo, _ := newTestRepository(t)
	aaha := createWithID(t, repo, id("AAHA"), "Aaha")
	beta := createWithID(t, repo, id("BETA"), "Beta")

	for _, want := range []string{aaha.ID, beta.ID, aaha.ID} {
		if err := repo.SetActive(want); err != nil {
			t.Fatalf("SetActive(%q) error = %v", want, err)
		}
		active, err := repo.ActiveID()
		if err != nil {
			t.Fatalf("ActiveID() error = %v", err)
		}
		if active != want {
			t.Errorf("ActiveID() = %q, want %q", active, want)
		}
	}
}
