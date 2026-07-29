package conversation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwogrady/echo/internal/config"
)

// newTestRepository returns a repository over a temporary data root with a fixed
// clock, so listing order is asserted against chosen timestamps rather than
// whatever the wall clock happened to be.
func newTestRepository(t *testing.T) (*Repository, config.Paths) {
	t.Helper()

	paths := config.Paths{Root: t.TempDir()}
	repo := NewRepository(paths)

	moment := time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)
	repo.now = func() time.Time {
		moment = moment.Add(time.Minute)
		return moment
	}

	return repo, paths
}

func TestCreateWritesAUsableConversation(t *testing.T) {
	repo, paths := newTestRepository(t)

	created, err := repo.Create("Product Strategy")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !ValidID(created.ID) {
		t.Errorf("ID = %q, which is not a valid conversation id", created.ID)
	}
	if created.Title != "Product Strategy" {
		t.Errorf("Title = %q", created.Title)
	}
	if created.Slug != "product-strategy" {
		t.Errorf("Slug = %q", created.Slug)
	}
	if created.Status != StatusCreated {
		t.Errorf("Status = %q, want %q", created.Status, StatusCreated)
	}

	if want := filepath.Join(paths.ConversationsDir(), created.ID); repo.Workspace(created.ID).Dir != want {
		t.Errorf("workspace = %q, want %q", repo.Workspace(created.ID).Dir, want)
	}

	loaded, err := repo.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded != created {
		t.Errorf("Get returned a different document:\n got %+v\nwant %+v", loaded, created)
	}
}

func TestCreateTrimsTheTitle(t *testing.T) {
	repo, _ := newTestRepository(t)

	created, err := repo.Create("   Q3 planning   ")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Title != "Q3 planning" {
		t.Errorf("Title = %q, want it trimmed", created.Title)
	}
}

func TestCreateRejectsAnEmptyTitle(t *testing.T) {
	repo, _ := newTestRepository(t)

	for _, title := range []string{"", "   ", "\t\n"} {
		if _, err := repo.Create(title); err == nil {
			t.Errorf("Create(%q) succeeded, want an error", title)
		}
	}
}

// A title is a label, not a key.
func TestDuplicateTitlesGetDistinctConversations(t *testing.T) {
	repo, _ := newTestRepository(t)

	first, err := repo.Create("Product Strategy")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := repo.Create("Product Strategy")
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}

	if first.ID == second.ID {
		t.Fatal("two conversations share an id")
	}
	if repo.Workspace(first.ID).Dir == repo.Workspace(second.ID).Dir {
		t.Error("two conversations share a directory")
	}
	if first.Slug != second.Slug {
		t.Error("slugs should match; they are labels, not identifiers")
	}

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("List() returned %d entries, want 2", len(entries))
	}
}

// A create that fails must not leave a directory that list would report as real.
func TestFailedCreateLeavesNoWorkspaceBehind(t *testing.T) {
	repo, paths := newTestRepository(t)

	// An ID generator that yields a name already taken by a plain file makes
	// workspace creation fail after the ID is chosen.
	if err := os.MkdirAll(paths.ConversationsDir(), 0o755); err != nil {
		t.Fatalf("preparing the data root: %v", err)
	}
	taken := IDPrefix + strings.Repeat("A", idBodyLength)
	if err := os.WriteFile(filepath.Join(paths.ConversationsDir(), taken), []byte("x"), 0o644); err != nil {
		t.Fatalf("preparing the collision: %v", err)
	}
	repo.newID = func() (string, error) { return taken, nil }

	if _, err := repo.Create("Doomed"); err == nil {
		t.Fatal("expected an error")
	}

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List() returned %d entries after a failed create, want 0", len(entries))
	}
}

func TestCreatePropagatesAnIDFailure(t *testing.T) {
	repo, _ := newTestRepository(t)
	repo.newID = func() (string, error) { return "", errors.New("no entropy") }

	if _, err := repo.Create("Anything"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestListIsEmptyBeforeAnythingExists(t *testing.T) {
	repo, _ := newTestRepository(t)

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List() returned %d entries, want 0", len(entries))
	}
}

// The same data must always render in the same order.
func TestListOrdersByUpdatedThenID(t *testing.T) {
	repo, _ := newTestRepository(t)

	var created []Conversation
	for _, title := range []string{"First", "Second", "Third"} {
		conversation, err := repo.Create(title)
		if err != nil {
			t.Fatalf("Create(%q) error = %v", title, err)
		}
		created = append(created, conversation)
	}

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// The fixed clock advances one minute per create, so newest-first is the
	// reverse of creation order.
	want := []string{created[2].ID, created[1].ID, created[0].ID}
	for i, id := range want {
		if entries[i].ID != id {
			t.Errorf("entries[%d].ID = %q, want %q", i, entries[i].ID, id)
		}
	}

	// Repeated calls must not vary.
	again, err := repo.List()
	if err != nil {
		t.Fatalf("second List() error = %v", err)
	}
	for i := range entries {
		if entries[i].ID != again[i].ID {
			t.Fatalf("List() is not deterministic at index %d", i)
		}
	}
}

// Identical timestamps must still produce a total order.
func TestListBreaksTimestampTiesByID(t *testing.T) {
	repo, _ := newTestRepository(t)
	fixed := time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)
	repo.now = func() time.Time { return fixed }

	for _, title := range []string{"A", "B", "C", "D"} {
		if _, err := repo.Create(title); err != nil {
			t.Fatalf("Create(%q) error = %v", title, err)
		}
	}

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	for i := 1; i < len(entries); i++ {
		if entries[i-1].ID > entries[i].ID {
			t.Errorf("entries are not ID-ordered on a timestamp tie: %q before %q",
				entries[i-1].ID, entries[i].ID)
		}
	}
}

// One damaged conversation must not hide the others.
func TestListSurvivesADamagedConversation(t *testing.T) {
	repo, paths := newTestRepository(t)

	healthy, err := repo.Create("Healthy")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	damagedID := IDPrefix + strings.Repeat("Z", idBodyLength)
	damagedDir := filepath.Join(paths.ConversationsDir(), damagedID)
	if err := os.MkdirAll(damagedDir, 0o755); err != nil {
		t.Fatalf("preparing the damaged workspace: %v", err)
	}
	damaged := []byte(`{"schema_version": 1, "id": "trunc`)
	if err := os.WriteFile(filepath.Join(damagedDir, MetadataFile), damaged, 0o644); err != nil {
		t.Fatalf("writing the damaged fixture: %v", err)
	}

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List() returned %d entries, want 2", len(entries))
	}

	var sawHealthy, sawDamaged bool
	for _, entry := range entries {
		switch entry.ID {
		case healthy.ID:
			sawHealthy = true
			if !entry.Readable() {
				t.Error("the healthy conversation reports unreadable")
			}
		case damagedID:
			sawDamaged = true
			if entry.Readable() {
				t.Error("the damaged conversation reports readable")
			}
		}
	}
	if !sawHealthy || !sawDamaged {
		t.Errorf("expected both entries; healthy=%v damaged=%v", sawHealthy, sawDamaged)
	}

	// Readable entries sort ahead of unreadable ones.
	if !entries[0].Readable() {
		t.Error("an unreadable entry sorted ahead of a readable one")
	}

	// Listing must not have touched the damaged file.
	after, err := os.ReadFile(filepath.Join(damagedDir, MetadataFile))
	if err != nil {
		t.Fatalf("reading the damaged fixture back: %v", err)
	}
	if string(after) != string(damaged) {
		t.Error("List modified the damaged file")
	}
}

// Stray files and foreign directories are not conversations.
func TestListIgnoresEntriesThatAreNotConversations(t *testing.T) {
	repo, paths := newTestRepository(t)

	if _, err := repo.Create("Real"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	dir := paths.ConversationsDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing the stray file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "some-other-thing"), 0o755); err != nil {
		t.Fatalf("creating the foreign directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cnv_TOOSHORT"), 0o755); err != nil {
		t.Fatalf("creating the malformed id directory: %v", err)
	}

	entries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("List() returned %d entries, want 1", len(entries))
	}
}

func TestGetRejectsAnythingThatIsNotAnID(t *testing.T) {
	repo, _ := newTestRepository(t)

	// Path traversal matters here: IDs become directory names.
	for _, candidate := range []string{
		"", "cnv_", "cnv_short", "nope", "../../etc/passwd",
		"cnv_../../etc", IDPrefix + strings.Repeat("A", idBodyLength-1),
	} {
		if _, err := repo.Get(candidate); !errors.Is(err, ErrInvalid) {
			t.Errorf("Get(%q) error = %v, want ErrInvalid", candidate, err)
		}
	}
}

func TestGetReportsAMissingConversation(t *testing.T) {
	repo, _ := newTestRepository(t)

	absent := IDPrefix + strings.Repeat("7", idBodyLength)
	if _, err := repo.Get(absent); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestCreatedDocumentIsValidJSONOnDisk(t *testing.T) {
	repo, _ := newTestRepository(t)

	created, err := repo.Create("Product Strategy")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	contents, err := os.ReadFile(repo.Workspace(created.ID).MetadataPath())
	if err != nil {
		t.Fatalf("reading metadata: %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if document["id"] != created.ID {
		t.Errorf("on-disk id = %v, want %q", document["id"], created.ID)
	}
	if !strings.HasSuffix(string(contents), "\n") {
		t.Error("metadata should end with a newline")
	}
}
